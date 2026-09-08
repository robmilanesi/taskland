package repository

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies every pending migration from the embedded SQL files to
// db. It is safe to call on every start: already-applied migrations are skipped.
//
// goose configuration (dialect, base-FS, logger) lives in package-level state,
// so this must not be called concurrently.
func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
