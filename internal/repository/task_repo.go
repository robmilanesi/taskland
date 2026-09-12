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

// ListTasksParams holds the pagination options for listing tasks.
type ListTasksParams struct {
	Page int
	Size int
}

// TaskRepository is the storage abstraction for tasks. Every method is scoped to
// a single owner: rows belonging to other users are invisible, and a lookup for
// one of them is reported as ErrTaskNotFound.
type TaskRepository interface {
	GetByID(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error)
	GetAll(ctx context.Context, ownerID uuid.UUID, params ListTasksParams) ([]models.Task, error)
	Count(ctx context.Context, ownerID uuid.UUID, params ListTasksParams) (int, error)
	Create(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error)
	Update(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error)
	Delete(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error)
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

	closer func() error
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
		return &Store{
			Tasks: newInMemoryTaskRepo(),
			Users: newInMemoryUserRepo(),
			Lists: newInMemoryListRepo(),
		}, nil
	case TaskRepoSQLite:
		db, err := newSQLiteDB(cfg.DSN)
		if err != nil {
			return nil, err
		}
		return &Store{
			Tasks:  newSQLiteTaskRepo(db),
			Users:  newSQLiteUserRepo(db),
			Lists:  newSQLiteListRepo(db),
			closer: db.Close,
		}, nil
	default:
		return nil, fmt.Errorf("unhandled repository type: %d", cfg.Type)
	}
}
