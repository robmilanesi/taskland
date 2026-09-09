package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

const userColumns = "id, email, password_hash, created_at"

type sqliteUserRepository struct {
	db *sql.DB
}

func newSQLiteUserRepo(db *sql.DB) *sqliteUserRepository {
	return &sqliteUserRepository{db: db}
}

func (r *sqliteUserRepository) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	user.ID = uuid.New()
	user.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users ("+userColumns+") VALUES (?, ?, ?, ?)",
		user.ID.String(), user.Email, user.PasswordHash, formatTime(user.CreatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("insert user: %w", err)
	}
	return user, nil
}

func (r *sqliteUserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE email = ?", email)
	return userFromRow(row, fmt.Sprintf("email %q", email))
}

func (r *sqliteUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id.String())
	return userFromRow(row, fmt.Sprintf("id %s", id))
}

func userFromRow(row rowScanner, who string) (models.User, error) {
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("get user by %s: %w", who, err)
	}
	return user, nil
}

func scanUser(s rowScanner) (models.User, error) {
	var (
		user      models.User
		idStr     string
		createdAt string
	)

	if err := s.Scan(&idStr, &user.Email, &user.PasswordHash, &createdAt); err != nil {
		return models.User{}, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return models.User{}, fmt.Errorf("parse user id %q: %w", idStr, err)
	}
	user.ID = id

	if user.CreatedAt, err = time.Parse(sqlTimeLayout, createdAt); err != nil {
		return models.User{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
	}

	return user, nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// It matches on the driver's error text rather than a code, which is fragile;
// the user repository contract test guards against the text changing.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
