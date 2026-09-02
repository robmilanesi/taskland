package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/robmilanesi/taskland/internal/models"
)

func TestTaskHandler_Create_Created(t *testing.T) {
	var passed models.Task
	repo := stubTaskRepo{createFn: func(in models.Task) (models.Task, error) {
		passed = in
		in.ID = uuid.New()
		return in, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"  Buy milk  "}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if passed.Title != "Buy milk" {
		t.Errorf("expected trimmed title passed to repo, got %q", passed.Title)
	}

	var body models.Task
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Title != "Buy milk" {
		t.Errorf("unexpected response title: %q", body.Title)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/v1/tasks/"+body.ID.String() {
		t.Errorf("unexpected Location header: %q", loc)
	}
}

func TestTaskHandler_Create_ValidationError(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		t.Fatal("repo.Create must not be called on invalid input")
		return models.Task{}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"   "}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_MalformedJSON(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		t.Fatal("repo.Create must not be called on bad JSON")
		return models.Task{}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_UnknownField(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		return models.Task{}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"ok","done":true}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field (DisallowUnknownFields), got %d", rec.Code)
	}
}

func TestTaskHandler_Create_RepoError(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		return models.Task{}, errors.New("boom")
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"ok"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
