package tado

import (
	"context"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type Client struct {
	app core.App
}

func NewClient(app core.App) *Client {
	return &Client{
		app: app,
	}
}

func (c *Client) Register() {
	c.app.OnRecordCreate("accounts").BindFunc(func(e *core.RecordEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		err = c.LoadAccountData(context.Background(), e.Record)
		if err != nil {
			c.app.Delete(e.Record)
			return err
		}

		return nil
	})

	c.app.OnRecordCreateRequest("codes").BindFunc(func(e *core.RecordRequestEvent) error {
		err := c.CreateCode(e)
		if err != nil {
			return err
		}

		return e.Next()
	})

	c.app.Cron().MustAdd("refresh-tokens", "* * * * *", func() {
		err := c.RefreshExpiredTokens(context.Background())
		if err != nil {
			c.app.Logger().Error("failed to refresh tokens", "error", err)
		}
	})

	c.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		err := c.DeleteUnusedCodes()
		if err != nil {
			c.app.Logger().Error("failed to delete unused codes", "error", err)
		}

		return e.Next()
	})
}

func (c *Client) LoadAccountData(ctx context.Context, account *core.Record) error {
	tokensCollection, err := c.app.FindCollectionByNameOrId("tokens")
	if err != nil {
		return err
	}

	homesCollection, err := c.app.FindCollectionByNameOrId("homes")
	if err != nil {
		return err
	}

	clients, err := c.app.FindRecordsByFilter(
		"clients",
		"type = 'passwordGrant'",
		"", 0, 0,
	)
	if err != nil {
		return err
	}

	var accessToken string
	var lastClientName string

	for _, client := range clients {
		clientName := client.GetString("name")
		token, err := c.Authorize(ctx,
			client.GetString("clientID"),
			client.GetString("redirectURI"),
			client.GetString("scope"),
			account.GetString("email"),
			account.GetString("password"),
			clientName,
		)
		if err != nil {
			return err
		}

		accessToken = token.AccessToken
		lastClientName = clientName

		tokenRecord := core.NewRecord(tokensCollection)
		tokenRecord.Set("account", account.Id)
		tokenRecord.Set("client", client.Id)
		tokenRecord.Set("status", "valid")
		tokenRecord.Set("accessToken", token.AccessToken)
		tokenRecord.Set("refreshToken", token.RefreshToken)

		expiry := CalculateTokenExpiry(token.ExpiresIn)
		tokenRecord.Set("expires", expiry)

		if err := c.app.Save(tokenRecord); err != nil {
			return err
		}
	}

	me, err := c.GetMe(ctx, accessToken, lastClientName)
	if err != nil {
		return err
	}

	var homeIDs []string
	for _, home := range me.Homes {
		homeRecord, err := c.app.FindFirstRecordByData("homes", "tadoID", home.ID)
		if err != nil {
			homeRecord = core.NewRecord(homesCollection)
			homeRecord.Set("tadoID", home.ID)
			homeRecord.Set("name", home.Name)
		}

		if err := c.app.Save(homeRecord); err != nil {
			return err
		}

		homeIDs = append(homeIDs, homeRecord.Id)
	}

	account.Set("homes", homeIDs)
	account.Set("tadoID", me.ID)
	if err := c.app.Save(account); err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateCode(e *core.RecordRequestEvent) error {
	clientRecord, err := c.app.FindRecordById("clients", e.Record.GetString("client"))
	if err != nil {
		return err
	}

	if clientRecord.GetString("type") != "deviceCode" {
		return nil
	}

	deviceAuth, err := c.DeviceAuthorize(
		e.Request.Context(),
		clientRecord.GetString("clientID"),
		clientRecord.GetString("scope"),
	)
	if err != nil {
		return err
	}

	e.Record.Set("status", "pending")
	e.Record.Set("deviceCode", deviceAuth.DeviceCode)
	e.Record.Set("userCode", deviceAuth.UserCode)

	verificationURI := deviceAuth.VerificationURIComplete
	verificationURI += "&client_id=" + clientRecord.GetString("clientID")
	e.Record.Set("verificationURI", verificationURI)

	expires := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)
	e.Record.Set("expires", expires)

	go func() {
		err := c.WaitForDeviceAuthorization(
			context.Background(),
			clientRecord,
			e.Record,
		)
		if err != nil {
			c.app.Logger().Error("device authorization failed", "error", err)
		}
	}()

	return nil
}

func (c *Client) WaitForDeviceAuthorization(
	ctx context.Context,
	clientRecord *core.Record,
	codeRecord *core.Record,
) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithDeadline(ctx, codeRecord.GetDateTime("expires").Time())
	defer cancel()

	tokensCollection, err := c.app.FindCollectionByNameOrId("tokens")
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			token, err := c.ExchangeDeviceCode(
				ctx,
				clientRecord.GetString("clientID"),
				codeRecord.GetString("deviceCode"),
			)
			if err != nil {
				continue
			}

			me, err := c.GetMe(ctx, token.AccessToken, clientRecord.GetString("name"))
			if err != nil {
				return err
			}

			accountRecord, err := c.app.FindFirstRecordByData("accounts", "tadoID", me.ID)
			if err != nil {
				codeRecord.Set("status", "unknownAccount")
				if err := c.app.Save(codeRecord); err != nil {
					return err
				}
				return err
			}

			// Check if a token already exists for this account and client
			tokenRecord, err := c.app.FindFirstRecordByFilter(
				"tokens",
				"account = {:accountID} && client = {:clientID}",
				map[string]any{
					"accountID": accountRecord.Id,
					"clientID":  clientRecord.Id,
				},
			)
			if err != nil {
				// Create a new token if not found
				tokenRecord = core.NewRecord(tokensCollection)
			}

			tokenRecord.Set("account", accountRecord.Id)
			tokenRecord.Set("client", clientRecord.Id)
			tokenRecord.Set("status", "valid")
			tokenRecord.Set("accessToken", token.AccessToken)
			tokenRecord.Set("refreshToken", token.RefreshToken)

			expiry := CalculateTokenExpiry(token.ExpiresIn)
			tokenRecord.Set("expires", expiry)

			if err := c.app.Save(tokenRecord); err != nil {
				return err
			}

			codeRecord.Set("status", "authorized")
			codeRecord.Set("token", tokenRecord.Id)
			if err := c.app.Save(codeRecord); err != nil {
				return err
			}

			return nil
		case <-ctx.Done():
			codeRecord.Set("status", "expired")
			if err := c.app.Save(codeRecord); err != nil {
				return err
			}

			return ctx.Err()
		}
	}
}

