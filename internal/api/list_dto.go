package api

import (
	"errors"
	"strings"
)

// listRequest is the body for both creating and renaming a list: the only
// editable field is the name, so there is no need for the pointer-field
// partial-update trick used by updateTaskRequest.
type listRequest struct {
	Name string `json:"name"`
}

func (req listRequest) validate() error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

// addMemberRequest is the body for POST /api/v1/lists/{id}/members: members
// are added by email, not by user id, since that is what the owner actually
// knows about the person they are sharing with.
type addMemberRequest struct {
	Email string `json:"email"`
}

func (req addMemberRequest) validate() error {
	if strings.TrimSpace(req.Email) == "" {
		return errors.New("email is required")
	}
	return nil
}
