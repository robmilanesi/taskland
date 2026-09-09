// Command server runs the taskland HTTP API.
package main

import (
	"cmp"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/robmilanesi/taskland/internal/api"
	"github.com/robmilanesi/taskland/internal/repository"
)

func main() {
	dsn := cmp.Or(os.Getenv("TASKLAND_DB_PATH"), "taskland.db")

	repo, err := repository.NewTaskRepository(repository.Config{
		Type: repository.TaskRepoSQLite,
		DSN:  dsn,
	})
	if err != nil {
		log.Fatalf("failed to initialize task repository: %v", err)
	}
	log.Println("task repository: sqlite")

	router := api.NewRouter(repo)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("server avviato su :8080")
	log.Fatal(server.ListenAndServe())
}
