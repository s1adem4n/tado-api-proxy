package proxy

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"net/http"

	"github.com/s1adem4n/tado-api-proxy/pkg/auth"
)

const (
	BaseURL = "https://my.tado.com"
)

type Handler struct {
	authHandler *auth.Handler
}

func NewHandler(authHandler *auth.Handler) *Handler {
	return &Handler{
		authHandler: authHandler,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, err := h.authHandler.GetAccessToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	url := BaseURL + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	bodyCopy, _ := io.ReadAll(r.Body)

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		url,
		bytes.NewReader(bodyCopy),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBodyCopy, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(respBodyCopy))

	fmt.Printf("----REQUEST %s----\n", req.URL.Path)
	for k, v := range req.Header {
		fmt.Printf("%s: %s\n", k, v)
	}

	fmt.Println()
	fmt.Printf("----RESPONSE %s----\n", req.URL.Path)
	for k, v := range resp.Header {
		fmt.Printf("%s: %s\n", k, v)
	}
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Println("Body:")
	fmt.Println(string(respBodyCopy))
	fmt.Println("Request Body:")
	fmt.Println(string(bodyCopy))
	fmt.Println("---------------")
	fmt.Println()

	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
