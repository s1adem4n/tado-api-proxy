package auth

import (
	"context"
	"encoding/json"
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

type BrowserAuth struct {
	chromeExecutable string
	cookiesPath      string
	email            string
	password         string
}

func NewBrowserAuth(
	chromeExecutable,
	cookiesPath,
	email,
	password string,
) *BrowserAuth {
	return &BrowserAuth{
		chromeExecutable: chromeExecutable,
		cookiesPath:      cookiesPath,
		email:            email,
		password:         password,
	}
}

func (b *BrowserAuth) GetToken(ctx context.Context) (*Token, error) {
	launchURL, err := launcher.New().
		Context(ctx).
		Bin(b.chromeExecutable).
		Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(launchURL)
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

	time.Sleep(5 * time.Second) // Wait for potential redirects

	info, err := page.Info()
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(info.URL, LoginURL) {
		emailInput, err := page.Element("#loginId")
		if err != nil {
			return nil, err
		}
		err = emailInput.Input(b.email)
		if err != nil {
			return nil, err
		}

		passwordInput, err := page.Element("#password")
		if err != nil {
			return nil, err
		}
		err = passwordInput.Input(b.password)
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
	err = page.Timeout(10 * time.Second).Wait(&rod.EvalOptions{
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
	err = token.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (b *BrowserAuth) loadCookies() ([]*proto.NetworkCookieParam, error) {
	file, err := os.Open(b.cookiesPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	var cookies []*proto.NetworkCookieParam
	err = json.NewDecoder(file).Decode(&cookies)
	if err != nil {
		return nil, err
	}

	return cookies, nil
}

func (b *BrowserAuth) saveCookies(cookies []*proto.NetworkCookie) error {
	file, err := os.Create(b.cookiesPath)
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
