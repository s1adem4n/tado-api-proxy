package tado

import (
	"context"

	"github.com/pocketbase/pocketbase/core"
	"github.com/s1adem4n/tado-api-proxy/internal/tokens"
)

// Client handles tado-specific business logic like account creation and device codes.
type Client struct {
	app          core.App
	auth         *Auth
	tokenManager *tokens.Manager
}

// NewClient creates a new Client.
func NewClient(app core.App, auth *Auth, tokenManager *tokens.Manager) *Client {
	return &Client{
		app:          app,
		auth:         auth,
		tokenManager: tokenManager,
	}
}

// Register sets up event hooks for the client.
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

	c.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		err := c.DeleteUnusedCodes()
		if err != nil {
			c.app.Logger().Error("failed to delete unused codes", "error", err)
		}

		return e.Next()
	})
}

// LoadAccountData creates tokens for all password grant clients and fetches account homes.
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
		token, err := c.auth.Authorize(ctx,
			client.GetString("clientID"),
			client.GetString("redirectURI"),
			client.GetString("scope"),
			account.GetString("email"),
			account.GetString("password"),
			client.GetString("platform"),
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

		expiry := tokens.CalculateTokenExpiry(token.ExpiresIn)
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
