package models

import (
	"time"

	"github.com/google/uuid"
)

// List is a named, shareable grouping of tasks.
type List struct {
	ID      uuid.UUID `json:"id"`
	OwnerID uuid.UUID `json:"owner_id"`
	Name    string    `json:"name"`
	// IsInbox marks the default list created for every user at registration.
	// It cannot be deleted.
	IsInbox   bool      `json:"is_inbox"`
	CreatedAt time.Time `json:"created_at"`
}
