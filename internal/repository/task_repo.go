// Package repository provides utilities to interact with various repositories
package repository

import (
	"errors"
	"fmt"

	"github.com/robmilanesi/taskland/internal/models"
)

type TaskRepositoryType int

const (
	TaskRepoInMemory TaskRepositoryType = iota
	TaskRepoDatabase
)

type ListTasksParams struct {
	Page int
	Size int
}

type TaskRepository interface {
	GetByID(string) (models.Task, error)
	GetAll(ListTasksParams) ([]models.Task, error)
	Count(ListTasksParams) (int, error)
}

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
