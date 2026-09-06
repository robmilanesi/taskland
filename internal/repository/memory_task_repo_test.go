package repository

import (
	"errors"
	"strings"
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

func TestInMemoryTaskREpo_Delete_NotFound(t *testing.T) {
	repo := newInMemoryTaskRepo()
	repo.tasks["123"] = models.Task{}

	tests := []struct {
		name       string
		idToDelete string
	}{
		{"empty-string", ""},
		{"uuid-not-present", uuid.New().String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Delete(tt.idToDelete)
			if err == nil {
				t.Errorf("expected error, got none")
			}
			if !errors.Is(err, ErrTaskNotFound) {
				t.Errorf("expected error to be ErrTaskNotFound, got %v", err)
			}

			if len(repo.tasks) != 1 {
				t.Errorf("length of tasks should not change, expected 1 got %d", len(repo.tasks))
			}
		})
	}
}

func TestInMemoryTaskREpo_Delete(t *testing.T) {
	repo := newInMemoryTaskRepo()
	uuid := uuid.New()
	taskToDelete := models.Task{ID: uuid, Title: "deleteme"}
	repo.tasks[uuid.String()] = models.Task{ID: uuid, Title: "deleteme"}

	task, err := repo.Delete(uuid.String())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if taskToDelete.ID != task.ID {
		t.Errorf("expected = %#v, got %#v", taskToDelete.ID, task.ID)
	}
}

func TestInMemoryTaskRepo_Delete_RemovesTask(t *testing.T) {
	repo := newInMemoryTaskRepo()
	id := uuid.New()
	stored := models.Task{ID: id, Title: "deleteme", CreatedAt: time.Now()}
	repo.tasks[id.String()] = stored

	got, err := repo.Delete(id.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != stored.ID || got.Title != stored.Title {
		t.Errorf("returned task = %+v, want %+v", got, stored)
	}

	if _, err := repo.GetByID(id.String()); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected task to be gone, GetByID returned %v", err)
	}
	if n, _ := repo.Count(ListTasksParams{}); n != 0 {
		t.Errorf("expected Count 0 after delete, got %d", n)
	}
}

func TestInMemoryTaskRepo_Delete_LeavesOtherTasks(t *testing.T) {
	repo := newInMemoryTaskRepo()
	ids := make([]uuid.UUID, 3)
	for i := range ids {
		id := uuid.New()
		ids[i] = id
		repo.tasks[id.String()] = models.Task{ID: id, Title: "t"}
	}

	if _, err := repo.Delete(ids[1].String()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if n, _ := repo.Count(ListTasksParams{}); n != 2 {
		t.Errorf("expected Count 2 after deleting one of three, got %d", n)
	}
	for _, keep := range []uuid.UUID{ids[0], ids[2]} {
		if _, err := repo.GetByID(keep.String()); err != nil {
			t.Errorf("expected %s to survive the delete, got %v", keep, err)
		}
	}
}

func TestInMemoryTaskRepo_Delete_Twice(t *testing.T) {
	repo := newInMemoryTaskRepo()
	id := uuid.New()
	repo.tasks[id.String()] = models.Task{ID: id}

	if _, err := repo.Delete(id.String()); err != nil {
		t.Fatalf("first delete: expected no error, got %v", err)
	}
	if _, err := repo.Delete(id.String()); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("second delete: expected ErrTaskNotFound, got %v", err)
	}
}

func TestInMemoryTaskRepo_Delete_NotFound_ErrorMentionsRequestedID(t *testing.T) {
	repo := newInMemoryTaskRepo()

	_, err := repo.Delete("task-42")
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), "task-42") {
		t.Errorf("expected error to mention the requested id, got %q", err.Error())
	}
}
func TestInMemoryTaskRepo_Update_NotFound(t *testing.T) {
	repo := newInMemoryTaskRepo()
	updateTask := models.Task{ID: uuid.New()}
	_, err := repo.Update(updateTask)
	if err == nil {
		t.Fatal("expected error, got none")
	}

	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("expected ErrTaskNotFound got: %#v instead", err)
	}

	if !strings.Contains(err.Error(), updateTask.ID.String()) {
		t.Errorf("expected error to mention the requested id, got %q", err.Error())
	}
}

func TestInMemoryTaskRepo_Update_CreatedAtNotChanged(t *testing.T) {
	repo := newInMemoryTaskRepo()
	storedTask := models.Task{ID: uuid.New(), Title: "saved", CreatedAt: time.Now()}
	repo.tasks[storedTask.ID.String()] = storedTask

	updateTask := models.Task{
		ID:        storedTask.ID,
		Title:     "updated",
		CreatedAt: time.Now().Add(10 * time.Second),
	}
	result, err := repo.Update(updateTask)
	if err != nil {
		t.Fatalf("expected no error, got: %#v", err)
	}
	persisted := repo.tasks[storedTask.ID.String()]

	if result.Title != updateTask.Title {
		t.Errorf("expected result title to be %s, got %s", updateTask.Title, result.Title)
	}
	if persisted.Title != updateTask.Title {
		t.Errorf("expected persisted title to be %s, got %s", updateTask.Title, persisted.Title)
	}

	if !result.CreatedAt.Equal(storedTask.CreatedAt) {
		t.Errorf("expected result creation date to be unchanged, got %v want %v", result.CreatedAt, storedTask.CreatedAt)
	}
	if !persisted.CreatedAt.Equal(storedTask.CreatedAt) {
		t.Errorf("expected persisted creation date to be unchanged, got %v want %v", persisted.CreatedAt, storedTask.CreatedAt)
	}
}

