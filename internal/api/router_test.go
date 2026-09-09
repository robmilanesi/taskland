package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

func TestRouter_NewRouter_UnknownRoute(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       func(id uuid.UUID) string // il path può dipendere dall'id generato nel setup
		body       string
		setup      func(t *testing.T, repo repository.TaskRepository) uuid.UUID // ritorna l'id creato
		wantStatus int
	}{
		{
			name:   "GET task by id - found",
			method: http.MethodGet,
			setup: func(t *testing.T, repo repository.TaskRepository) uuid.UUID {
				task, err := repo.Create(models.Task{ID: uuid.New(), Title: "test"})
				if err != nil {
					t.Fatalf("setup: failed to create task: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET task by id - not found",
			method:     http.MethodGet,
			path:       func(_ uuid.UUID) string { return "/api/v1/tasks/" + uuid.New().String() },
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
			setup: func(t *testing.T, repo repository.TaskRepository) uuid.UUID {
				task, err := repo.Create(models.Task{ID: uuid.New(), Title: "to update"})
				if err != nil {
					t.Fatalf("setup: failed to create task: %v", err)
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
			setup: func(t *testing.T, repo repository.TaskRepository) uuid.UUID {
				task, err := repo.Create(models.Task{ID: uuid.New(), Title: "to delete"})
				if err != nil {
					t.Fatalf("setup: failed to create task: %v", err)
				}
				return task.ID
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusNoContent,
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
				t.Fatalf("not expected error during store initialization: %v", err)
			}

			var id uuid.UUID
			if tt.setup != nil {
				id = tt.setup(t, store.Tasks)
			}

			router := NewRouter(store.Tasks)

			path := tt.path(id)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, path, nil)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
