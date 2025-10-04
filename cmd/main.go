package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/s1adem4n/tado-api-proxy/internal/config"
	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

func main() {
	godotenv.Load()

	ctx := context.Background()

	config, err := config.Parse()
	if err != nil {
		panic(err)
	}

	handler := auth.NewHandler(
		auth.NewBrowserAuth(
			config.ChromeExecutable,
			config.CookiesPath,
			config.Email,
			config.Password,
		),
		config.TokenPath,
	)
	err = handler.Init(ctx)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	fmt.Printf("Listening on %s\n", config.ListenAddr)
	err = http.ListenAndServe(config.ListenAddr, mux)
	if err != nil {
		panic(err)
	}
}
