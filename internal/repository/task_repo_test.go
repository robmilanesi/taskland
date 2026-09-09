package repository

import (
	"context"
	"testing"

	"github.com/robmilanesi/taskland/internal/models"
)

func TestNewStore_InMemory(t *testing.T) {
	store, err := NewStore(Config{Type: TaskRepoInMemory})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Tasks == nil || store.Users == nil {
		t.Fatal("expected both repositories to be set")
	}
	if err := store.Close(); err != nil {
		t.Errorf("Close on in-memory store: %v", err)
	}
}

func TestNewStore_SQLite(t *testing.T) {
	store, err := NewStore(Config{Type: TaskRepoSQLite, DSN: t.TempDir() + "/store.db"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	task, err := store.Tasks.Create(models.Task{Title: "via store"})
	if err != nil {
		t.Fatalf("Create task: %v", err)
	}
	if _, err := store.Tasks.GetByID(task.ID.String()); err != nil {
		t.Fatalf("GetByID task: %v", err)
	}

	ctx := context.Background()
	user, err := store.Users.CreateUser(ctx, models.User{Email: "a@b.c", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := store.Users.GetUserByID(ctx, user.ID); err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
}

func TestNewStore_SQLite_BadPath(t *testing.T) {
	_, err := NewStore(Config{Type: TaskRepoSQLite, DSN: t.TempDir() + "/no-such-dir/db.sqlite"})
	if err == nil {
		t.Fatal("expected an error opening a database in a missing directory")
	}
}

func TestNewStore_UnknownType(t *testing.T) {
	_, err := NewStore(Config{Type: TaskRepositoryType(99)})
	if err == nil {
		t.Fatal("expected an error for an unknown backend")
	}
}
