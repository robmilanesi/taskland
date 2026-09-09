// Command server runs the taskland HTTP API.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/robmilanesi/taskland/internal/api"
	"github.com/robmilanesi/taskland/internal/config"
	"github.com/robmilanesi/taskland/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	repo, err := repository.NewTaskRepository(repository.Config{
		Type: repository.TaskRepoSQLite,
		DSN:  cfg.DBPath,
	})
	if err != nil {
		log.Fatalf("failed to initialize task repository: %v", err)
	}
	log.Println("task repository: sqlite")

	router := api.NewRouter(repo)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("starting server")
	log.Fatal(server.ListenAndServe())
}
