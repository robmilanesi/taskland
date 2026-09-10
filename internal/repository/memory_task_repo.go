package repository

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"

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

func (r *inMemoryTaskRepository) GetByID(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error) {
	if err := ctx.Err(); err != nil {
		return models.Task{}, err
	}

	task, ok := r.tasks[id]
	if !ok || task.OwnerID != ownerID {
		return models.Task{}, newErrTaskNotFound(id)
	}
	return task, nil
}

func (r *inMemoryTaskRepository) GetAll(ctx context.Context, ownerID uuid.UUID, params ListTasksParams) ([]models.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}

	taskList := []models.Task{}
	for task := range maps.Values(r.tasks) {
		if task.OwnerID == ownerID {
			taskList = append(taskList, task)
		}
	}
	slices.SortFunc(taskList, func(a, b models.Task) int {
		return a.CreatedAt.Compare(b.CreatedAt)
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

func (r *inMemoryTaskRepository) Count(ctx context.Context, ownerID uuid.UUID, _ ListTasksParams) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	n := 0
	for _, task := range r.tasks {
		if task.OwnerID == ownerID {
			n++
		}
	}
	return n, nil
}

func (r *inMemoryTaskRepository) Create(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error) {
	if err := ctx.Err(); err != nil {
		return models.Task{}, err
	}

	task.ID = uuid.New()
	task.OwnerID = ownerID
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	r.tasks[task.ID.String()] = task
	return task, nil
}

func (r *inMemoryTaskRepository) Update(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error) {
	saved, err := r.GetByID(ctx, ownerID, task.ID.String())
	if err != nil {
		return models.Task{}, err
	}

	saved.Title = task.Title
	saved.Description = task.Description
	switch {
	case !saved.Completed && task.Completed:
		now := time.Now()
		saved.Completed = true
		saved.CompletedAt = &now
	case saved.Completed && !task.Completed:
		saved.Completed = false
		saved.CompletedAt = nil
	}
	saved.Priority = task.Priority
	saved.DueDate = task.DueDate
	saved.UpdatedAt = time.Now()

	r.tasks[saved.ID.String()] = saved
	return saved, nil
}

func (r *inMemoryTaskRepository) Delete(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error) {
	task, err := r.GetByID(ctx, ownerID, id)
	if err != nil {
		return models.Task{}, err
	}
	delete(r.tasks, id)
	return task, nil
}
