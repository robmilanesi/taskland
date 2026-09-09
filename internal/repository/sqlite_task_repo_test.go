package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

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

// The behavioural surface is covered by testTaskRepositoryContract against both
// implementations. This test pins the SQLite-specific on-disk time format.
func TestSQLiteRepo_StoresTimestampsAsFixedWidthUTC(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	created, err := repo.Create(context.Background(), uuid.New(), models.Task{Title: "t"})
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
