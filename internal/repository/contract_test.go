package repository

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// testTaskRepositoryContract runs the behavioural checks that every
// TaskRepository implementation must satisfy. mk must return a fresh, empty
// repository each time it is called.
func testTaskRepositoryContract(t *testing.T, mk func(t *testing.T) TaskRepository) {
	t.Helper()

	create := func(t *testing.T, repo TaskRepository, title string) models.Task {
		t.Helper()
		task, err := repo.Create(models.Task{Title: title})
		if err != nil {
			t.Fatalf("Create(%q): %v", title, err)
		}
		return task
	}

	count := func(t *testing.T, repo TaskRepository) int {
		t.Helper()
		n, err := repo.Count(ListTasksParams{})
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		return n
	}

	t.Run("Create assigns id and equal timestamps", func(t *testing.T) {
		repo := mk(t)
		before := time.Now().Add(-time.Second)

		task, err := repo.Create(models.Task{Title: "x"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		after := time.Now().Add(time.Second)

		if task.ID == uuid.Nil {
			t.Error("expected a generated id")
		}
		if task.CreatedAt.Before(before) || task.CreatedAt.After(after) {
			t.Errorf("CreatedAt %v outside [%v, %v]", task.CreatedAt, before, after)
		}
		if !task.UpdatedAt.Equal(task.CreatedAt) {
			t.Errorf("UpdatedAt %v != CreatedAt %v on a fresh task", task.UpdatedAt, task.CreatedAt)
		}
	})

	t.Run("Create then GetByID roundtrips fields", func(t *testing.T) {
		repo := mk(t)
		due := time.Date(2031, 6, 1, 8, 30, 0, 0, time.UTC)

		created, err := repo.Create(models.Task{
			Title:       "roundtrip",
			Description: "desc",
			Priority:    models.PriorityHigh,
			DueDate:     &due,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := repo.GetByID(created.ID.String())
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Title != "roundtrip" || got.Description != "desc" || got.Priority != models.PriorityHigh {
			t.Errorf("fields not preserved: %+v", got)
		}
		if got.Completed || got.CompletedAt != nil {
			t.Errorf("expected not completed, got completed=%v completedAt=%v", got.Completed, got.CompletedAt)
		}
		if got.DueDate == nil || !got.DueDate.Equal(due) {
			t.Errorf("DueDate = %v, want %v", got.DueDate, due)
		}
	})

	t.Run("GetByID unknown returns ErrTaskNotFound with id", func(t *testing.T) {
		repo := mk(t)
		id := uuid.NewString()

		_, err := repo.GetByID(id)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got %v", err)
		}
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error %q should mention the requested id %q", err, id)
		}
	})

	t.Run("GetAll empty returns no tasks", func(t *testing.T) {
		repo := mk(t)
		got, err := repo.GetAll(ListTasksParams{Page: 1, Size: 10})
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(got))
		}
	})

	t.Run("GetAll orders by creation and paginates", func(t *testing.T) {
		repo := mk(t)
		var ids []uuid.UUID
		for _, title := range []string{"a", "b", "c", "d", "e"} {
			ids = append(ids, create(t, repo, title).ID)
			time.Sleep(2 * time.Millisecond)
		}

		page1, err := repo.GetAll(ListTasksParams{Page: 1, Size: 2})
		if err != nil {
			t.Fatalf("GetAll page 1: %v", err)
		}
		if len(page1) != 2 || page1[0].ID != ids[0] || page1[1].ID != ids[1] {
			t.Errorf("page 1 = %v, want the first two in creation order", idsOf(page1))
		}

		page3, err := repo.GetAll(ListTasksParams{Page: 3, Size: 2})
		if err != nil {
			t.Fatalf("GetAll page 3: %v", err)
		}
		if len(page3) != 1 || page3[0].ID != ids[4] {
			t.Errorf("page 3 = %v, want just the last task", idsOf(page3))
		}

		past, err := repo.GetAll(ListTasksParams{Page: 99, Size: 2})
		if err != nil {
			t.Fatalf("GetAll page 99: %v", err)
		}
		if len(past) != 0 {
			t.Errorf("page past the end = %d tasks, want 0", len(past))
		}
	})

	t.Run("GetAll clamps non-positive page and size", func(t *testing.T) {
		repo := mk(t)
		create(t, repo, "only")

		got, err := repo.GetAll(ListTasksParams{Page: 0, Size: 0})
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("expected page/size to default, got %d tasks", len(got))
		}
	})

	t.Run("Count reflects inserts and deletes", func(t *testing.T) {
		repo := mk(t)
		if n := count(t, repo); n != 0 {
			t.Fatalf("fresh repo Count = %d, want 0", n)
		}

		a := create(t, repo, "a")
		create(t, repo, "b")
		if n := count(t, repo); n != 2 {
			t.Fatalf("after 2 inserts Count = %d, want 2", n)
		}

		if _, err := repo.Delete(a.ID.String()); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if n := count(t, repo); n != 1 {
			t.Errorf("after delete Count = %d, want 1", n)
		}
	})

	t.Run("Update changes mutable fields, preserves id/created_at, bumps updated_at", func(t *testing.T) {
		repo := mk(t)
		orig := create(t, repo, "before")
		time.Sleep(2 * time.Millisecond)

		due := time.Date(2032, 2, 2, 0, 0, 0, 0, time.UTC)
		updated, err := repo.Update(models.Task{
			ID:          orig.ID,
			Title:       "after",
			Description: "new",
			Priority:    models.PriorityLow,
			DueDate:     &due,
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}

		if updated.ID != orig.ID {
			t.Errorf("id changed: %v -> %v", orig.ID, updated.ID)
		}
		if !updated.CreatedAt.Equal(orig.CreatedAt) {
			t.Errorf("CreatedAt changed: %v -> %v", orig.CreatedAt, updated.CreatedAt)
		}
		if !updated.UpdatedAt.After(orig.UpdatedAt) {
			t.Errorf("UpdatedAt not advanced: %v -> %v", orig.UpdatedAt, updated.UpdatedAt)
		}
		if updated.Title != "after" || updated.Description != "new" || updated.Priority != models.PriorityLow {
			t.Errorf("mutable fields not applied: %+v", updated)
		}
		if updated.DueDate == nil || !updated.DueDate.Equal(due) {
			t.Errorf("DueDate = %v, want %v", updated.DueDate, due)
		}

		got, err := repo.GetByID(orig.ID.String())
		if err != nil {
			t.Fatalf("GetByID after update: %v", err)
		}
		if got.Title != "after" {
			t.Errorf("update not persisted: title = %q", got.Title)
		}
	})

	t.Run("Update unknown returns ErrTaskNotFound with id", func(t *testing.T) {
		repo := mk(t)
		id := uuid.New()

		_, err := repo.Update(models.Task{ID: id, Title: "x"})
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got %v", err)
		}
		if !strings.Contains(err.Error(), id.String()) {
			t.Errorf("error %q should mention the requested id %q", err, id)
		}
	})

	t.Run("Update completion transitions completed_at", func(t *testing.T) {
		repo := mk(t)
		task := create(t, repo, "task")

		done, err := repo.Update(models.Task{ID: task.ID, Title: "task", Completed: true})
		if err != nil {
			t.Fatalf("Update complete: %v", err)
		}
		if !done.Completed || done.CompletedAt == nil {
			t.Fatalf("expected completed with a timestamp, got %+v", done)
		}
		stamp := *done.CompletedAt

		again, err := repo.Update(models.Task{ID: task.ID, Title: "task", Completed: true})
		if err != nil {
			t.Fatalf("Update still-complete: %v", err)
		}
		if again.CompletedAt == nil || !again.CompletedAt.Equal(stamp) {
			t.Errorf("completed_at changed on a no-op re-complete: %v -> %v", stamp, again.CompletedAt)
		}

		reopened, err := repo.Update(models.Task{ID: task.ID, Title: "task", Completed: false})
		if err != nil {
			t.Fatalf("Update uncomplete: %v", err)
		}
		if reopened.Completed || reopened.CompletedAt != nil {
			t.Errorf("expected uncompleted with no timestamp, got %+v", reopened)
		}
	})

	t.Run("Update leaves other tasks untouched", func(t *testing.T) {
		repo := mk(t)
		keep := create(t, repo, "keep")
		target := create(t, repo, "target")

		if _, err := repo.Update(models.Task{ID: target.ID, Title: "changed"}); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, err := repo.GetByID(keep.ID.String())
		if err != nil {
			t.Fatalf("GetByID keep: %v", err)
		}
		if got.Title != "keep" || !got.UpdatedAt.Equal(keep.UpdatedAt) {
			t.Errorf("bystander task was modified: %+v", got)
		}
	})

	t.Run("Delete returns the task and removes it", func(t *testing.T) {
		repo := mk(t)
		task := create(t, repo, "goner")

		deleted, err := repo.Delete(task.ID.String())
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if deleted.ID != task.ID || deleted.Title != "goner" {
			t.Errorf("Delete returned %+v, want the created task", deleted)
		}
		if _, err := repo.GetByID(task.ID.String()); !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("task still retrievable after delete: %v", err)
		}
	})

	t.Run("Delete unknown returns ErrTaskNotFound with id", func(t *testing.T) {
		repo := mk(t)
		id := uuid.NewString()

		_, err := repo.Delete(id)
		if !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("expected ErrTaskNotFound, got %v", err)
		}
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error %q should mention the requested id %q", err, id)
		}
	})

	t.Run("Delete leaves other tasks untouched", func(t *testing.T) {
		repo := mk(t)
		keep := create(t, repo, "keep")
		gone := create(t, repo, "gone")

		if _, err := repo.Delete(gone.ID.String()); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if n := count(t, repo); n != 1 {
			t.Errorf("Count = %d after deleting one of two, want 1", n)
		}
		if _, err := repo.GetByID(keep.ID.String()); err != nil {
			t.Errorf("kept task not retrievable: %v", err)
		}
	})
}

func idsOf(tasks []models.Task) []uuid.UUID {
	ids := make([]uuid.UUID, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}

func TestTaskRepositoryContract_InMemory(t *testing.T) {
	testTaskRepositoryContract(t, func(*testing.T) TaskRepository {
		return newInMemoryTaskRepo()
	})
}

func TestTaskRepositoryContract_SQLite(t *testing.T) {
	testTaskRepositoryContract(t, func(t *testing.T) TaskRepository {
		return newSQLiteTaskRepo(newTestDB(t))
	})
}
