// Package syncer handle data sync in LWW for offline/online synchs
package syncer

import (
	"slices"
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
	return nil
}
