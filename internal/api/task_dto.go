package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/nldate"
)

type createTaskRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Priority    models.Priority `json:"priority"`
	// DueDate accepts either RFC 3339 or a natural-language expression (see
	// nldate.Parse); the parsed result is cached in dueDate by validate.
	DueDate *string `json:"due_date"`
	dueDate *time.Time
	// ListID is optional: when absent, the handler resolves it to the caller's
	// inbox.
	ListID *uuid.UUID `json:"list_id"`
}

func (req *createTaskRequest) validate() error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	if req.Priority < models.PriorityNone || req.Priority > models.PriorityHigh {
		return errors.New("priority out of range")
	}
	if req.DueDate != nil {
		due, err := nldate.Parse(*req.DueDate, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("due_date: %w", err)
		}
		req.dueDate = &due
	}
	return nil
}

func (req createTaskRequest) toModel() models.Task {
	m := models.Task{
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Priority:    req.Priority,
	}
	if req.dueDate != nil {
		due := *req.dueDate
		m.DueDate = &due
	}
	return m
}

type updateTaskRequest struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	Completed   *bool            `json:"completed"`
	Priority    *models.Priority `json:"priority"`
	// DueDate accepts either RFC 3339 or a natural-language expression (see
	// nldate.Parse); the parsed result is cached in dueDate by validate.
	DueDate *string `json:"due_date"`
	dueDate *time.Time
	// ListID is optional: present, it moves the task to that list.
	ListID *uuid.UUID `json:"list_id"`
}

func (dto *updateTaskRequest) validate() error {
	if dto.Title != nil && strings.TrimSpace(*dto.Title) == "" {
		return errors.New("title cannot be empty")
	}
	if dto.Priority != nil && (*dto.Priority < models.PriorityNone || *dto.Priority > models.PriorityHigh) {
		return errors.New("priority out of range")
	}
	if dto.DueDate != nil {
		due, err := nldate.Parse(*dto.DueDate, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("due_date: %w", err)
		}
		dto.dueDate = &due
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

	if dto.dueDate != nil {
		due := *dto.dueDate
		t.DueDate = &due
	}

	if dto.ListID != nil {
		t.ListID = *dto.ListID
	}
}
