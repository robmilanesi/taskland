// Command server runs the taskland HTTP API.
package main

import (
	"log"
	"net/http"

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

	log.Println("server avviato su :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
