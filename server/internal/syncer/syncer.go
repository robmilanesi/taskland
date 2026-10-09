// Package syncer handle data sync in LWW for offline/online synchs
package syncer

import (
	"slices"
	"unicode/utf8"
)

type FieldVersion struct {
	TimeMS   int64
	DeviceID string
}

func (fv FieldVersion) IsNewerThan(v FieldVersion) bool {
	if v.TimeMS == fv.TimeMS {
		return v.DeviceID < fv.DeviceID
	}
	return v.TimeMS < fv.TimeMS
}

type EntityChange struct {
	EntityID   string
	EntityType string
	Field      string
	Value      any
	Version    FieldVersion
}

func (ec *EntityChange) Validate() error {
	validTypes := []string{"project", "task"}

	if !slices.Contains(validTypes, ec.EntityType) {
		return ErrInvalidEntityType
	}

	var validFields []string
	switch ec.EntityType {
	case "project":
		validFields = []string{
			"title",
			"deleted_at_ms",
		}
	case "task":
		validFields = []string{
			"title",
			"description",
			"priority",
			"due_at_ms",
			"completed_at_ms",
			"deleted_at_ms",
			"project_id",
		}
	default:
		return ErrInvalidEntityType
	}

	if !slices.Contains(validFields, ec.Field) {
		return ErrInvalidEntityField
	}

	switch ec.Field {
	case "title":
		maxLen := maxTaskTitleLen
		if ec.EntityType == "project" {
			maxLen = maxProjectTitleLen
		}
		return validateTitle(ec.Value, maxLen)
	case "description":
		return validateDescription(ec.Value)
	case "priority":
		return validatePriority(ec.Value)
	case "due_at_ms", "completed_at_ms", "deleted_at_ms":
		return validateTimestamp(ec.Value)
	case "project_id":
		return validateProjectID(ec.Value)
	}

	return nil
}

const (
	maxTaskTitleLen    = 140
	maxProjectTitleLen = 100
	minPriority        = 0
	maxPriority        = 3
	idLen              = 36
)

func validateTitle(titleRaw any, maxLen int) error {
	title, ok := titleRaw.(string)
	if !ok {
		return ErrInvalidTitle
	}
	if utf8.RuneCountInString(title) > maxLen {
		return ErrTooLongTitle
	}
	return nil
}

func validateDescription(descriptionRaw any) error {
	if descriptionRaw == nil {
		return nil
	}
	if _, ok := descriptionRaw.(string); !ok {
		return ErrInvalidDescription
	}
	return nil
}

func validatePriority(priorityRaw any) error {
	priority, ok := asInt64(priorityRaw)
	if !ok {
		return ErrInvalidPriority
	}
	if priority < minPriority || priority > maxPriority {
		return ErrInvalidPriority
	}
	return nil
}

func validateTimestamp(timestampRaw any) error {
	if timestampRaw == nil {
		return nil
	}
	if _, ok := asInt64(timestampRaw); !ok {
		return ErrInvalidTimestamp
	}
	return nil
}

func validateProjectID(projectIDRaw any) error {
	projectID, ok := projectIDRaw.(string)
	if !ok || len(projectID) != idLen {
		return ErrInvalidProjectID
	}
	return nil
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}
