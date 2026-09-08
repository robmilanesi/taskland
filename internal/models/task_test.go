package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTask_JSON_OmitsUnsetOptionalTimes(t *testing.T) {
	body, err := json.Marshal(Task{ID: uuid.New(), Title: "a", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := string(body)
	if strings.Contains(got, "completed_at") {
		t.Errorf("expected completed_at to be omitted when nil, got %s", got)
	}
	if strings.Contains(got, "due_date") {
		t.Errorf("expected due_date to be omitted when nil, got %s", got)
	}
}

func TestTask_JSON_IncludesSetOptionalTimes(t *testing.T) {
	now := time.Now()
	body, err := json.Marshal(Task{
		ID:          uuid.New(),
		Title:       "b",
		Completed:   true,
		CompletedAt: &now,
		DueDate:     &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := string(body)
	if !strings.Contains(got, "completed_at") {
		t.Errorf("expected completed_at to be present when set, got %s", got)
	}
	if !strings.Contains(got, "due_date") {
		t.Errorf("expected due_date to be present when set, got %s", got)
	}
}
