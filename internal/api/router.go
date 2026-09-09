package api

import (
	"net/http"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/repository"
)

// NewRouter builds and returns the HTTP router for the tasks API.
// It wires up a TaskHandler backed by the given repository and registers
// the following routes:
//
//	GET    /api/v1/tasks/{id}  - retrieve a single task by ID
//	GET    /api/v1/tasks       - retrieve all tasks
//	POST   /api/v1/tasks       - create a new task
//	PATCH  /api/v1/tasks/{id}  - partially update a task by ID
//	DELETE /api/v1/tasks/{id}  - delete a task by ID
//
// Every route runs through the RequestID, RequestLogger and Recover middleware.
func NewRouter(repo repository.TaskRepository) http.Handler {
	th := NewTaskHandler(repo)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tasks/{id}", th.GetTask)
	mux.HandleFunc("GET /api/v1/tasks", th.GetAllTasks)
	mux.HandleFunc("POST /api/v1/tasks", th.Create)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", th.Update)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", th.Delete)

	return httpx.Chain(mux, httpx.RequestID, httpx.RequestLogger, httpx.Recover)
}
