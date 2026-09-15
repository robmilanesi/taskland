package web

import (
	"errors"
	"fmt"
	"strings"
)

// minPasswordLen mirrors the same constant in internal/api: the two DTOs are
// shaped differently (form fields here, JSON there) so aren't worth sharing,
// but the rule itself should stay in sync.
const minPasswordLen = 8

type registerForm struct {
	Email    string
	Password string
}

func (f registerForm) validate() error {
	if strings.TrimSpace(f.Email) == "" {
		return errors.New("email is required")
	}
	if len(f.Password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	return nil
}
