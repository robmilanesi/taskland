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

// Config holds the settings needed to build a TaskRepository.
type Config struct {
	Type TaskRepositoryType
	// DSN is the SQLite file path. It is ignored by TaskRepoInMemory.
	DSN string
}

// NewTaskRepository builds a TaskRepository from the given config.
func NewTaskRepository(cfg Config) (TaskRepository, error) {
	switch cfg.Type {
	case TaskRepoInMemory:
		return newInMemoryTaskRepo(), nil
	case TaskRepoDatabase:
		return nil, errors.New("database task repository not yet implemented")
	default:
		return nil, fmt.Errorf("unhandled repository type: %d", cfg.Type)
	}
}
