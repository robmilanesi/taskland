package api

import (
	"net/http"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/repository"
)

// NewRouter builds the HTTP router for the API. It registers:
//
//	POST   /api/v1/auth/register - create an account
//	POST   /api/v1/auth/login    - exchange credentials for a token
//	GET    /api/v1/tasks/{id}    - retrieve a single task by ID
//	GET    /api/v1/tasks         - retrieve all tasks
//	POST   /api/v1/tasks         - create a new task
//	PATCH  /api/v1/tasks/{id}    - partially update a task by ID
//	DELETE /api/v1/tasks/{id}    - delete a task by ID
//
// The /auth routes are public; the /tasks routes require a valid Bearer token
// and every route runs through the RequestID, RequestLogger and Recover middleware.
func NewRouter(store *repository.Store, issuer *auth.Issuer) http.Handler {
	th := NewTaskHandler(store.Tasks)
	ah := NewAuthHandler(store.Users, issuer)

	authed := Authenticate(issuer)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/auth/register", ah.Register)
	mux.HandleFunc("POST /api/v1/auth/login", ah.Login)

	mux.Handle("GET /api/v1/tasks/{id}", authed(http.HandlerFunc(th.GetTask)))
	mux.Handle("GET /api/v1/tasks", authed(http.HandlerFunc(th.GetAllTasks)))
	mux.Handle("POST /api/v1/tasks", authed(http.HandlerFunc(th.Create)))
	mux.Handle("PATCH /api/v1/tasks/{id}", authed(http.HandlerFunc(th.Update)))
	mux.Handle("DELETE /api/v1/tasks/{id}", authed(http.HandlerFunc(th.Delete)))

	return httpx.Chain(mux, httpx.RequestID, httpx.RequestLogger, httpx.Recover)
}
