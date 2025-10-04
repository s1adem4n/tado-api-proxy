package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/s1adem4n/tado-api-proxy/internal/config"
	"github.com/s1adem4n/tado-api-proxy/internal/proxy"
	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

func main() {
	godotenv.Load()

	ctx := context.Background()

	config, err := config.Parse()
	if err != nil {
		panic(err)
	}

	authHandler := auth.NewHandler(
		auth.NewBrowserAuth(
			config.ChromeExecutable,
			config.CookiesPath,
			config.Email,
			config.Password,
		),
		config.TokenPath,
	)
	err = authHandler.Init(ctx)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxy.NewHandler(authHandler))

	fmt.Printf("Listening on %s\n", config.ListenAddr)
	err = http.ListenAndServe(config.ListenAddr, mux)
	if err != nil {
		panic(err)
	}
}
