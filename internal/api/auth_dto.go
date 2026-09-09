package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const minPasswordLen = 8

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req registerRequest) validate() error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.New("email is required")
	}
	if len(req.Password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	return nil
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type tokenResponse struct {
	Token string `json:"token"`
}
