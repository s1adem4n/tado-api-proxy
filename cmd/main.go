package main

import (
	"io/fs"
	"log"
	"log/slog"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/s1adem4n/tado-api-proxy/internal/proxy"
	"github.com/s1adem4n/tado-api-proxy/internal/tado"
	_ "github.com/s1adem4n/tado-api-proxy/migrations"
	"github.com/s1adem4n/tado-api-proxy/web"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: app.IsDev(),
	})

	tadoClient := tado.NewClient(app)
	tadoClient.Register()

	proxyHandler := proxy.NewHandler(app)
	proxyHandler.Register()

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		slog.SetDefault(app.Logger())

		superuserCount, err := app.CountRecords("_superusers")
		if err != nil {
			return err
		}

		if superuserCount == 0 {
			app.Logger().Info("No superusers found, creating one using environment variables")
			collection, err := app.FindCollectionByNameOrId("_superusers")
			if err != nil {
				return err
			}

			superuser := core.NewRecord(collection)
			superuser.SetEmail(os.Getenv("SUPERUSER_EMAIL"))
			superuser.SetPassword(os.Getenv("SUPERUSER_PASSWORD"))

			if err := app.Save(superuser); err != nil {
				return err
			}
		}

		return nil
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		seedClients(app)

		subFS, err := fs.Sub(web.FS, "dist")
		if err != nil {
			return err
		}
		se.Router.Any("/{path...}", apis.Static(subFS, true))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func seedClients(app core.App) {
	clients := []map[string]any{
		{
			"name":        "Official API",
			"clientID":    "1bb50063-6b0c-4d11-bd99-387f4a91cc46",
			"type":        "deviceCode",
			"redirectURI": "https://login.tado.com/oauth2/device",
			"scope":       "offline_access",
		},
		{
			"name":        "Web App",
			"clientID":    "af44f89e-ae86-4ebe-905f-6bf759cf6473",
			"type":        "passwordGrant",
			"redirectURI": "https://app.tado.com",
			"scope":       "home.user offline_access",
		},
		{
			"name":        "Mobile App",
			"clientID":    "eec8b609-9e2d-4403-9336-4f62a475271e",
			"type":        "passwordGrant",
			"redirectURI": "tado://auth/redirect",
			"scope":       "home.user offline_access",
		},
	}

	collection, err := app.FindCollectionByNameOrId("clients")
	if err != nil {
		slog.Error("clients collection not found", "error", err)
		return
	}

	for _, c := range clients {
		existing, err := app.FindFirstRecordByFilter("clients", "clientID = {:clientID}", map[string]any{"clientID": c["clientID"]})
		if err != nil {
			record := core.NewRecord(collection)
			for k, v := range c {
				record.Set(k, v)
			}
			if err := app.Save(record); err != nil {
				slog.Error("failed to seed client", "name", c["name"], "error", err)
			}
		} else {
			// Update if needed
			changed := false
			for k, v := range c {
				if existing.Get(k) != v {
					existing.Set(k, v)
					changed = true
				}
			}
			if changed {
				app.Save(existing)
			}
		}
	}
}
