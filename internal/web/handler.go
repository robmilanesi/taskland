// Package web serves the server-rendered HTML frontend, as a direct
// consumer of the repository (not an HTTP client of the JSON API).
package web

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/robmilanesi/taskland/internal/repository"
	"github.com/robmilanesi/taskland/internal/web/views"
)

// Handler serves the web UI's HTTP endpoints.
type Handler struct {
	users repository.UserRepository
}

// NewHandler returns a Handler backed by users.
func NewHandler(users repository.UserRepository) *Handler {
	return &Handler{users: users}
}

// Home handles GET /.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.users.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	render(w, r, views.Home(user.Email))
}

// render writes a templ component to the response, logging (rather than
// reporting to the client) any failure: by the time Render starts writing,
// headers and part of the body may already be flushed, so the status code
// can no longer be changed.
func render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("render template", "error", err, "path", r.URL.Path)
	}
}
