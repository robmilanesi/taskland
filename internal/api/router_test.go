package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

const routerTestSecret = "router-test-secret-with-enough-length"

const (
	authValid   = ""
	authNone    = "none"
	authForeign = "foreign"
)

func TestRouter_Routes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       func(id uuid.UUID) string
		body       string
		auth       string
		setup      func(t *testing.T, repo repository.TaskRepository, owner uuid.UUID) uuid.UUID
		wantStatus int
	}{
		{
			name:   "GET task by id - found",
			method: http.MethodGet,
			setup: func(t *testing.T, repo repository.TaskRepository, owner uuid.UUID) uuid.UUID {
				task, err := repo.Create(context.Background(), owner, models.Task{Title: "test"})
				if err != nil {
					t.Fatalf("setup: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET task by id - not found",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks/" + uuid.NewString() },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET all tasks",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST create task",
			method:     http.MethodPost,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			body:       `{"title":"new task"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:   "PATCH update task",
			method: http.MethodPatch,
			setup: func(t *testing.T, repo repository.TaskRepository, owner uuid.UUID) uuid.UUID {
				task, err := repo.Create(context.Background(), owner, models.Task{Title: "to update"})
				if err != nil {
					t.Fatalf("setup: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			body:       `{"title":"updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:   "DELETE task",
			method: http.MethodDelete,
			setup: func(t *testing.T, repo repository.TaskRepository, owner uuid.UUID) uuid.UUID {
				task, err := repo.Create(context.Background(), owner, models.Task{Title: "to delete"})
				if err != nil {
					t.Fatalf("setup: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "tasks without a token",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			auth:       authNone,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "tasks with a foreign token",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			auth:       authForeign,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "cannot GET another owner's task",
			method: http.MethodGet,
			setup: func(t *testing.T, repo repository.TaskRepository, _ uuid.UUID) uuid.UUID {
				task, err := repo.Create(context.Background(), uuid.New(), models.Task{Title: "not yours"})
				if err != nil {
					t.Fatalf("setup: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "method not allowed on collection",
			method:     http.MethodDelete,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/unknown" },
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
			if err != nil {
				t.Fatalf("NewStore: %v", err)
			}

			owner := uuid.New()
			issuer := auth.NewIssuer(routerTestSecret, time.Hour)
			token, err := issuer.Issue(owner)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			router := NewRouter(store, issuer)

			var id uuid.UUID
			if tt.setup != nil {
				id = tt.setup(t, store.Tasks, owner)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path(id), strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path(id), nil)
			}

			switch tt.auth {
			case authNone:
				// send no Authorization header
			case authForeign:
				foreign, _ := auth.NewIssuer("a-totally-different-secret-value!!", time.Hour).Issue(uuid.New())
				req.Header.Set("Authorization", "Bearer "+foreign)
			default:
				req.Header.Set("Authorization", "Bearer "+token)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
