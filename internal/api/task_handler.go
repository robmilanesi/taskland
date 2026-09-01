// Package api provides api related functions
package api

import (
	"errors"
	"net/http"

	"github.com/robmilanesi/taskland/internal/httpx"
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
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, task)
}
