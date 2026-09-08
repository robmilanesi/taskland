// Package api provides api related functions
package api

import (
	"errors"
	"net/http"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

// TaskHandler serves the HTTP endpoints for the task resource.
type TaskHandler struct {
	repo repository.TaskRepository
}

// NewTaskHandler returns a TaskHandler backed by the given repository.
func NewTaskHandler(repo repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// GetTask handles GET /api/v1/tasks/{id}.
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.repo.GetByID(id)

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
	page, size := httpx.ParsePagination(r)
	listParams := repository.ListTasksParams{Page: page, Size: size}
	taskList, err := h.repo.GetAll(listParams)

	if err != nil {
		httpx.WriteISE(w)
		return
	}

	count, err := h.repo.Count(listParams)

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
	var req createTaskRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.repo.Create(req.toModel())
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	w.Header().Set("Location", "/api/v1/tasks/"+task.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, task)
}

// Delete handles DELETE /api/v1/tasks/{id}.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := h.repo.Delete(id)

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
	var req updateTaskRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := r.PathValue("id")
	task, err := h.repo.GetByID(id)
	if errors.Is(err, repository.ErrTaskNotFound) {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	req.applyTo(&task)

	updated, err := h.repo.Update(task)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updated)
}
