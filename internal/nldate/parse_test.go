package nldate

import (
	"testing"
	"time"
)

// refTime is a Wednesday, chosen so the weekday-name test cases below cover
// both a day earlier in the week (already passed) and later in the week
// (still to come).
var refTime = time.Date(2024, time.January, 10, 15, 4, 5, 0, time.UTC)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "today",
			input: "today",
			want:  time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			want:  time.Date(2024, time.January, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "case insensitive and trims whitespace",
			input: "  ToMoRRoW  ",
			want:  time.Date(2024, time.January, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "weekday later this week",
			input: "thursday",
			want:  time.Date(2024, time.January, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "weekday already passed this week rolls to next week",
			input: "tuesday",
			want:  time.Date(2024, time.January, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "bare weekday on today never means today",
			input: "wednesday",
			want:  time.Date(2024, time.January, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "next weekday skips a full week past the bare occurrence",
			input: "next thursday",
			want:  time.Date(2024, time.January, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "in N days",
			input: "in 3 days",
			want:  time.Date(2024, time.January, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "in 1 day singular",
			input: "in 1 day",
			want:  time.Date(2024, time.January, 11, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "RFC3339 fallback",
			input: "2025-06-01T08:30:00Z",
			want:  time.Date(2025, time.June, 1, 8, 30, 0, 0, time.UTC),
		},
		{
			name:    "unrecognized expression",
			input:   "next month",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "asdfasdf",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, refTime)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %v, want an error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
