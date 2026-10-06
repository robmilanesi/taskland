package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robmilanesi/taskland/server/internal/syncer"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	ctx := t.Context()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = s.RunMigrations(ctx)
	if err != nil {
		t.Fatalf("error applying migrations: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})
	return s
}

func insertTask(t *testing.T, s *SQLiteStore, id string, title string, updatedAt int64) {
	t.Helper()
	_, err := s.db.Exec("INSERT INTO tasks (id, title, priority, project_id, created_at_ms, updated_at_ms) VALUES (?, ?, 0, '00000000-0000-0000-0000-000000000000', 0, ?)", id, title, updatedAt)
	if err != nil {
		t.Fatalf("error task fixture: %v", err)
	}
}

func insertVersion(t *testing.T, s *SQLiteStore, taskID, field string, v *syncer.FieldVersion) {
	t.Helper()
	var sequence int64
	err := s.db.QueryRow("UPDATE field_version_sequence set value = value + 1 RETURNING value").Scan(&sequence)
	if err != nil {
		t.Fatalf("fixture sequence: %v", err)
	}

	_, err = s.db.Exec("INSERT INTO task_field_versions (task_id, field, time_ms, device_id, sequence_id) VALUES (?, ?, ?, ?, ?)", taskID, field, v.TimeMS, v.DeviceID, sequence)
	if err != nil {
		t.Fatalf("error task version fixture: %v", err)
	}
}

func readTitle(t *testing.T, s *SQLiteStore, id string) string {
	t.Helper()
	var title string
	err := s.db.QueryRow("SELECT title FROM tasks WHERE id = ?", id).Scan(&title)
	if err != nil {
		t.Fatalf("fixture read title: %v", err)
	}
	return title
}

func readUpdatedAt(t *testing.T, s *SQLiteStore, id string) int64 {
	t.Helper()
	var updatedAt int64
	err := s.db.QueryRow("SELECT updated_at_ms FROM tasks WHERE id = ?", id).Scan(&updatedAt)
	if err != nil {
		t.Fatalf("fixture read title: %v", err)
	}
	return updatedAt
}

func readSequence(t *testing.T, s *SQLiteStore) int64 {
	t.Helper()
	var sequence int64
	err := s.db.QueryRow("SELECT value from field_version_sequence").Scan(&sequence)
	if err != nil {
		t.Fatalf("fixture read sequence: %v", err)
	}
	return sequence
}

func readTaskVersion(t *testing.T, s *SQLiteStore, taskID, field string) (*syncer.FieldVersion, int64) {
	t.Helper()
	var version syncer.FieldVersion
	var sequenceID int64
	err := s.db.QueryRow("SELECT time_ms, device_id, sequence_id from task_field_versions where task_id = ? AND field = ?", taskID, field).Scan(&version.TimeMS, &version.DeviceID, &sequenceID)
	if err != nil {
		t.Fatalf("fixture read task version: %v", err)
	}
	return &version, sequenceID
}

func TestInboxProject(t *testing.T) {
	expectedID := "00000000-0000-0000-0000-000000000000"
	expectedTitle := "Inbox"

	s := newTestStore(t)
	row := s.db.QueryRowContext(t.Context(), "SELECT id, title FROM projects where id = ?", expectedID)

	var id, title string

	err := row.Scan(&id, &title)

	if err != nil {
		t.Fatalf("expected one row find none: %v", err)
	}

	if id != expectedID {
		t.Fatalf("found a different id. expected :%s, got: %s", expectedID, id)
	}

	if title != expectedTitle {
		t.Fatalf("found a different title. expected :%s, got: %s", expectedTitle, title)
	}
}

func TestPragmas(t *testing.T) {
	ctx := t.Context()

	s := newTestStore(t)

	testCases := []struct {
		key      string
		expected string
	}{
		{
			key:      "foreign_keys",
			expected: "1",
		},
		{
			key:      "journal_mode",
			expected: "wal",
		},
		{
			key:      "busy_timeout",
			expected: "5000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			row := s.db.QueryRowContext(ctx, fmt.Sprintf("PRAGMA %s", tc.key))

			var result string

			err := row.Scan(&result)

			if err != nil {
				t.Fatalf("error retrieving pragma: %v", err)
			}

			if result != tc.expected {
				t.Fatalf("expected %s to be %s, got %s", tc.key, tc.expected, result)
			}
		})
	}
}

