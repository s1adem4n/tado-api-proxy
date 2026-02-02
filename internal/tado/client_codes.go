package tado

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

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
