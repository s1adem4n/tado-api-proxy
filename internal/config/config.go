package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	env "github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddr       string        `env:"LISTEN_ADDR" envDefault:":8080"`
	TokenPath        string        `env:"TOKEN_PATH" envDefault:"token.json"`
	CookiesPath      string        `env:"COOKIES_PATH" envDefault:"cookies.json"`
	Email            string        `env:"EMAIL"`
	Password         string        `env:"PASSWORD"`
	ChromeExecutable string        `env:"CHROME_EXECUTABLE" envDefault:"/usr/bin/chromium"`
	BrowserTimeout   time.Duration `env:"BROWSER_TIMEOUT" envDefault:"5m"`
	Headless         bool          `env:"HEADLESS" envDefault:"true"`
	ClientID         string        `env:"CLIENT_ID" envDefault:"af44f89e-ae86-4ebe-905f-6bf759cf6473"`
	Debug            bool          `env:"DEBUG" envDefault:"false"`
}

func New() *Config {
	return &Config{}
}

func Parse() (*Config, error) {
	cfg := New()
	err := env.ParseWithOptions(cfg, env.Options{
		RequiredIfNoDef: true,
	})
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.validateWriteable(c.CookiesPath); err != nil {
		return err
	}

	if err := c.validateWriteable(c.TokenPath); err != nil {
		return err
	}

	if err := c.validateChromeExecutable(); err != nil {
		return err
	}

	return nil
}

// validateWriteable checks if a file path is writeable
func (c *Config) validateWriteable(path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("%s is not writeable: %w (please ensure correct permissions are set, refer to README if running in Docker)", path, err)
	}
	f.Close()
	return nil
}

// validateChromeExecutable checks if the Chrome executable is valid
func (c *Config) validateChromeExecutable() error {
	stat, err := os.Stat(c.ChromeExecutable)
	if os.IsNotExist(err) {
		return fmt.Errorf("chrome executable %s does not exist", c.ChromeExecutable)
	}
	if err != nil {
		return fmt.Errorf("chrome executable: failed to stat %s: %w", c.ChromeExecutable, err)
	}

	if stat.IsDir() {
		return fmt.Errorf("chrome executable %s is a directory, not an executable", c.ChromeExecutable)
	}

	if (stat.Mode() & 0111) == 0 {
		return fmt.Errorf("chrome executable %s is not executable", c.ChromeExecutable)
	}

	// Verify we can execute Chrome
	cmd := exec.Command(c.ChromeExecutable, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute chrome at %s: %w", c.ChromeExecutable, err)
	}

	log.Printf("Using Chrome: %s", output)

	return nil
}
