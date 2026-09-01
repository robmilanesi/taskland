package repository

import (
	"github.com/robmilanesi/taskland/internal/models"
)

type inMemoryTaskRepository struct {
	tasks map[string]models.Task
}

func newInMemoryTaskRepo() *inMemoryTaskRepository {
	return &inMemoryTaskRepository{
		tasks: map[string]models.Task{},
	}
}

func (r *inMemoryTaskRepository) GetByID(id string) (models.Task, error) {
	task, ok := r.tasks[id]
	if !ok {
		return task, newErrTaskNotFound(id)
	}
	return task, nil
}
