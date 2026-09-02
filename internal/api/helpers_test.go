package api

import (
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

type stubTaskRepo struct {
	createFn func(models.Task) (models.Task, error)
}

func (s stubTaskRepo) GetByID(string) (models.Task, error)                      { return models.Task{}, nil }
func (s stubTaskRepo) GetAll(repository.ListTasksParams) ([]models.Task, error) { return nil, nil }
func (s stubTaskRepo) Count(repository.ListTasksParams) (int, error)            { return 0, nil }
func (s stubTaskRepo) Create(t models.Task) (models.Task, error)                { return s.createFn(t) }
