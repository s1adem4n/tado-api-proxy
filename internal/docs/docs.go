package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yml
var Spec string

//go:embed docs.html
var Page string

func Register(mux *http.ServeMux) {
	mux.HandleFunc("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(Spec))
	})

	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(Page))
	})
}
