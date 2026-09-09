package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

	h.Create(rec, withOwner(req, testOwner))

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

	h.Create(rec, withOwner(req, testOwner))

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

	h.Create(rec, withOwner(req, testOwner))

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

	h.Create(rec, withOwner(req, testOwner))

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

	h.Create(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_WithAllFields(t *testing.T) {
	var passed models.Task
	repo := stubTaskRepo{createFn: func(in models.Task) (models.Task, error) {
		passed = in
		in.ID = uuid.New()
		return in, nil
	}}
	h := NewTaskHandler(repo)

	due := time.Date(2030, 1, 2, 15, 4, 5, 0, time.UTC)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"  write tests  ","description":"the full set","priority":3,"due_date":"2030-01-02T15:04:05Z"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if passed.Title != "write tests" {
		t.Errorf("expected trimmed title, got %q", passed.Title)
	}
	if passed.Description != "the full set" {
		t.Errorf("unexpected description: %q", passed.Description)
	}
	if passed.Priority != models.PriorityHigh {
		t.Errorf("expected priority %v, got %v", models.PriorityHigh, passed.Priority)
	}
	if passed.DueDate == nil || !passed.DueDate.Equal(due) {
		t.Errorf("expected due date %v, got %v", due, passed.DueDate)
	}
}

func TestTaskHandler_Create_PriorityOutOfRange(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		t.Fatal("repo.Create must not be called on invalid priority")
		return models.Task{}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"ok","priority":9}`))
	rec := httptest.NewRecorder()

	h.Create(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTaskHandler_Create_MalformedDueDate(t *testing.T) {
	repo := stubTaskRepo{createFn: func(models.Task) (models.Task, error) {
		t.Fatal("repo.Create must not be called on unparseable due_date")
		return models.Task{}, nil
	}}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"ok","due_date":"next tuesday"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unparseable due_date, got %d", rec.Code)
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

	h.Delete(rec, withOwner(req, testOwner))

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

	h.Delete(rec, withOwner(req, testOwner))

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

	h.Delete(rec, withOwner(req, testOwner))

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

	h.Delete(rec, withOwner(req, testOwner))

	if gotID != "" {
		t.Errorf("expected empty id forwarded to repo, got %q", gotID)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing id (current behaviour), got %d", rec.Code)
	}
}

func TestTaskHandler_Update_OK(t *testing.T) {
	existing := models.Task{
		ID:        uuid.New(),
		Title:     "old",
		CreatedAt: time.Now().Add(-time.Hour),
	}
	var updatedArg models.Task
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) { return existing, nil },
		updateFn: func(in models.Task) (models.Task, error) {
			updatedArg = in
			in.UpdatedAt = time.Now()
			return in, nil
		},
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/"+existing.ID.String(),
		strings.NewReader(`{"title":"  new title  ","completed":true}`))
	req.SetPathValue("id", existing.ID.String())
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if updatedArg.ID != existing.ID {
		t.Errorf("expected id preserved, got %s", updatedArg.ID)
	}
	if updatedArg.Title != "new title" {
		t.Errorf("expected trimmed title passed to repo, got %q", updatedArg.Title)
	}
	if !updatedArg.Completed {
		t.Errorf("expected completed=true passed to repo")
	}

	var body models.Task
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body.Title != "new title" {
		t.Errorf("unexpected response title: %q", body.Title)
	}
}

func TestTaskHandler_Update_PartialLeavesOtherFields(t *testing.T) {
	existing := models.Task{
		ID:          uuid.New(),
		Title:       "keep",
		Description: "keep desc",
		Priority:    models.PriorityHigh,
	}
	var updatedArg models.Task
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) { return existing, nil },
		updateFn:  func(in models.Task) (models.Task, error) { updatedArg = in; return in, nil },
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/x",
		strings.NewReader(`{"description":"new desc"}`))
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if updatedArg.Description != "new desc" {
		t.Errorf("expected description updated, got %q", updatedArg.Description)
	}
	if updatedArg.Title != "keep" {
		t.Errorf("expected title untouched, got %q", updatedArg.Title)
	}
	if updatedArg.Priority != models.PriorityHigh {
		t.Errorf("expected priority untouched, got %v", updatedArg.Priority)
	}
}

func TestTaskHandler_Update_NotFound(t *testing.T) {
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) {
			return models.Task{}, repository.ErrTaskNotFound
		},
		updateFn: func(models.Task) (models.Task, error) {
			t.Fatal("repo.Update must not be called when the task does not exist")
			return models.Task{}, nil
		},
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/missing",
		strings.NewReader(`{"title":"x"}`))
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestTaskHandler_Update_MalformedJSON(t *testing.T) {
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) {
			t.Fatal("repo.GetByID must not be called on bad JSON")
			return models.Task{}, nil
		},
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/x",
		strings.NewReader(`{"title":`))
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTaskHandler_Update_UnknownField(t *testing.T) {
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) {
			t.Fatal("repo must not be touched on unknown field")
			return models.Task{}, nil
		},
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/x",
		strings.NewReader(`{"nope":1}`))
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", rec.Code)
	}
}

func TestTaskHandler_Update_BlankTitleRejected(t *testing.T) {
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) {
			t.Fatal("repo must not be touched when validation fails")
			return models.Task{}, nil
		},
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/x",
		strings.NewReader(`{"title":"   "}`))
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank title, got %d", rec.Code)
	}
}

func TestTaskHandler_Update_RepoError(t *testing.T) {
	repo := stubTaskRepo{
		getByIDFn: func(string) (models.Task, error) { return models.Task{ID: uuid.New()}, nil },
		updateFn:  func(models.Task) (models.Task, error) { return models.Task{}, errors.New("boom") },
	}
	h := NewTaskHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/x",
		strings.NewReader(`{"title":"x"}`))
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()

	h.Update(rec, withOwner(req, testOwner))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
