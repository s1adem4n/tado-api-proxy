package config

import (
	env "github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddr       string `env:"LISTEN_ADDR" envDefault:":8080"`
	TokenPath        string `env:"TOKEN_PATH" envDefault:"token.json"`
	CookiesPath      string `env:"COOKIES_PATH" envDefault:"cookies.json"`
	Email            string `env:"EMAIL" envDefault:""`
	Password         string `env:"PASSWORD" envDefault:""`
	ChromeExecutable string `env:"CHROME_EXECUTABLE" envDefault:"/usr/bin/chromium"`
	Headless         bool   `env:"HEADLESS" envDefault:"true"`
	ClientID         string `env:"CLIENT_ID" envDefault:"af44f89e-ae86-4ebe-905f-6bf759cf6473"`
}

func New() *Config {
	return &Config{}
}

func Parse() (*Config, error) {
	cfg := New()
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
