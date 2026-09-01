package repository

import (
	"errors"
	"fmt"
)

var ErrTaskNotFound = errors.New("task not found")

func newErrTaskNotFound(taskID string) error {
	return fmt.Errorf("task %q: %w", taskID, ErrTaskNotFound)
}
