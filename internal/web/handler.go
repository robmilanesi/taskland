// Package web serves the server-rendered HTML frontend, as a direct
// consumer of the repository (not an HTTP client of the JSON API).
package web

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
	"github.com/robmilanesi/taskland/internal/web/views"
)

// inboxPageSize caps how many inbox tasks a single page render fetches.
// There is no pagination UI yet - this is just a sane upper bound.
const inboxPageSize = 200

// Handler serves the web UI's HTTP endpoints.
type Handler struct {
	users repository.UserRepository
	tasks repository.TaskRepository
	lists repository.ListRepository
}

// NewHandler returns a Handler backed by users, tasks and lists.
func NewHandler(users repository.UserRepository, tasks repository.TaskRepository, lists repository.ListRepository) *Handler {
	return &Handler{users: users, tasks: tasks, lists: lists}
}

// Home handles GET /: the caller's inbox and its tasks.
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

	tasks, ok := h.inboxTasks(w, r, userID)
	if !ok {
		return
	}

	render(w, r, views.Home(user.Email, tasks))
}

// CreateTask handles POST /tasks: adds a task to the caller's inbox.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	inbox, err := h.lists.InboxFor(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := h.tasks.Create(r.Context(), userID, models.Task{Title: title, ListID: inbox.ID}); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTaskList(w, r, userID)
}

// ToggleTask handles POST /tasks/{id}/toggle: flips a task's completed state.
func (h *Handler) ToggleTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	task, err := h.tasks.GetByID(r.Context(), userID, r.PathValue("id"))
	if errors.Is(err, repository.ErrTaskNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	task.Completed = !task.Completed
	if _, err := h.tasks.Update(r.Context(), userID, task); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTaskList(w, r, userID)
}

// DeleteTask handles POST /tasks/{id}/delete. Deleting an already-gone task
// is treated as success: the htmx button that triggers this just wants the
// task gone from the list, and it already is.
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err := h.tasks.Delete(r.Context(), userID, r.PathValue("id"))
	if err != nil && !errors.Is(err, repository.ErrTaskNotFound) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTaskList(w, r, userID)
}

// renderTaskList re-renders the caller's inbox task list fragment, the
// htmx target of every task mutation above.
func (h *Handler) renderTaskList(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	tasks, ok := h.inboxTasks(w, r, userID)
	if !ok {
		return
	}
	render(w, r, views.TaskList(tasks))
}

// inboxTasks resolves the caller's inbox and returns its tasks, or writes a
// 500 and reports false on failure.
func (h *Handler) inboxTasks(w http.ResponseWriter, r *http.Request, userID uuid.UUID) ([]models.Task, bool) {
	inbox, err := h.lists.InboxFor(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, false
	}

	tasks, err := h.tasks.GetAll(r.Context(), userID, repository.ListTasksParams{Page: 1, Size: inboxPageSize, ListID: &inbox.ID})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, false
	}
	return tasks, true
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
