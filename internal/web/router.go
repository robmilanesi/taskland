package web

import (
	"net/http"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/repository"
)

// NewRouter builds the HTTP router for the server-rendered web UI. It
// registers:
//
//	GET  /            - the home page (requires a session cookie)
//	GET  /login       - the login form
//	POST /login       - log in, sets the session cookie
//	GET  /register    - the registration form
//	POST /register    - create an account, sets the session cookie
//	POST /logout      - clear the session cookie
//	GET  /static/*    - vendored static assets (htmx, ...)
func NewRouter(store *repository.Store, issuer *auth.Issuer) http.Handler {
	h := NewHandler(store.Users)
	ah := NewAuthHandler(store, issuer)
	requireAuth := RequireAuth(issuer)

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", requireAuth(http.HandlerFunc(h.Home)))

	mux.HandleFunc("GET /login", ah.LoginForm)
	mux.HandleFunc("POST /login", ah.Login)
	mux.HandleFunc("GET /register", ah.RegisterForm)
	mux.HandleFunc("POST /register", ah.Register)
	mux.HandleFunc("POST /logout", ah.Logout)

	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	return httpx.Chain(mux, httpx.RequestID, httpx.RequestLogger, httpx.Recover)
}
