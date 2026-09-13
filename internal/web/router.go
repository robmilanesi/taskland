package web

import (
	"net/http"

	"github.com/robmilanesi/taskland/internal/httpx"
)

// NewRouter builds the HTTP router for the server-rendered web UI. It
// registers:
//
//	GET /            - the home page
//	GET /static/*    - vendored static assets (htmx, ...)
func NewRouter() http.Handler {
	h := NewHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.Home)
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	return httpx.Chain(mux, httpx.RequestID, httpx.RequestLogger, httpx.Recover)
}
