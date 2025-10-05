package auth

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

type Handler struct {
	browserAuth *BrowserAuth
	tokenPath   string
	token       *Token
	lock        sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewHandler(browserAuth *BrowserAuth, tokenPath string) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Handler{
		browserAuth: browserAuth,
		tokenPath:   tokenPath,
		token:       &Token{},
		lock:        sync.RWMutex{},
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (h *Handler) Init(ctx context.Context) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	err := h.token.Load(h.tokenPath)
	if os.IsNotExist(err) {
		h.token = &Token{}
	} else if err != nil {
		return err
	}

	if !h.token.Valid() || h.token.Test(ctx) != nil {
		log.Print("Token invalid, using browser auth")
		h.token, err = h.browserAuth.GetToken(ctx)
		if err != nil {
			return err
		}
		if err := h.token.Save(h.tokenPath); err != nil {
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
	}

	h.lock.Lock()
	defer h.lock.Unlock()
	err := token.Refresh(ctx)
	if err != nil {
		log.Printf("OAuth refresh failed, attempting browser auth: %v", err)
		token, err = h.browserAuth.GetToken(ctx)
		if err != nil {
			return err
		}
	}
	h.token = token

	return token.Save(h.tokenPath)
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
		}
	}
}

func (h *Handler) getToken() *Token {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.token
}
