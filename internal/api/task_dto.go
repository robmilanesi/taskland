package api

import (
	"errors"
	"strings"
	"time"

	"github.com/robmilanesi/taskland/internal/models"
)

type createTaskRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Priority    models.Priority `json:"priority"`
	DueDate     *time.Time      `json:"due_date"`
}

func (req createTaskRequest) validate() error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	if req.Priority < models.PriorityNone || req.Priority > models.PriorityHigh {
		return errors.New("priority out of range")
	}
	return nil
}

func (req createTaskRequest) toModel() models.Task {
	m := models.Task{
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Priority:    req.Priority,
	}
	if req.DueDate != nil {
		due := *req.DueDate
		m.DueDate = &due
	}
	return m
}

type updateTaskRequest struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	Completed   *bool            `json:"completed"`
	Priority    *models.Priority `json:"priority"`
	DueDate     *time.Time       `json:"due_date"`
}

func (dto updateTaskRequest) validate() error {
	if dto.Title != nil && strings.TrimSpace(*dto.Title) == "" {
		return errors.New("title cannot be empty")
	}
	if dto.Priority != nil && (*dto.Priority < models.PriorityNone || *dto.Priority > models.PriorityHigh) {
		return errors.New("priority out of range")
	}
	return nil
}

func (dto updateTaskRequest) applyTo(t *models.Task) {
	if dto.Title != nil {
		t.Title = strings.TrimSpace(*dto.Title)
	}

	if dto.Description != nil {
		t.Description = *dto.Description
	}

	if dto.Completed != nil {
		t.Completed = *dto.Completed
	}

	if dto.Priority != nil {
		t.Priority = *dto.Priority
	}

	if dto.DueDate != nil {
		due := *dto.DueDate
		t.DueDate = &due
	}

}
