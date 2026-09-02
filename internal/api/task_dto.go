package api

import (
	"errors"
	"strings"

	"github.com/robmilanesi/taskland/internal/models"
)

type createTaskRequest struct {
	Title string `json:"title"`
}

func (req createTaskRequest) validate() error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	return nil
}

func (req createTaskRequest) toModel() models.Task {
	return models.Task{
		Title: strings.TrimSpace(req.Title),
	}
}