func TestInMemoryTaskRepo_Update_Completed(t *testing.T) {
	repo := newInMemoryTaskRepo()
	storedTask := models.Task{ID: uuid.New(), Title: "saved", Completed: false}
	repo.tasks[storedTask.ID.String()] = storedTask

	updateTask := models.Task{
		ID:        storedTask.ID,
		Title:     "saved",
		Completed: true,
	}
	result, err := repo.Update(updateTask)
	if err != nil {
		t.Fatalf("expected no error, got: %#v", err)
	}
	persisted := repo.tasks[storedTask.ID.String()]

	if result.Title != updateTask.Title {
		t.Errorf("expected result title to be %s, got %s", updateTask.Title, result.Title)
	}
	if persisted.Title != updateTask.Title {
		t.Errorf("expected persisted title to be %s, got %s", updateTask.Title, persisted.Title)
	}

	if !result.Completed {
		t.Errorf("expected result completed to be true")
	}
	if !persisted.Completed {
		t.Errorf("expected persisted completed to be true")
	}

	if result.CompletedAt.IsZero() {
		t.Errorf("expected result completedAt to be valorized upon completion")
	}
	if persisted.CompletedAt.IsZero() {
		t.Errorf("expected persisted completedAt to be valorized upon completion")
	}
}

func TestInMemoryTaskRepo_Update_Full(t *testing.T) {
	repo := newInMemoryTaskRepo()
	now := time.Now()
	storedTask := models.Task{
		ID:        uuid.New(),
		Title:     "saved",
		Completed: false,
		Priority:  models.PriorityLow,
		DueDate:   now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.tasks[storedTask.ID.String()] = storedTask

	updateTask := models.Task{
		ID:        storedTask.ID,
		Title:     "updated",
		Completed: true,
		Priority:  models.PriorityNone,
		DueDate:   now.Add(10 * time.Second),
		CreatedAt: now.Add(10 * time.Hour),
		UpdatedAt: now.Add(-10 * time.Hour),
	}
	result, err := repo.Update(updateTask)
	if err != nil {
		t.Fatalf("expected no error, got: %#v", err)
	}
	persisted := repo.tasks[storedTask.ID.String()]

	if result.Title != updateTask.Title {
		t.Errorf("expected result title to be %s, got %s", updateTask.Title, result.Title)
	}
	if persisted.Title != updateTask.Title {
		t.Errorf("expected persisted title to be %s, got %s", updateTask.Title, persisted.Title)
	}

	if result.Completed != updateTask.Completed {
		t.Errorf("expected result completed to be %t, got %t", updateTask.Completed, result.Completed)
	}
	if persisted.Completed != updateTask.Completed {
		t.Errorf("expected persisted completed to be %t, got %t", updateTask.Completed, persisted.Completed)
	}

	if result.Priority != updateTask.Priority {
		t.Errorf("expected result priority to be %v, got %v", updateTask.Priority, result.Priority)
	}
	if persisted.Priority != updateTask.Priority {
		t.Errorf("expected persisted priority to be %v, got %v", updateTask.Priority, persisted.Priority)
	}

	if !result.DueDate.Equal(updateTask.DueDate) {
		t.Errorf("expected result due date to be %v, got %v", updateTask.DueDate, result.DueDate)
	}
	if !persisted.DueDate.Equal(updateTask.DueDate) {
		t.Errorf("expected persisted due date to be %v, got %v", updateTask.DueDate, persisted.DueDate)
	}

	if !result.Completed {
		t.Errorf("expected completed to be true")
	}
	if result.CompletedAt.IsZero() {
		t.Errorf("expected completedAt to be valorized upon completion")
	}
}

func TestInMemoryTaskRepo_Update_SetsUpdatedAt(t *testing.T) {
	repo := newInMemoryTaskRepo()
	past := time.Now().Add(-24 * time.Hour)
	storedTask := models.Task{
		ID:        uuid.New(),
		Title:     "saved",
		UpdatedAt: past,
	}
	repo.tasks[storedTask.ID.String()] = storedTask

	before := time.Now()
	updateTask := models.Task{
		ID:        storedTask.ID,
		Title:     "updated",
		UpdatedAt: past.Add(-10 * time.Hour),
	}
	result, err := repo.Update(updateTask)
	after := time.Now()
	if err != nil {
		t.Fatalf("expected no error, got: %#v", err)
	}

	if result.UpdatedAt.Before(before) || result.UpdatedAt.After(after) {
		t.Errorf("expected UpdatedAt to be set to now (between %v and %v), got %v", before, after, result.UpdatedAt)
	}

	persisted := repo.tasks[storedTask.ID.String()]
	if !persisted.UpdatedAt.Equal(result.UpdatedAt) {
		t.Errorf("expected persisted UpdatedAt to match returned result, got %v want %v", persisted.UpdatedAt, result.UpdatedAt)
	}
}
