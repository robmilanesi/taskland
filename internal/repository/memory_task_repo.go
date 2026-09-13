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
	// lists resolves list membership: a task is visible to userID only if
	// userID belongs to the list it is in.
	lists ListRepository
}

func newInMemoryTaskRepo(lists ListRepository) *inMemoryTaskRepository {
	return &inMemoryTaskRepository{
		tasks: map[string]models.Task{},
		lists: lists,
	}
}

// myListIDs returns the set of list ids userID belongs to, for filtering a
// full scan of r.tasks in GetAll/Count.
func (r *inMemoryTaskRepository) myListIDs(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	lists, err := r.lists.GetAllLists(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make(map[uuid.UUID]bool, len(lists))
	for _, list := range lists {
		ids[list.ID] = true
	}
	return ids, nil
}

func (r *inMemoryTaskRepository) GetByID(ctx context.Context, userID uuid.UUID, id string) (models.Task, error) {
	if err := ctx.Err(); err != nil {
		return models.Task{}, err
	}

	task, ok := r.tasks[id]
	if !ok {
		return models.Task{}, newErrTaskNotFound(id)
	}

	member, err := r.lists.IsMember(ctx, userID, task.ListID)
	if err != nil {
		return models.Task{}, err
	}
	if !member {
		return models.Task{}, newErrTaskNotFound(id)
	}
	return task, nil
}

func (r *inMemoryTaskRepository) GetAll(ctx context.Context, userID uuid.UUID, params ListTasksParams) ([]models.Task, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}

	myLists, err := r.myListIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	taskList := []models.Task{}
	for task := range maps.Values(r.tasks) {
		if myLists[task.ListID] {
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

func (r *inMemoryTaskRepository) Count(ctx context.Context, userID uuid.UUID, _ ListTasksParams) (int, error) {
	myLists, err := r.myListIDs(ctx, userID)
	if err != nil {
		return 0, err
	}

	n := 0
	for task := range maps.Values(r.tasks) {
		if myLists[task.ListID] {
			n++
		}
	}
	return n, nil
}

func (r *inMemoryTaskRepository) Create(ctx context.Context, userID uuid.UUID, task models.Task) (models.Task, error) {
	if err := ctx.Err(); err != nil {
		return models.Task{}, err
	}

	member, err := r.lists.IsMember(ctx, userID, task.ListID)
	if err != nil {
		return models.Task{}, err
	}
	if !member {
		return models.Task{}, newErrTaskNotFound(task.ListID.String())
	}

	task.ID = uuid.New()
	task.OwnerID = userID
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	r.tasks[task.ID.String()] = task
	return task, nil
}

func (r *inMemoryTaskRepository) Update(ctx context.Context, userID uuid.UUID, task models.Task) (models.Task, error) {
	saved, err := r.GetByID(ctx, userID, task.ID.String())
	if err != nil {
		return models.Task{}, err
	}

	if task.ListID != saved.ListID {
		member, err := r.lists.IsMember(ctx, userID, task.ListID)
		if err != nil {
			return models.Task{}, err
		}
		if !member {
			return models.Task{}, newErrTaskNotFound(task.ListID.String())
		}
	}

	saved.ListID = task.ListID
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

func (r *inMemoryTaskRepository) Delete(ctx context.Context, userID uuid.UUID, id string) (models.Task, error) {
	task, err := r.GetByID(ctx, userID, id)
	if err != nil {
		return models.Task{}, err
	}
	delete(r.tasks, id)
	return task, nil
}
