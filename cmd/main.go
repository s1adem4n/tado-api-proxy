package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/s1adem4n/tado-api-proxy/internal/config"
	"github.com/s1adem4n/tado-api-proxy/internal/docs"
	"github.com/s1adem4n/tado-api-proxy/internal/proxy"
	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

func main() {
	godotenv.Load()

	ctx := context.Background()

	config, err := config.Parse()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	browserAuth := auth.NewBrowserAuth(&auth.BrowserAuthConfig{
		ChromeExecutable: config.ChromeExecutable,
		Headless:         config.Headless,
		CookiesPath:      config.CookiesPath,
		ClientID:         config.ClientID,
		Email:            config.Email,
		Password:         config.Password,
	})

	authHandler := auth.NewHandler(browserAuth, &auth.HandlerConfig{
		TokenPath: config.TokenPath,
		ClientID:  config.ClientID,
	})

	log.Print("Loading token before starting server")
	err = authHandler.Init(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize auth handler: %v", err)
	}

	mux := http.NewServeMux()
	docs.Register(mux)
	mux.Handle("/", proxy.NewHandler(authHandler))

	log.Printf("Starting server on %s", config.ListenAddr)
	err = http.ListenAndServe(config.ListenAddr, mux)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
