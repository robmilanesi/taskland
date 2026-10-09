package syncer

import (
	"errors"
)

var ErrInvalidEntityType = errors.New("invalid entity type")
var ErrInvalidEntityField = errors.New("invalid entity field")
var ErrTooLongTitle = errors.New("title too long")
var ErrInvalidTitle = errors.New("title must be a string")
var ErrInvalidPriority = errors.New("priority must be an integer between 0 and 3")
var ErrInvalidDescription = errors.New("description must be a string or null")
var ErrInvalidTimestamp = errors.New("timestamp must be an integer or null")
var ErrInvalidProjectID = errors.New("project id must be a 36 char string")
