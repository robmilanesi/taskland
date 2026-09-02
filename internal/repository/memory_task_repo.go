package repository

import (
	"cmp"
	"maps"
	"slices"

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

func (r *inMemoryTaskRepository) GetAll(params ListTasksParams) ([]models.Task, error) {

	if params.Page <= 0 {
		params.Page = 1
	}

	if params.Size <= 0 {
		params.Size = 20
	}

	taskList := slices.Collect(maps.Values(r.tasks))
	slices.SortFunc(taskList, func(a, b models.Task) int {
		return cmp.Compare(a.ID, b.ID)
	})

	offset := (params.Page - 1) * params.Size
	if offset >= len(taskList) {
		return []models.Task{}, nil
	}

	end := offset + params.Size
	if end > len(taskList) {
		end = len(taskList)
	}

	return taskList[offset:end], nil
}

func (r *inMemoryTaskRepository) Count(params ListTasksParams) (int, error) {
	return len(r.tasks), nil
}
