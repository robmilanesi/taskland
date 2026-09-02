// Package api provides api related functions
package api

import (
	"errors"
	"net/http"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

type TaskHandler struct {
	repo repository.TaskRepository
}

func NewTaskHandler(repo repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

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
