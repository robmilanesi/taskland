package api

import (
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

type stubTaskRepo struct {
	createFn  func(models.Task) (models.Task, error)
	getByIDFn func(string) (models.Task, error)
	getAllFn  func(repository.ListTasksParams) ([]models.Task, error)
	count     func(repository.ListTasksParams) (int, error)
	deleteFn  func(string) (models.Task, error)
	updateFn  func(models.Task) (models.Task, error)
}

func (s stubTaskRepo) GetByID(t string) (models.Task, error) { return s.getByIDFn(t) }
func (s stubTaskRepo) GetAll(t repository.ListTasksParams) ([]models.Task, error) {
	return s.getAllFn(t)
}
func (s stubTaskRepo) Count(t repository.ListTasksParams) (int, error) { return s.count(t) }
func (s stubTaskRepo) Create(t models.Task) (models.Task, error)       { return s.createFn(t) }
func (s stubTaskRepo) Delete(t string) (models.Task, error)            { return s.deleteFn(t) }
func (s stubTaskRepo) Update(t models.Task) (models.Task, error)       { return s.updateFn(t) }
