package repository

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRunMigrations_CreatesTasksTable(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	var n int
	if err := db.QueryRow("SELECT count(*) FROM tasks").Scan(&n); err != nil {
		t.Fatalf("querying tasks after migration: %v", err)
	}
	if n != 0 {
		t.Errorf("expected empty tasks table, got %d rows", n)
	}

	want := []string{
		"id", "title", "description", "completed", "completed_at",
		"priority", "due_date", "created_at", "updated_at",
	}
	rows, err := db.Query("SELECT name FROM pragma_table_info('tasks')")
	if err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan column name: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating columns: %v", err)
	}

	for _, col := range want {
		if !got[col] {
			t.Errorf("missing column %q in tasks table", col)
		}
	}
}

func TestRunMigrations_CreatesUsersTable(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	var n int
	if err := db.QueryRow("SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatalf("querying users after migration: %v", err)
	}
	if n != 0 {
		t.Errorf("expected empty users table, got %d rows", n)
	}
}

func TestRunMigrations_TasksHaveOwnerColumn(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	var n int
	if err := db.QueryRow("SELECT count(*) FROM tasks WHERE owner_id = ''").Scan(&n); err != nil {
		t.Fatalf("querying tasks by owner_id: %v", err)
	}
}

func TestRunMigrations_CreatesListsTables(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	for _, table := range []string{"lists", "list_members"} {
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("querying %s after migration: %v", table, err)
		}
	}
}

func TestRunMigrations_OnlyOneInboxPerOwner(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	owner := "owner-1"
	insertInbox := "INSERT INTO lists (id, owner_id, name, is_inbox, created_at) VALUES (?, ?, 'Inbox', 1, '2030-01-01T00:00:00.000000000Z')"

	if _, err := db.Exec(insertInbox, "list-1", owner); err != nil {
		t.Fatalf("first inbox insert: %v", err)
	}
	if _, err := db.Exec(insertInbox, "list-2", owner); err == nil {
		t.Error("expected a second inbox for the same owner to violate the unique index")
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := openTestDB(t)

	if err := runMigrations(db); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := runMigrations(db); err != nil {
		t.Fatalf("second run should be a no-op: %v", err)
	}
}
