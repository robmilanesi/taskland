// Package models contains all domain data models
package models

import (
	"time"

	"github.com/google/uuid"
)

// Priority represent the grade of urgency of a task
type Priority int

// Priority enum values
const (
	PriorityNone Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
)

// Task is a single to-do item.
type Task struct {
	ID uuid.UUID `json:"id"`
	// OwnerID is the user who created the task. It no longer controls access
	// (ListID and the list's membership do); it is informational metadata
	// managed by the repository, never accepted from or exposed to API clients.
	OwnerID uuid.UUID `json:"-"`
	// ListID is the list the task belongs to. Every task belongs to exactly one
	// list, and access is governed by membership in that list.
	ListID      uuid.UUID  `json:"list_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
