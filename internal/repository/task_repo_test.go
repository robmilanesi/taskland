package repository

import (
	"testing"

	"github.com/robmilanesi/taskland/internal/models"
)

func TestNewTaskRepository_InMemory(t *testing.T) {
	repo, err := NewTaskRepository(Config{Type: TaskRepoInMemory})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo == nil {
		t.Fatal("expected a repository, got nil")
	}
}

func TestNewTaskRepository_SQLite(t *testing.T) {
	repo, err := NewTaskRepository(Config{
		Type: TaskRepoSQLite,
		DSN:  t.TempDir() + "/factory.db",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created, err := repo.Create(models.Task{Title: "via factory"})
	if err != nil {
		t.Fatalf("Create through factory repo: %v", err)
	}
	got, err := repo.GetByID(created.ID.String())
	if err != nil {
		t.Fatalf("GetByID through factory repo: %v", err)
	}
	if got.Title != "via factory" {
		t.Errorf("title = %q, want %q", got.Title, "via factory")
	}
}

func TestNewTaskRepository_SQLite_BadPath(t *testing.T) {
	_, err := NewTaskRepository(Config{
		Type: TaskRepoSQLite,
		DSN:  t.TempDir() + "/no-such-dir/db.sqlite",
	})
	if err == nil {
		t.Fatal("expected an error opening a database in a missing directory")
	}
}

func TestNewTaskRepository_UnknownType(t *testing.T) {
	_, err := NewTaskRepository(Config{Type: TaskRepositoryType(99)})
	if err == nil {
		t.Fatal("expected an error for an unknown repository type")
	}
}
