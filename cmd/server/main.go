// Command server runs the taskland HTTP API.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/robmilanesi/taskland/internal/api"
	"github.com/robmilanesi/taskland/internal/repository"
)

func main() {
	repo, err := repository.NewTaskRepository(repository.TaskRepoInMemory)
	if err != nil {
		log.Fatalf("failed to initialize task repository: %v", err)
	}

	taskHandler := api.NewTaskHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tasks/{id}", taskHandler.GetTask)
	mux.HandleFunc("GET /api/v1/tasks", taskHandler.GetAllTasks)
	mux.HandleFunc("POST /api/v1/tasks", taskHandler.Create)
	mux.HandleFunc("DELETE /api/v1/tasks", taskHandler.Delete)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("server avviato su :8080")
	log.Fatal(server.ListenAndServe())
}
