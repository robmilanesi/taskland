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
