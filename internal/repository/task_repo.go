// Package repository provides utilities to interact with various repositories
package repository

import (
	"errors"
	"fmt"

	"github.com/robmilanesi/taskland/internal/models"
)

// TaskRepositoryType selects which TaskRepository implementation to build.
type TaskRepositoryType int

// Supported TaskRepository implementations.
const (
	TaskRepoInMemory TaskRepositoryType = iota
	TaskRepoDatabase
)

// ListTasksParams holds the pagination options for listing tasks.
type ListTasksParams struct {
	Page int
	Size int
}

// TaskRepository is the storage abstraction for tasks.
type TaskRepository interface {
	GetByID(string) (models.Task, error)
	GetAll(ListTasksParams) ([]models.Task, error)
	Count(ListTasksParams) (int, error)
	Create(models.Task) (models.Task, error)
	Update(models.Task) (models.Task, error)
	Delete(string) (models.Task, error)
}

// NewTaskRepository builds a TaskRepository of the requested type.
func NewTaskRepository(taskRepoType TaskRepositoryType) (TaskRepository, error) {
	switch taskRepoType {
	case TaskRepoInMemory:
		return newInMemoryTaskRepo(), nil
	case TaskRepoDatabase:
		return nil, errors.New("database task repository not yet implemented")
	default:
		return nil, fmt.Errorf("unhandled repository type: %d", taskRepoType)
	}
}
