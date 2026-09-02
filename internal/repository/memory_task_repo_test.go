package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robmilanesi/taskland/internal/models"
)

func TestInMemoryTaskRepo_GetByID_NotFound(t *testing.T) {
	repo := newInMemoryTaskRepo()

	_, err := repo.GetByID("nonexistent")

	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestInMemoryTaskRepo_GetByID_Found(t *testing.T) {
	repo := newInMemoryTaskRepo()

	repo.tasks["1"] = models.Task{ID: uuid.New(), Title: "Test"}

	task, err := repo.GetByID("1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.ID != repo.tasks["1"].ID {
		t.Errorf("expected task ID '1', got %q", task.ID)
	}
}

func TestInMemoryTaskRepo_GetAll_ZeroPage_DefaultsToPageOne(t *testing.T) {
	repo := newInMemoryTaskRepo()
	repo.tasks["1"] = models.Task{ID: uuid.New(), Title: "Uno"}

	taskList, err := repo.GetAll(ListTasksParams{Page: 0, Size: 20})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != 1 {
		t.Errorf("expected page=0 to default to page 1, got %d tasks", len(taskList))
	}
}

func TestInMemoryTaskRepo_GetAll_Empty(t *testing.T) {
	repo := newInMemoryTaskRepo()

	taskList, err := repo.GetAll(ListTasksParams{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != 0 {
		t.Errorf("expected tasks to be empty, got %d tasks", len(taskList))
	}

}

func TestInMemoryTaskRepo_GetAll_ZeroSize(t *testing.T) {
	repo := newInMemoryTaskRepo()

	taskList, err := repo.GetAll(ListTasksParams{Size: 0})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != 0 {
		t.Errorf("expected tasks to be empty, got %d tasks", len(taskList))
	}

}

func TestInMemoryTaskRepo_GetAll_DefaultSort(t *testing.T) {
	repo := newInMemoryTaskRepo()

	base := time.Now()
	ids := make([]uuid.UUID, 5)
	for i, title := range []string{"Uno", "Due", "Tre", "Quattro", "Cinque"} {
		id := uuid.New()
		ids[i] = id
		repo.tasks[id.String()] = models.Task{
			ID:        id,
			Title:     title,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
	}
	taskList, err := repo.GetAll(ListTasksParams{Page: 1, Size: 20})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(taskList) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(taskList))
	}

	for i, task := range taskList {
		if task.ID != ids[i] {
			t.Errorf("expected task at position %d to be %s, got %s", i, ids[i], task.ID)
		}
	}
}

func TestInMemoryTaskRepo_GetAll_Pagination(t *testing.T) {
	repo := newInMemoryTaskRepo()

	base := time.Now()
	ids := make([]uuid.UUID, 5)
	titles := []string{"Uno", "Due", "Tre", "Quattro", "Cinque"}
	for i, title := range titles {
		id := uuid.New()
		ids[i] = id
		repo.tasks[id.String()] = models.Task{
			ID:        id,
			Title:     title,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
	}
	taskList, err := repo.GetAll(ListTasksParams{Page: 3, Size: 1})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != 1 {
		t.Fatalf("expected 1 task, got %d", len(taskList))
	}
	if taskList[0].ID != ids[2] {
		t.Errorf("expected task at page 3 to be %s (%q), got %s (%q)",
			ids[2], titles[2], taskList[0].ID, taskList[0].Title)
	}
}

func TestInMemoryTaskRepo_Create_Empty(t *testing.T) {
	repo := newInMemoryTaskRepo()

	task, err := repo.Create(models.Task{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if task.ID == uuid.Nil {
		t.Error("expected uuid to be valorized, got empty value")
	}

	if task.Title != "" {
		t.Error("expected title to be empty")
	}

	if task.CreatedAt.IsZero() {
		t.Error("expected creation date to be populated, zero value got instead")
	}
}
