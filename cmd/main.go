package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/s1adem4n/tado-api-proxy/internal/config"
	"github.com/s1adem4n/tado-api-proxy/internal/docs"
	"github.com/s1adem4n/tado-api-proxy/internal/logger"
	"github.com/s1adem4n/tado-api-proxy/internal/proxy"
	"github.com/s1adem4n/tado-api-proxy/internal/stats"
	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

func main() {
	godotenv.Load()

	ctx := context.Background()

	config, err := config.Parse()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Setup(config.LogLevel)

	// Select auth provider based on configuration
	var authProvider auth.AuthProvider
	switch config.AuthMethod {
	case "browser":
		slog.Info("using browser authentication method")
		authProvider = auth.NewBrowserAuth(&auth.BrowserAuthConfig{
			ChromeExecutable: config.ChromeExecutable,
			Headless:         config.Headless,
			CookiesPath:      config.CookiesPath,
			ClientID:         config.ClientID,
			Email:            config.Email,
			Password:         config.Password,
			Timeout:          config.BrowserTimeout,
		})
	case "mobile":
		slog.Info("using mobile authentication method")
		authProvider = auth.NewMobileAuth(&auth.MobileAuthConfig{
			Email:    config.Email,
			Password: config.Password,
		})
	default:
		slog.Error("invalid auth method", "method", config.AuthMethod)
		os.Exit(1)
	}

	authHandler := auth.NewHandler(authProvider, &auth.HandlerConfig{
		TokenPath: config.TokenPath,
		ClientID:  config.ClientID,
	})

	slog.Info("loading token before starting server")
	err = authHandler.Init(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("browser authentication timed out during initialization, please try increasing the BROWSER_TIMEOUT, or set LOG_LEVEL=debug to investigate further")
			os.Exit(1)
		}
		slog.Error("failed to initialize auth handler", "error", err)
		os.Exit(1)
	}

	statsTracker := stats.NewTracker(ctx)

	mux := http.NewServeMux()
	mux.Handle("/stats", statsTracker)
	docs.Register(mux)
	mux.Handle("/", proxy.NewHandler(authHandler, statsTracker))

	slog.Info("starting server", "addr", config.ListenAddr)
	err = http.ListenAndServe(config.ListenAddr, mux)
	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
