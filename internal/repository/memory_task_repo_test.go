package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// The behavioural surface of the in-memory repository is exercised by the
// shared contract in contract_test.go (TestTaskRepositoryContract_InMemory).
// The tests below are white-box: they seed r.tasks directly to check things
// the contract cannot express portably.

func TestInMemoryTaskRepo_GetAll_SortsByCreatedAt(t *testing.T) {
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
			t.Errorf("position %d = %s, want %s", i, task.ID, ids[i])
		}
	}
}

func TestInMemoryTaskRepo_Update_ManagesTimestampsItself(t *testing.T) {
	repo := newInMemoryTaskRepo()

	created := time.Now().Add(-24 * time.Hour)
	id := uuid.New()
	repo.tasks[id.String()] = models.Task{
		ID:        id,
		Title:     "saved",
		CreatedAt: created,
		UpdatedAt: created,
	}

	before := time.Now()
	result, err := repo.Update(models.Task{
		ID:        id,
		Title:     "updated",
		CreatedAt: time.Now().Add(10 * time.Hour),
		UpdatedAt: time.Now().Add(-10 * time.Hour),
	})
	after := time.Now()
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !result.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt should keep the stored value %v, got %v", created, result.CreatedAt)
	}
	if result.UpdatedAt.Before(before) || result.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt should be ~now (in [%v, %v]), got %v", before, after, result.UpdatedAt)
	}

	persisted := repo.tasks[id.String()]
	if !persisted.CreatedAt.Equal(result.CreatedAt) || !persisted.UpdatedAt.Equal(result.UpdatedAt) {
		t.Errorf("persisted timestamps %v/%v don't match returned %v/%v",
			persisted.CreatedAt, persisted.UpdatedAt, result.CreatedAt, result.UpdatedAt)
	}
}
