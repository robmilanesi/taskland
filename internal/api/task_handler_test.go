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
	"github.com/robmilanesi/taskland/internal/repository"
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

func TestTaskHandler_Delete_NoContent(t *testing.T) {
	var gotID string
	repo := stubTaskRepo{deleteFn: func(id string) (models.Task, error) {
		gotID = id
		return models.Task{ID: uuid.New(), Title: "gone"}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body on 204, got %q", rec.Body.String())
	}
	if gotID != "abc" {
		t.Errorf("expected path id %q forwarded to repo, got %q", "abc", gotID)
	}
}

func TestTaskHandler_Delete_NotFound(t *testing.T) {
	repo := stubTaskRepo{deleteFn: func(string) (models.Task, error) {
		return models.Task{}, repository.ErrTaskNotFound
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected an error envelope, got %v", body)
	}
}

func TestTaskHandler_Delete_RepoError(t *testing.T) {
	repo := stubTaskRepo{deleteFn: func(string) (models.Task, error) {
		return models.Task{}, errors.New("boom")
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestTaskHandler_Delete_MissingPathValue(t *testing.T) {
	gotID := "unset"
	repo := stubTaskRepo{deleteFn: func(id string) (models.Task, error) {
		gotID = id
		return models.Task{}, repository.ErrTaskNotFound
	}}
	h := NewTaskHandler(repo)

	// no SetPathValue: r.PathValue("id") returns ""
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/", nil)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if gotID != "" {
		t.Errorf("expected empty id forwarded to repo, got %q", gotID)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing id (current behaviour), got %d", rec.Code)
	}
}
