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

	go h.autoRefresh()
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

func (h *Handler) autoRefresh() {
	for {
		h.lock.RLock()
		if h.token.AccessToken == "" {
			h.lock.RUnlock()
			time.Sleep(30 * time.Second)
			continue
		}
		refreshAt := h.token.Expiry.Add(-5 * time.Minute)
		h.lock.RUnlock()

		waitDuration := time.Until(refreshAt)
		if waitDuration < 0 {
			waitDuration = 0
		}

		select {
		case <-h.ctx.Done():
			return
		case <-time.After(waitDuration):
		}

		h.lock.Lock()
		err := h.token.Refresh(h.ctx)
		if err != nil {
			log.Printf("Auto-refresh failed: %v, attempting browser auth", err)
			token, err := h.browserAuth.GetToken(h.ctx)
			if err != nil {
				log.Printf("Browser auth failed: %v", err)
				h.lock.Unlock()
				time.Sleep(30 * time.Second)
				continue
			}
			h.token = token
		}
		h.token.Save(h.tokenPath)
		h.lock.Unlock()
	}
}

func (h *Handler) Close() error {
	h.cancel()
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.token.AccessToken != "" {
		return h.token.Save(h.tokenPath)
	}
	return nil
}