func (c *Client) DeleteUnusedCodes() error {
	codes, err := c.app.FindRecordsByFilter(
		"codes",
		"status != 'authorized'",
		"", 0, 0,
		map[string]any{
			"now": time.Now(),
		},
	)
	if err != nil {
		return err
	}

	err = c.app.RunInTransaction(func(txApp core.App) error {
		for _, code := range codes {
			if err := txApp.Delete(code); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RefreshExpiredTokens(ctx context.Context) error {
	tokens, err := c.app.FindAllRecords("tokens")
	if err != nil {
		return err
	}

	for _, tokenRecord := range tokens {
		if tokenRecord.GetString("status") != "valid" {
			client, err := c.app.FindRecordById("clients", tokenRecord.GetString("client"))
			if err != nil {
				return err
			}

			if client.GetString("type") == "passwordGrant" {
				err := c.fixPasswordGrantToken(ctx, tokenRecord, client)
				if err != nil {
					c.app.Logger().Error("failed to fix password grant token", "id", tokenRecord.Id, "error", err)
				}
			} else if client.GetString("type") == "deviceCode" {
				err := c.fixDeviceCodeToken(tokenRecord)
				if err != nil {
					c.app.Logger().Error("failed to fix device code token", "id", tokenRecord.Id, "error", err)
				}
			}
		}

		err := c.refreshToken(ctx, tokenRecord)
		if err != nil {
			tokenRecord.Set("status", "invalid")
			c.app.Save(tokenRecord)
			c.app.Logger().Error("failed to refresh token", "id", tokenRecord.Id, "error", err)
		}
	}

	return nil
}

func (c *Client) fixPasswordGrantToken(ctx context.Context, tokenRecord *core.Record, clientRecord *core.Record) error {
	account, err := c.app.FindRecordById("accounts", tokenRecord.GetString("account"))
	if err != nil {
		return err
	}

	newToken, err := c.Authorize(
		ctx,
		clientRecord.GetString("clientID"),
		clientRecord.GetString("redirectURI"),
		clientRecord.GetString("scope"),
		account.GetString("email"),
		account.GetString("password"),
		clientRecord.GetString("name"),
	)
	if err != nil {
		return err
	}

	tokenRecord.Set("status", "valid")
	tokenRecord.Set("accessToken", newToken.AccessToken)
	tokenRecord.Set("refreshToken", newToken.RefreshToken)
	expiry := CalculateTokenExpiry(newToken.ExpiresIn)
	tokenRecord.Set("expires", expiry)

	if err := c.app.Save(tokenRecord); err != nil {
		return err
	}

	c.app.Logger().Info("fixed password grant token", "id", tokenRecord.Id)
	return nil
}

func (c *Client) fixDeviceCodeToken(tokenRecord *core.Record) error {
	// Check if last used day is before cutoff, if yes mark as valid.
	// This means that the rate-limit has been reset since last use.
	cutoff, err := GetRatelimitCutoff()
	if err != nil {
		return err
	}

	lastUsed := tokenRecord.GetDateTime("lastUsed")
	if time.Now().After(cutoff) && lastUsed.Time().Before(cutoff) {
		tokenRecord.Set("status", "valid")
		if err := c.app.Save(tokenRecord); err != nil {
			return err
		}
		c.app.Logger().Info("enabled token because of rate-limit reset", "id", tokenRecord.Id)
		return nil
	}

	return nil
}

func (c *Client) refreshToken(ctx context.Context, tokenRecord *core.Record) error {
	expires := tokenRecord.GetDateTime("expires")
	bufferedExpiry := expires.Add(-2 * time.Minute)
	if time.Now().Before(bufferedExpiry.Time()) {
		return nil
	}

	clientRecord, err := c.app.FindRecordById("clients", tokenRecord.GetString("client"))
	if err != nil {
		return err
	}

	newToken, err := c.RefreshToken(ctx,
		clientRecord.GetString("clientID"),
		tokenRecord.GetString("refreshToken"),
		clientRecord.GetString("name"),
	)
	if err != nil {
		return err
	}

	tokenRecord.Set("accessToken", newToken.AccessToken)
	tokenRecord.Set("refreshToken", newToken.RefreshToken)
	expiry := CalculateTokenExpiry(newToken.ExpiresIn)
	tokenRecord.Set("expires", expiry)

	if err := c.app.Save(tokenRecord); err != nil {
		return err
	}

	c.app.Logger().Info("refreshed token", "id", tokenRecord.Id)
	return nil
}

func (c *Client) absURL(loc string) string {
	if strings.HasPrefix(loc, "http") {
		return loc
	}
	return "https://login.tado.com" + loc
}
