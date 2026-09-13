// Package api provides api related functions
package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

// TaskHandler serves the HTTP endpoints for the task resource.
type TaskHandler struct {
	repo  repository.TaskRepository
	lists repository.ListRepository
}

// NewTaskHandler returns a TaskHandler backed by the given repositories. lists
// is used to resolve a task's list when a request omits one, defaulting it to
// the caller's inbox.
func NewTaskHandler(repo repository.TaskRepository, lists repository.ListRepository) *TaskHandler {
	return &TaskHandler{repo: repo, lists: lists}
}

// GetTask handles GET /api/v1/tasks/{id}.
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	task, err := h.repo.GetByID(r.Context(), ownerID, r.PathValue("id"))
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, task)
}

// GetAllTasks handles GET /api/v1/tasks with pagination.
func (h *TaskHandler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	page, size := httpx.ParsePagination(r)
	listParams := repository.ListTasksParams{Page: page, Size: size}

	taskList, err := h.repo.GetAll(r.Context(), ownerID, listParams)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	count, err := h.repo.Count(r.Context(), ownerID, listParams)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.PaginatedResponse[models.Task]{
		Data:       taskList,
		Page:       page,
		Size:       size,
		Total:      count,
		TotalPages: (count + size - 1) / size,
	})
}

// Create handles POST /api/v1/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	var req createTaskRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	listID, ok := h.resolveListID(w, r, ownerID, req.ListID)
	if !ok {
		return
	}

	toCreate := req.toModel()
	toCreate.ListID = listID

	task, err := h.repo.Create(r.Context(), ownerID, toCreate)
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusBadRequest, "list_id: not a member of this list")
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	w.Header().Set("Location", "/api/v1/tasks/"+task.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, task)
}

// resolveListID returns the request's list_id, or the caller's inbox when
// none was given.
func (h *TaskHandler) resolveListID(w http.ResponseWriter, r *http.Request, ownerID uuid.UUID, requested *uuid.UUID) (uuid.UUID, bool) {
	if requested != nil {
		return *requested, true
	}

	inbox, err := h.lists.InboxFor(r.Context(), ownerID)
	if err != nil {
		httpx.WriteISE(w)
		return uuid.Nil, false
	}
	return inbox.ID, true
}

// Delete handles DELETE /api/v1/tasks/{id}.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	_, err := h.repo.Delete(r.Context(), ownerID, r.PathValue("id"))
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Update handles PATCH /api/v1/tasks/{id}.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	var req updateTaskRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := r.PathValue("id")
	task, err := h.repo.GetByID(r.Context(), ownerID, id)
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	req.applyTo(&task)

	updated, err := h.repo.Update(r.Context(), ownerID, task)
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusBadRequest, "list_id: not a member of this list")
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updated)
}
