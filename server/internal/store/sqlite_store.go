// Package store handle database and storage logics
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/robmilanesi/taskland/server/internal/syncer"
	_ "modernc.org/sqlite"
)

const (
	pragmaJournalMode = "_pragma=journal_mode(WAL)"
	pragmaForeignKeys = "_pragma=foreign_keys(1)"
	pragmaBusyTimeout = "_pragma=busy_timeout(5000)"
	txImmediate       = "_txlock=immediate"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

const (
	sqliteDialect   = "sqlite"
	migrationFolder = "migrations"
)

type ChangeIssue struct {
	Change syncer.EntityChange
	Err    error
}

type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) RunMigrations(ctx context.Context) error {
	goose.SetBaseFS(embeddedMigrations)

	if err := goose.SetDialect(sqliteDialect); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	if err := goose.UpContext(ctx, s.db, migrationFolder); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func (s *SQLiteStore) PushChanges(ctx context.Context, changes []syncer.EntityChange) ([]ChangeIssue, error) {
	tx, err := s.db.BeginTx(ctx, nil)

	if err != nil {
		return nil, fmt.Errorf("push changes: %w", err)
	}
	defer tx.Rollback()
	var issues []ChangeIssue
	for _, change := range changes {
		err := s.syncChange(ctx, change, tx)

		switch {
		case err == nil:
			continue
		case errors.Is(err, ErrTaskNotFound),
			errors.Is(err, ErrProjectNotImplemented),
			errors.Is(err, syncer.ErrInvalidEntityField),
			errors.Is(err, syncer.ErrInvalidEntityType),
			errors.Is(err, syncer.ErrInvalidTitle),
			errors.Is(err, syncer.ErrTooLongTitle),
			errors.Is(err, syncer.ErrInvalidDescription),
			errors.Is(err, syncer.ErrInvalidPriority),
			errors.Is(err, syncer.ErrInvalidTimestamp),
			errors.Is(err, syncer.ErrInvalidProjectID):
			issues = append(issues, ChangeIssue{Change: change, Err: err})
		default:
			return nil, fmt.Errorf("push changes sync: %w", err)
		}
	}

	if len(issues) > 0 {
		return issues, nil
	}
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("push change commit: %w", err)
	}
	return nil, nil
}

func (s *SQLiteStore) syncChange(ctx context.Context, incoming syncer.EntityChange, tx *sql.Tx) error {
	if err := incoming.Validate(); err != nil {
		return fmt.Errorf("sync change validate: %w", err)
	}

	if incoming.EntityType == "project" {
		return fmt.Errorf("sync change: %w", ErrProjectNotImplemented)
	}
	var err error
	handleTx := tx == nil

	if handleTx {
		tx, err = s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("sync change open transaction: %w", err)
		}
		defer tx.Rollback()
	}

	var curr syncer.FieldVersion
	row := tx.QueryRowContext(
		ctx,
		"SELECT time_ms, device_id FROM task_field_versions WHERE task_id = ? AND field = ?",
		incoming.EntityID, incoming.Field,
	)

	err = row.Scan(&curr.TimeMS, &curr.DeviceID)

	isErrNotFound := errors.Is(err, sql.ErrNoRows)

	if err != nil && !isErrNotFound {
		return fmt.Errorf("sync check version: %w", err)
	}

	if !isErrNotFound && !incoming.Version.IsNewerThan(curr) {
		return nil
	}
	var nextSequence int64
	err = tx.QueryRowContext(ctx, "UPDATE field_version_sequence SET value = value + 1 RETURNING value").Scan(&nextSequence)

	if err != nil {
		return fmt.Errorf("sync update sequence: %w", err)
	}

	updateFieldQuery := fmt.Sprintf("UPDATE tasks SET %s = ?, updated_at_ms = MAX(?, updated_at_ms) WHERE id = ?", incoming.Field)
	result, err := tx.ExecContext(ctx, updateFieldQuery, incoming.Value, incoming.Version.TimeMS, incoming.EntityID)

	if err != nil {
		return fmt.Errorf("sync update version: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sync update version rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO task_field_versions(device_id, time_ms, sequence_id, task_id, field) VALUES (?, ?, ?, ?, ?) ON CONFLICT (task_id, field) DO UPDATE SET time_ms= excluded.time_ms, device_id = excluded.device_id, sequence_id = excluded.sequence_id",
		incoming.Version.DeviceID, incoming.Version.TimeMS, nextSequence, incoming.EntityID, incoming.Field,
	)

	if err != nil {
		return fmt.Errorf("sync upsert: %w", err)
	}

	if handleTx {
		return tx.Commit()
	}
	return nil
}

func NewSQLiteStore(ctx context.Context, dbPath string) (*SQLiteStore, error) {
	dsn := fmt.Sprintf(
		"%s?%s&%s&%s&%s",
		dbPath,
		pragmaJournalMode,
		pragmaForeignKeys,
		pragmaBusyTimeout,
		txImmediate,
	)

	db, err := sql.Open("sqlite", dsn)

	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("db initial ping: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}
