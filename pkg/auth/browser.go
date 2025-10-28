package auth

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

const (
	AppURL   = "https://app.tado.com"
	LoginURL = "https://login.tado.com"
)

type BrowserAuthConfig struct {
	ChromeExecutable string
	Headless         bool
	CookiesPath      string
	ClientID         string
	Email            string
	Password         string
	Timeout          time.Duration
}

type BrowserAuth struct {
	config *BrowserAuthConfig
}

func NewBrowserAuth(config *BrowserAuthConfig) *BrowserAuth {
	return &BrowserAuth{
		config: config,
	}
}

func (b *BrowserAuth) GetToken(ctx context.Context) (*Token, error) {
	ctx, cancel := context.WithTimeout(ctx, b.config.Timeout)
	defer cancel()

	launcher := launcher.New().
		Context(ctx).
		Bin(b.config.ChromeExecutable).
		Headless(b.config.Headless)
	defer launcher.Cleanup()

	launchURL, err := launcher.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(launchURL).Context(ctx)
	defer browser.Close()

	err = browser.Connect()
	if err != nil {
		return nil, err
	}

	cookies, err := b.loadCookies()
	if err != nil {
		return nil, err
	}
	err = browser.SetCookies(cookies)
	if err != nil {
		return nil, err
	}

	page, err := stealth.Page(browser)
	if err != nil {
		return nil, err
	}
	err = page.Navigate(AppURL)
	if err != nil {
		return nil, err
	}

	time.Sleep(5 * time.Second)

	info, err := page.Info()
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(info.URL, LoginURL) {
		emailInput, err := page.Element("#loginId")
		if err != nil {
			return nil, err
		}
		err = emailInput.Input(b.config.Email)
		if err != nil {
			return nil, err
		}

		passwordInput, err := page.Element("#password")
		if err != nil {
			return nil, err
		}
		err = passwordInput.Input(b.config.Password)
		if err != nil {
			return nil, err
		}

		submitButton, err := page.ElementR("button", "Sign in")
		if err != nil {
			return nil, err
		}
		err = submitButton.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			return nil, err
		}
	}

	// Save cookies after login
	newCookies, err := browser.GetCookies()
	if err != nil {
		return nil, err
	}
	err = b.saveCookies(newCookies)
	if err != nil {
		return nil, err
	}

	// wait for token refresh
	err = page.Wait(&rod.EvalOptions{
		JS: `() => {
			const token = JSON.parse(window.localStorage.getItem("ngStorage-token"));
			return token && token.refresh_token && new Date(token.expires_at) > new Date();
		}`,
	})
	if err != nil {
		return nil, err
	}

	refreshTokenObj, err := page.Eval(`() => {
		const tokenData = window.localStorage.getItem("ngStorage-token");
		const token = JSON.parse(tokenData);
		return token.refresh_token;
	}`)
	if err != nil {
		return nil, err
	}
	refreshToken := refreshTokenObj.Value.Str()

	token := NewToken("", refreshToken, 0)
	err = token.Refresh(ctx, b.config.ClientID)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (b *BrowserAuth) loadCookies() ([]*proto.NetworkCookieParam, error) {
	file, err := os.Open(b.config.CookiesPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	var cookies []*proto.NetworkCookieParam
	err = json.NewDecoder(file).Decode(&cookies)
	if err != nil {
		log.Printf("INFO: cookie file seems to be invalid, it will be recreated")
		return nil, nil
	}

	return cookies, nil
}

func (b *BrowserAuth) saveCookies(cookies []*proto.NetworkCookie) error {
	file, err := os.Create(b.config.CookiesPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(cookies)
	if err != nil {
		return err
	}

	return nil
}
