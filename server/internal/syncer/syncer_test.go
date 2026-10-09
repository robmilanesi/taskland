package syncer

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name       string
		entityType string
		field      string
		value      any
		expected   error
	}{
		{name: "title ok", entityType: "task", field: "title", value: "buy milk", expected: nil},
		{name: "title max length", entityType: "task", field: "title", value: strings.Repeat("a", 140), expected: nil},
		{name: "title too long", entityType: "task", field: "title", value: strings.Repeat("a", 141), expected: ErrTooLongTitle},
		{name: "title multibyte counts chars", entityType: "task", field: "title", value: strings.Repeat("è", 140), expected: nil},
		{name: "title not string", entityType: "task", field: "title", value: 42, expected: ErrInvalidTitle},
		{name: "project title too long", entityType: "project", field: "title", value: strings.Repeat("a", 101), expected: ErrTooLongTitle},
		{name: "description ok", entityType: "task", field: "description", value: "**md**", expected: nil},
		{name: "description null", entityType: "task", field: "description", value: nil, expected: nil},
		{name: "description not string", entityType: "task", field: "description", value: 1, expected: ErrInvalidDescription},
		{name: "priority min", entityType: "task", field: "priority", value: 0, expected: nil},
		{name: "priority max", entityType: "task", field: "priority", value: 3, expected: nil},
		{name: "priority int64", entityType: "task", field: "priority", value: int64(2), expected: nil},
		{name: "priority too high", entityType: "task", field: "priority", value: 4, expected: ErrInvalidPriority},
		{name: "priority negative", entityType: "task", field: "priority", value: -1, expected: ErrInvalidPriority},
		{name: "priority not integer", entityType: "task", field: "priority", value: "high", expected: ErrInvalidPriority},
		{name: "priority null", entityType: "task", field: "priority", value: nil, expected: ErrInvalidPriority},
		{name: "due ok", entityType: "task", field: "due_at_ms", value: int64(1760000000000), expected: nil},
		{name: "due null", entityType: "task", field: "due_at_ms", value: nil, expected: nil},
		{name: "completed not integer", entityType: "task", field: "completed_at_ms", value: "yesterday", expected: ErrInvalidTimestamp},
		{name: "deleted not integer", entityType: "task", field: "deleted_at_ms", value: 1.5, expected: ErrInvalidTimestamp},
		{name: "project id ok", entityType: "task", field: "project_id", value: strings.Repeat("0", 36), expected: nil},
		{name: "project id wrong length", entityType: "task", field: "project_id", value: "inbox", expected: ErrInvalidProjectID},
		{name: "project id null", entityType: "task", field: "project_id", value: nil, expected: ErrInvalidProjectID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			change := EntityChange{
				EntityID:   strings.Repeat("1", 36),
				EntityType: tc.entityType,
				Field:      tc.field,
				Value:      tc.value,
				Version:    FieldVersion{TimeMS: 1, DeviceID: "phone"},
			}
			err := change.Validate()
			if !errors.Is(err, tc.expected) {
				t.Fatalf("expected error: %v, got: %v", tc.expected, err)
			}
		})
	}
}

func TestIsNewerThan(t *testing.T) {
	testCases := []struct {
		name     string
		incoming FieldVersion
		current  FieldVersion
		expected bool
	}{
		{
			name:     "greater incoming time",
			current:  FieldVersion{0, "desktop"},
			incoming: FieldVersion{2, "desktop"},
			expected: true,
		},
		{
			name:     "lesser incoming time",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{1, "desktop"},
			expected: false,
		},
		{
			name:     "same timestamp, greater device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "laptop"},
			expected: true,
		},
		{
			name:     "same timestamp, lesser device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "a"},
			expected: false,
		},
		{
			name:     "same timestamp, same device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "desktop"},
			expected: false,
		},
		{
			name:     "lesser timestamp, greater device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{1, "laptop"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.incoming.IsNewerThan(tc.current)
			if tc.expected != result {
				t.Fatalf("wrong compare result. expected: %t, got %t", tc.expected, result)
			}
		})
	}

}
