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
//	GET  /                   - the caller's inbox (requires a session cookie)
//	POST /tasks              - add a task to the inbox
//	POST /tasks/{id}/toggle  - flip a task's completed state
//	POST /tasks/{id}/delete  - delete a task
//	GET  /login              - the login form
//	POST /login              - log in, sets the session cookie
//	GET  /register           - the registration form
//	POST /register           - create an account, sets the session cookie
//	POST /logout             - clear the session cookie
//	GET  /static/*           - vendored static assets (htmx, pico.css, ...)
func NewRouter(store *repository.Store, issuer *auth.Issuer) http.Handler {
	h := NewHandler(store.Users, store.Tasks, store.Lists)
	ah := NewAuthHandler(store, issuer)
	requireAuth := RequireAuth(issuer)

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", requireAuth(http.HandlerFunc(h.Home)))
	mux.Handle("POST /tasks", requireAuth(http.HandlerFunc(h.CreateTask)))
	mux.Handle("POST /tasks/{id}/toggle", requireAuth(http.HandlerFunc(h.ToggleTask)))
	mux.Handle("POST /tasks/{id}/delete", requireAuth(http.HandlerFunc(h.DeleteTask)))

	mux.HandleFunc("GET /login", ah.LoginForm)
	mux.HandleFunc("POST /login", ah.Login)
	mux.HandleFunc("GET /register", ah.RegisterForm)
	mux.HandleFunc("POST /register", ah.Register)
	mux.HandleFunc("POST /logout", ah.Logout)

	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	return httpx.Chain(mux, httpx.RequestID, httpx.RequestLogger, httpx.Recover)
}
