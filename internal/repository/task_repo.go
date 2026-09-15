// Package repository provides utilities to interact with various repositories
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// TaskRepositoryType selects which TaskRepository implementation to build.
type TaskRepositoryType int

// Supported TaskRepository implementations.
const (
	TaskRepoInMemory TaskRepositoryType = iota
	TaskRepoSQLite
)

// ListTasksParams holds the pagination and filtering options for listing tasks.
type ListTasksParams struct {
	Page int
	Size int
	// ListID restricts results to a single list, when set. Both GetAll and
	// Count honor it; left nil, results span every list userID belongs to.
	ListID *uuid.UUID
}

// TaskRepository is the storage abstraction for tasks. Every method is scoped to
// userID, the user acting on behalf of the request: a task is visible only
// through a list userID is a member of, and a lookup for one it cannot reach is
// reported as ErrTaskNotFound.
//
// Create requires task.ListID to already be a list userID belongs to (the
// caller resolves "no list given" to the user's inbox before calling); Update
// may change ListID to move the task, but only into another list userID
// belongs to. Both report ErrTaskNotFound for an inaccessible list, exactly
// like an inaccessible task.
type TaskRepository interface {
	GetByID(ctx context.Context, userID uuid.UUID, id string) (models.Task, error)
	GetAll(ctx context.Context, userID uuid.UUID, params ListTasksParams) ([]models.Task, error)
	Count(ctx context.Context, userID uuid.UUID, params ListTasksParams) (int, error)
	Create(ctx context.Context, userID uuid.UUID, task models.Task) (models.Task, error)
	Update(ctx context.Context, userID uuid.UUID, task models.Task) (models.Task, error)
	Delete(ctx context.Context, userID uuid.UUID, id string) (models.Task, error)
}

// Config holds the settings needed to build a Store.
type Config struct {
	Type TaskRepositoryType
	// DSN is the SQLite file path. It is ignored by TaskRepoInMemory.
	DSN string
}

// Store bundles the repositories that share a storage backend.
type Store struct {
	Tasks TaskRepository
	Users UserRepository
	Lists ListRepository

	closer          func() error
	registerAccount func(context.Context, models.User) (models.User, error)
}

// Close releases resources held by the store, such as the database handle.
// It is safe to call on an in-memory store.
func (s *Store) Close() error {
	if s.closer == nil {
		return nil
	}
	return s.closer()
}

// NewStore builds the repositories for the configured backend.
func NewStore(cfg Config) (*Store, error) {
	switch cfg.Type {
	case TaskRepoInMemory:
		users := newInMemoryUserRepo()
		lists := newInMemoryListRepo()
		return &Store{
			Tasks:           newInMemoryTaskRepo(lists),
			Users:           users,
			Lists:           lists,
			registerAccount: registerAccountSequential(users, lists),
		}, nil
	case TaskRepoSQLite:
		db, err := newSQLiteDB(cfg.DSN)
		if err != nil {
			return nil, err
		}
		return &Store{
			Tasks: newSQLiteTaskRepo(db),
			Users: newSQLiteUserRepo(db),
			Lists: newSQLiteListRepo(db),
			registerAccount: func(ctx context.Context, user models.User) (models.User, error) {
				return registerAccountTx(ctx, db, user)
			},
			closer: db.Close,
		}, nil
	default:
		return nil, fmt.Errorf("unhandled repository type: %d", cfg.Type)
	}
}
