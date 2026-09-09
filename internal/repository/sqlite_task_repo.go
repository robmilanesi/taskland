package repository

import (
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

const taskColumns = "id, title, description, completed, completed_at, priority, due_date, created_at, updated_at"

type sqliteTaskRepository struct {
	db *sql.DB
}

func newSQLiteTaskRepo(dsn string) (*sqliteTaskRepository, error) {
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

	return &sqliteTaskRepository{db: db}, nil
}

// Close releases the underlying database handle.
func (r *sqliteTaskRepository) Close() error {
	return r.db.Close()
}

func (r *sqliteTaskRepository) GetByID(id string) (models.Task, error) {
	row := r.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE id = ?", id)

	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, newErrTaskNotFound(id)
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("get task %q: %w", id, err)
	}
	return task, nil
}

func (r *sqliteTaskRepository) GetAll(params ListTasksParams) ([]models.Task, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	offset := (params.Page - 1) * params.Size

	rows, err := r.db.Query(
		"SELECT "+taskColumns+" FROM tasks ORDER BY created_at LIMIT ? OFFSET ?",
		params.Size, offset,
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

func (r *sqliteTaskRepository) Count(_ ListTasksParams) (int, error) {
	var n int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&n); err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return n, nil
}

func (r *sqliteTaskRepository) Create(task models.Task) (models.Task, error) {
	task.ID = uuid.New()
	task.CreatedAt = time.Now().UTC()
	task.UpdatedAt = task.CreatedAt

	_, err := r.db.Exec(
		"INSERT INTO tasks ("+taskColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID.String(), task.Title, task.Description, task.Completed,
		formatNullTime(task.CompletedAt), task.Priority, formatNullTime(task.DueDate),
		formatTime(task.CreatedAt), formatTime(task.UpdatedAt),
	)
	if err != nil {
		return models.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return task, nil
}

func (r *sqliteTaskRepository) Update(task models.Task) (models.Task, error) {
	saved, err := r.GetByID(task.ID.String())
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

	_, err = r.db.Exec(
		"UPDATE tasks SET title = ?, description = ?, completed = ?, completed_at = ?, priority = ?, due_date = ?, updated_at = ? WHERE id = ?",
		saved.Title, saved.Description, saved.Completed, formatNullTime(saved.CompletedAt),
		saved.Priority, formatNullTime(saved.DueDate), formatTime(saved.UpdatedAt), saved.ID.String(),
	)
	if err != nil {
		return models.Task{}, fmt.Errorf("update task %q: %w", task.ID, err)
	}
	return saved, nil
}

func (r *sqliteTaskRepository) Delete(id string) (models.Task, error) {
	row := r.db.QueryRow("DELETE FROM tasks WHERE id = ? RETURNING "+taskColumns, id)

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
		completedAt sql.NullString
		dueDate     sql.NullString
		createdAt   string
		updatedAt   string
	)

	err := s.Scan(
		&idStr, &task.Title, &task.Description, &task.Completed,
		&completedAt, &task.Priority, &dueDate, &createdAt, &updatedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return models.Task{}, fmt.Errorf("parse task id %q: %w", idStr, err)
	}
	task.ID = id

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
