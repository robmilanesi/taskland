package repository

import (
	"errors"
	"testing"

	"github.com/robmilanesi/taskland/internal/models"
)

func TestInMemoryTaskRepo_GetByID_NotFound(t *testing.T) {
    repo := newInMemoryTaskRepo()

    _, err := repo.GetByID("nonexistent")

    if !errors.Is(err, ErrTaskNotFound) {
        t.Errorf("expected ErrTaskNotFound, got %v", err)
    }
}

func TestInMemoryTaskRepo_GetByID_Found(t *testing.T) {
    repo := newInMemoryTaskRepo()
	repo.tasks["1"] = models.Task{ID: "1", Title: "Test"}

    task, err := repo.GetByID("1")

    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if task.ID != "1" {
        t.Errorf("expected task ID '1', got %q", task.ID)
    }
}