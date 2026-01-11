package auth

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

// AuthProvider is the interface that both BrowserAuth and MobileAuth implement
// It provides a unified way to obtain OAuth tokens regardless of the authentication method
type AuthProvider interface {
	// GetToken obtains a new token using the provider's authentication method
	GetToken(ctx context.Context) (*Token, error)
}

type HandlerConfig struct {
	TokenPath string
	ClientID  string
}

type Handler struct {
	authProvider AuthProvider
	config       *HandlerConfig
	token        *Token
	lock         sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewHandler(authProvider AuthProvider, config *HandlerConfig) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Handler{
		authProvider: authProvider,
		config:       config,
		token:        &Token{},
		lock:         sync.RWMutex{},
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (h *Handler) Init(ctx context.Context) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	err := h.token.Load(h.config.TokenPath)
	if os.IsNotExist(err) {
		h.token = &Token{}
	} else if err != nil {
		log.Printf("INFO: token file seems to be invalid, it will be recreated")
	}

	if !h.token.Valid() {
		log.Print("Token invalid or expired, attempting refresh")
		err := h.token.Refresh(ctx, h.config.ClientID)
		if err != nil {
			log.Print("Token refresh failed, attempting authentication with provider")
			h.token, err = h.authProvider.GetToken(ctx)
			if err != nil {
				return err
			}
		}

		err = h.token.Save(h.config.TokenPath)
		if err != nil {
			return err
		}
	}

	go h.autoRefresh(h.ctx)
	return nil
}

func (h *Handler) Close() error {
	h.cancel()
	return nil
}

func (h *Handler) GetAccessToken() (string, error) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if !h.token.Valid() {
		return "", errors.New("token is invalid")
	}

	return h.token.AccessToken, nil
}

func (h *Handler) refreshToken(ctx context.Context) error {
	token := h.getToken()

	waitDuration := max(time.Until(token.Expiry.Add(-1*time.Minute)), 0)
	log.Printf("Next token refresh in %v", waitDuration.Round(time.Second))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	err := token.Refresh(ctx, h.config.ClientID)
	if err != nil {
		log.Printf("OAuth refresh failed, attempting authentication with provider")
		token, err = h.authProvider.GetToken(ctx)
		if err != nil {
			return err
		}
	}
	h.token = token

	return token.Save(h.config.TokenPath)
}

func (h *Handler) autoRefresh(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := h.refreshToken(ctx)
			if err != nil {
				log.Printf("Token refresh failed: %v", err)
			}
			log.Print("Token refreshed")
		}
	}
}

func (h *Handler) getToken() *Token {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.token
}
