package repository

import (
	"errors"
	"fmt"
)

// ErrTaskNotFound is returned when a task lookup finds no matching record.
var ErrTaskNotFound = errors.New("task not found")

func newErrTaskNotFound(taskID string) error {
	return fmt.Errorf("task %q: %w", taskID, ErrTaskNotFound)
}
