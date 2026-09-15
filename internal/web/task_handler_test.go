package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robmilanesi/taskland/internal/repository"
)

func createTask(t *testing.T, router http.Handler, cookie *http.Cookie, title string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"title": {title}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRouter_CreateTask_AddsToInbox(t *testing.T) {
	router, _ := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "task-a@example.com")

	rec := createTask(t, router, cookie, "buy milk")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "buy milk") {
		t.Errorf("expected the returned fragment to contain the new task, got: %s", rec.Body)
	}

	home := httptest.NewRequest(http.MethodGet, "/", nil)
	home.AddCookie(cookie)
	homeRec := httptest.NewRecorder()
	router.ServeHTTP(homeRec, home)
	if !strings.Contains(homeRec.Body.String(), "buy milk") {
		t.Errorf("expected the home page to show the new task, got: %s", homeRec.Body)
	}
}

func TestRouter_CreateTask_RequiresAuth(t *testing.T) {
	router, _ := newRouterAndStore(t)

	form := url.Values{"title": {"nope"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rec.Code)
	}
}

func TestRouter_CreateTask_EmptyTitle(t *testing.T) {
	router, _ := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "task-b@example.com")

	rec := createTask(t, router, cookie, "   ")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRouter_ToggleTask(t *testing.T) {
	router, store := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "task-c@example.com")
	createTask(t, router, cookie, "wash dishes")

	user, err := store.Users.GetUserByEmail(t.Context(), "task-c@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	tasks, err := store.Tasks.GetAll(t.Context(), user.ID, repository.ListTasksParams{Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	taskID := tasks[0].ID.String()

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/toggle", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "checked") {
		t.Errorf("expected the toggled task to render checked, got: %s", rec.Body)
	}

	// Toggling again flips it back to incomplete.
	req2 := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/toggle", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if strings.Contains(rec2.Body.String(), "checked") {
		t.Errorf("expected the task to be unchecked again, got: %s", rec2.Body)
	}
}

func TestRouter_ToggleTask_NotFound(t *testing.T) {
	router, _ := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "task-d@example.com")

	req := httptest.NewRequest(http.MethodPost, "/tasks/00000000-0000-0000-0000-000000000000/toggle", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRouter_DeleteTask(t *testing.T) {
	router, store := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "task-e@example.com")
	createTask(t, router, cookie, "throw away")

	user, err := store.Users.GetUserByEmail(t.Context(), "task-e@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	tasks, err := store.Tasks.GetAll(t.Context(), user.ID, repository.ListTasksParams{Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	taskID := tasks[0].ID.String()

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/delete", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "throw away") {
		t.Errorf("expected the deleted task to be gone, got: %s", rec.Body)
	}

	// Deleting again is idempotent: still 200, not 404.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/delete", nil)
	req2.AddCookie(cookie)
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("second delete status = %d, want 200", rec2.Code)
	}
}
