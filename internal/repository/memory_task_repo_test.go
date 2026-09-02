package repository

import (
	"errors"
	"strconv"
	"testing"

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
	repo.tasks["1"] = models.Task{ID: "1", Title: "Test"}

	task, err := repo.GetByID("1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.ID != "1" {
		t.Errorf("expected task ID '1', got %q", task.ID)
	}
}

func TestInMemoryTaskRepo_GetAll_ZeroPage_DefaultsToPageOne(t *testing.T) {
	repo := newInMemoryTaskRepo()
	repo.tasks["1"] = models.Task{ID: "1", Title: "Uno"}

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
	repo.tasks["1"] = models.Task{ID: "1", Title: "Uno"}
	repo.tasks["2"] = models.Task{ID: "2", Title: "Due"}
	repo.tasks["3"] = models.Task{ID: "3", Title: "Tre"}
	repo.tasks["4"] = models.Task{ID: "4", Title: "Quattro"}
	repo.tasks["5"] = models.Task{ID: "5", Title: "Cinque"}

	taskList, err := repo.GetAll(ListTasksParams{Page: 1, Size: 20})
	expectedLen := 5

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != expectedLen {
		t.Errorf("expected tasks to be %d, got %d", expectedLen, len(taskList))
	}

	for i := range expectedLen {
		index := strconv.Itoa(i + 1)
		if taskList[i].ID != index {
			t.Errorf("expected first task to be with id '%s', got '%s' instead", index, taskList[i].ID)
		}
	}

}

func TestInMemoryTaskRepo_GetAll_Pagination(t *testing.T) {
	repo := newInMemoryTaskRepo()
	repo.tasks["1"] = models.Task{ID: "1", Title: "Uno"}
	repo.tasks["2"] = models.Task{ID: "2", Title: "Due"}
	repo.tasks["3"] = models.Task{ID: "3", Title: "Tre"}
	repo.tasks["4"] = models.Task{ID: "4", Title: "Quattro"}
	repo.tasks["5"] = models.Task{ID: "5", Title: "Cinque"}

	taskList, err := repo.GetAll(ListTasksParams{Page: 3, Size: 1})
	expectedLen := 1

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(taskList) != expectedLen {
		t.Errorf("expected tasks to be %d, got %d", expectedLen, len(taskList))
	}
	if taskList[0].ID != "3" {
		t.Errorf("expected to filter for task 3, got %s", taskList[0].ID)
	}
}
