package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"

	// Registers the "sqlite" driver with database/sql.
	_ "modernc.org/sqlite"
)

// sqlTimeLayout is the on-disk representation for timestamps: UTC, fixed 9-digit
// fractional seconds so that lexicographic ordering matches chronological order.
const sqlTimeLayout = "2006-01-02T15:04:05.000000000Z"

const taskColumns = "id, owner_id, title, description, completed, completed_at, priority, due_date, created_at, updated_at"

// newSQLiteDB opens the SQLite database at dsn, verifies the connection and
// applies pending migrations. The returned handle is shared by every SQLite
// repository and is closed by the owning Store.
func newSQLiteDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}

	return db, nil
}

type sqliteTaskRepository struct {
	db *sql.DB
}

func newSQLiteTaskRepo(db *sql.DB) *sqliteTaskRepository {
	return &sqliteTaskRepository{db: db}
}

func (r *sqliteTaskRepository) GetByID(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+taskColumns+" FROM tasks WHERE id = ? AND owner_id = ?",
		id, ownerID.String(),
	)

	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, newErrTaskNotFound(id)
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("get task %q: %w", id, err)
	}
	return task, nil
}

func (r *sqliteTaskRepository) GetAll(ctx context.Context, ownerID uuid.UUID, params ListTasksParams) ([]models.Task, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	offset := (params.Page - 1) * params.Size

	rows, err := r.db.QueryContext(ctx,
		"SELECT "+taskColumns+" FROM tasks WHERE owner_id = ? ORDER BY created_at LIMIT ? OFFSET ?",
		ownerID.String(), params.Size, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := []models.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}
	return tasks, nil
}

func (r *sqliteTaskRepository) Count(ctx context.Context, ownerID uuid.UUID, _ ListTasksParams) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE owner_id = ?", ownerID.String(),
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return n, nil
}

func (r *sqliteTaskRepository) Create(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error) {
	task.ID = uuid.New()
	task.OwnerID = ownerID
	task.CreatedAt = time.Now().UTC()
	task.UpdatedAt = task.CreatedAt

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO tasks ("+taskColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID.String(), task.OwnerID.String(), task.Title, task.Description, task.Completed,
		formatNullTime(task.CompletedAt), task.Priority, formatNullTime(task.DueDate),
		formatTime(task.CreatedAt), formatTime(task.UpdatedAt),
	)
	if err != nil {
		return models.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return task, nil
}

func (r *sqliteTaskRepository) Update(ctx context.Context, ownerID uuid.UUID, task models.Task) (models.Task, error) {
	saved, err := r.GetByID(ctx, ownerID, task.ID.String())
	if err != nil {
		return models.Task{}, err
	}

	saved.Title = task.Title
	saved.Description = task.Description
	switch {
	case !saved.Completed && task.Completed:
		now := time.Now().UTC()
		saved.Completed = true
		saved.CompletedAt = &now
	case saved.Completed && !task.Completed:
		saved.Completed = false
		saved.CompletedAt = nil
	}
	saved.Priority = task.Priority
	saved.DueDate = task.DueDate
	saved.UpdatedAt = time.Now().UTC()

	_, err = r.db.ExecContext(ctx,
		"UPDATE tasks SET title = ?, description = ?, completed = ?, completed_at = ?, priority = ?, due_date = ?, updated_at = ? WHERE id = ? AND owner_id = ?",
		saved.Title, saved.Description, saved.Completed, formatNullTime(saved.CompletedAt),
		saved.Priority, formatNullTime(saved.DueDate), formatTime(saved.UpdatedAt),
		saved.ID.String(), ownerID.String(),
	)
	if err != nil {
		return models.Task{}, fmt.Errorf("update task %q: %w", task.ID, err)
	}
	return saved, nil
}

func (r *sqliteTaskRepository) Delete(ctx context.Context, ownerID uuid.UUID, id string) (models.Task, error) {
	row := r.db.QueryRowContext(ctx,
		"DELETE FROM tasks WHERE id = ? AND owner_id = ? RETURNING "+taskColumns,
		id, ownerID.String(),
	)

	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, newErrTaskNotFound(id)
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("delete task %q: %w", id, err)
	}
	return task, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(s rowScanner) (models.Task, error) {
	var (
		task        models.Task
		idStr       string
		ownerIDStr  string
		completedAt sql.NullString
		dueDate     sql.NullString
		createdAt   string
		updatedAt   string
	)

	err := s.Scan(
		&idStr, &ownerIDStr, &task.Title, &task.Description, &task.Completed,
		&completedAt, &task.Priority, &dueDate, &createdAt, &updatedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	if task.ID, err = uuid.Parse(idStr); err != nil {
		return models.Task{}, fmt.Errorf("parse task id %q: %w", idStr, err)
	}
	if task.OwnerID, err = uuid.Parse(ownerIDStr); err != nil {
		return models.Task{}, fmt.Errorf("parse owner id %q: %w", ownerIDStr, err)
	}

	if task.CompletedAt, err = parseNullTime(completedAt); err != nil {
		return models.Task{}, fmt.Errorf("parse completed_at: %w", err)
	}
	if task.DueDate, err = parseNullTime(dueDate); err != nil {
		return models.Task{}, fmt.Errorf("parse due_date: %w", err)
	}
	if task.CreatedAt, err = time.Parse(sqlTimeLayout, createdAt); err != nil {
		return models.Task{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
	}
	if task.UpdatedAt, err = time.Parse(sqlTimeLayout, updatedAt); err != nil {
		return models.Task{}, fmt.Errorf("parse updated_at %q: %w", updatedAt, err)
	}

	return task, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(sqlTimeLayout)
}

func formatNullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

func parseNullTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := time.Parse(sqlTimeLayout, ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