func TestSyncChange(t *testing.T) {
	fixtureTaskID := strings.Repeat("1", 36)
	fixtureField := "title"

	cases := []struct {
		name             string
		existingVersion  *syncer.FieldVersion
		incoming         syncer.EntityChange
		expectedTitle    string
		expectedSequence int64
		expectedError    error
		expectedVersion  syncer.FieldVersion
	}{
		{
			name:            "no version",
			existingVersion: nil,
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "task",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 0, DeviceID: "phone"},
			},
			expectedTitle:    "new",
			expectedSequence: 1,
			expectedError:    nil,
			expectedVersion:  syncer.FieldVersion{TimeMS: 0, DeviceID: "phone"},
		},
		{
			name:            "newer",
			existingVersion: &syncer.FieldVersion{TimeMS: 0, DeviceID: "desktop"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "task",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 1, DeviceID: "phone"},
			},
			expectedTitle:    "new",
			expectedSequence: 2,
			expectedError:    nil,
			expectedVersion:  syncer.FieldVersion{TimeMS: 1, DeviceID: "phone"},
		},
		{
			name:            "older",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "desktop"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "task",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 1, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    nil,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "desktop"},
		},
		{
			name:            "identical",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "task",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    nil,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
		},
		{
			name:            "invalid type",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "invalid",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 3, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    syncer.ErrInvalidEntityType,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
		},
		{
			name:            "invalid field",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: "task",
				Field:      "invalid",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 3, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    syncer.ErrInvalidEntityField,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
		},
		{
			name:            "not existing task",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("2", 36),
				EntityType: "task",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 3, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    ErrTaskNotFound,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
		},
		{
			name:            "project not implemnted",
			existingVersion: &syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
			incoming: syncer.EntityChange{
				EntityID:   strings.Repeat("0", 36),
				EntityType: "project",
				Field:      "title",
				Value:      "new",
				Version:    syncer.FieldVersion{TimeMS: 3, DeviceID: "phone"},
			},
			expectedTitle:    "old",
			expectedSequence: 1,
			expectedError:    ErrProjectNotImplemented,
			expectedVersion:  syncer.FieldVersion{TimeMS: 2, DeviceID: "phone"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			s := newTestStore(t)
			insertTask(t, s, fixtureTaskID, "old", 10)
			if tc.existingVersion != nil {
				insertVersion(t, s, fixtureTaskID, fixtureField, tc.existingVersion)
			}
			err := s.syncChange(ctx, tc.incoming, nil)
			if !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error: \n%v\n got \n%v", tc.expectedError, err)
			}

			title := readTitle(t, s, fixtureTaskID)
			if title != tc.expectedTitle {
				t.Fatalf("expected title: %s, got %s", tc.expectedTitle, title)
			}

			sequence := readSequence(t, s)
			if sequence != tc.expectedSequence {
				t.Fatalf("expected sequence: %d, got %d", tc.expectedSequence, sequence)
			}

			version, sequence := readTaskVersion(t, s, fixtureTaskID, fixtureField)
			if version.DeviceID != tc.expectedVersion.DeviceID {
				t.Fatalf("expected saved device_id: %s, got %s", tc.expectedVersion.DeviceID, version.DeviceID)
			}

			if version.TimeMS != tc.expectedVersion.TimeMS {
				t.Fatalf("expected saved time_ms: %d, got: %d", tc.expectedVersion.TimeMS, version.TimeMS)
			}

			if sequence != tc.expectedSequence {
				t.Fatalf("expected saved sequence: %d, got %d", tc.expectedSequence, sequence)
			}
		})
	}
}

func TestUpdatedAtDontGoBackwards(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	taskID := strings.Repeat("1", 36)
	insertTask(t, s, taskID, "title", 10)
	insertVersion(t, s, taskID, "title", &syncer.FieldVersion{TimeMS: 10, DeviceID: "phone"})
	change := syncer.EntityChange{
		EntityID:   taskID,
		EntityType: "task",
		Field:      "priority",
		Value:      2,
		Version:    syncer.FieldVersion{TimeMS: 4, DeviceID: "pc"},
	}
	err := s.syncChange(ctx, change, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	savedUpdatedAt := readUpdatedAt(t, s, taskID)
	if savedUpdatedAt != 10 {
		t.Fatalf("expected update time to be %d, got %d", 10, savedUpdatedAt)
	}
}

func TestPush_Ok(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	insertTask(t, s, strings.Repeat("1", 36), "saved", 2)
	insertTask(t, s, strings.Repeat("2", 36), "dos", 2)
	insertVersion(t, s, strings.Repeat("1", 36), "title", &syncer.FieldVersion{TimeMS: 4, DeviceID: "phone"})

	changes := []syncer.EntityChange{
		{
			EntityID:   strings.Repeat("1", 36),
			EntityType: "task",
			Field:      "title",
			Value:      "updated",
			Version:    syncer.FieldVersion{TimeMS: 5, DeviceID: "PC"},
		},
		{
			EntityID:   strings.Repeat("2", 36),
			EntityType: "task",
			Field:      "title",
			Value:      "changed",
			Version:    syncer.FieldVersion{TimeMS: 5, DeviceID: "Test"},
		},
	}
	issues, err := s.PushChanges(ctx, changes)

	if err != nil {
		t.Fatalf("unhandled error occurred: %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got: %v", issues)
	}

	for _, c := range changes {
		savedTitle := readTitle(t, s, c.EntityID)
		if savedTitle != c.Value {
			t.Fatalf("expected saved title to be: %s, got: %s", c.Value, savedTitle)
		}
	}

}

func TestPush_RollbackAllChanges(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	insertTask(t, s, strings.Repeat("1", 36), "saved", 2)
	insertVersion(t, s, strings.Repeat("1", 36), "title", &syncer.FieldVersion{TimeMS: 4, DeviceID: "phone"})

	changes := []syncer.EntityChange{
		{
			EntityID:   strings.Repeat("1", 36),
			EntityType: "task",
			Field:      "title",
			Value:      "updated",
			Version:    syncer.FieldVersion{TimeMS: 5, DeviceID: "PC"},
		},
		{
			EntityID:   strings.Repeat("2", 36),
			EntityType: "task",
			Field:      "priority",
			Value:      4,
			Version:    syncer.FieldVersion{TimeMS: 5, DeviceID: "Test"},
		},
		{
			EntityID:   strings.Repeat("1", 36),
			EntityType: "task",
			Field:      "invalid",
			Value:      4,
			Version:    syncer.FieldVersion{TimeMS: 5, DeviceID: "Test"},
		},
	}
	issues, err := s.PushChanges(ctx, changes)

	if err != nil {
		t.Fatalf("unhandled error occurred: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("expected issues to be 2, got: %v", issues)
	}
	savedTitle := readTitle(t, s, changes[0].EntityID)
	if savedTitle != "saved" {
		t.Fatalf("expected saved title to be: saved, got: %s", savedTitle)
	}
}
