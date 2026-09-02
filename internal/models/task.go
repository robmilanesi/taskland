// Package models contains all domain data models
package models

import (
	"time"

	"github.com/google/uuid"
)

// Task is a single to-do item.
type Task struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
