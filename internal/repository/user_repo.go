package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// ErrUserNotFound is returned when a user lookup finds no matching record.
var ErrUserNotFound = errors.New("user not found")

// ErrEmailTaken is returned when creating a user with an email that already exists.
var ErrEmailTaken = errors.New("email already registered")

// UserRepository is the storage abstraction for user accounts.
type UserRepository interface {
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error)
}
