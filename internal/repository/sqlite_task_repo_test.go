package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := newSQLiteDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("newSQLiteDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestSQLiteRepo(t *testing.T) *sqliteTaskRepository {
	t.Helper()
	return newSQLiteTaskRepo(newTestDB(t))
}

func TestSQLiteRepo_CreateGetRoundtrip(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	due := time.Date(2030, 1, 2, 15, 4, 5, 123456789, time.UTC)
	created, err := repo.Create(models.Task{
		Title:       "roundtrip",
		Description: "with a due date",
		Priority:    models.PriorityMedium,
		DueDate:     &due,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("expected a generated id")
	}

	got, err := repo.GetByID(created.ID.String())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Title != "roundtrip" || got.Description != "with a due date" {
		t.Errorf("text fields not preserved: %+v", got)
	}
	if got.Priority != models.PriorityMedium {
		t.Errorf("priority = %v, want %v", got.Priority, models.PriorityMedium)
	}
	if got.CompletedAt != nil {
		t.Errorf("expected completed_at nil, got %v", got.CompletedAt)
	}
	if got.DueDate == nil || !got.DueDate.Equal(due) {
		t.Errorf("due date = %v, want %v", got.DueDate, due)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) || !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("timestamps not preserved: got %v/%v want %v/%v",
			got.CreatedAt, got.UpdatedAt, created.CreatedAt, created.UpdatedAt)
	}
}

func TestSQLiteRepo_GetByID_NotFound(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	_, err := repo.GetByID(uuid.NewString())
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestSQLiteRepo_StoresTimestampsAsFixedWidthUTC(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	created, err := repo.Create(models.Task{Title: "t"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var raw string
	if err := repo.db.QueryRow("SELECT created_at FROM tasks WHERE id = ?", created.ID.String()).Scan(&raw); err != nil {
		t.Fatalf("reading raw created_at: %v", err)
	}

	fixedWidthUTC := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{9}Z$`)
	if !fixedWidthUTC.MatchString(raw) {
		t.Errorf("stored timestamp %q is not fixed-width UTC RFC3339", raw)
	}
}

func TestSQLiteRepo_Delete_ReturnsRowThenNotFound(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	created, err := repo.Create(models.Task{Title: "goner"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	deleted, err := repo.Delete(created.ID.String())
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.ID != created.ID || deleted.Title != "goner" {
		t.Errorf("Delete returned %+v, want the created task", deleted)
	}

	if _, err := repo.Delete(created.ID.String()); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("second Delete: expected ErrTaskNotFound, got %v", err)
	}
}
