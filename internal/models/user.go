package models

import (
	"time"

	"github.com/google/uuid"
)

// User is an account that owns tasks.
type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	// PasswordHash is never serialised: the json tag drops it from responses.
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
