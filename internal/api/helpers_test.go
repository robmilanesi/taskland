package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

// stubTaskRepo satisfies repository.TaskRepository. Its *Fn fields take only the
// arguments a given test cares about; ctx and ownerID are dropped unless a test
// points gotOwner at a variable to capture the owner.
type stubTaskRepo struct {
	createFn  func(models.Task) (models.Task, error)
	getByIDFn func(string) (models.Task, error)
	getAllFn  func(repository.ListTasksParams) ([]models.Task, error)
	count     func(repository.ListTasksParams) (int, error)
	deleteFn  func(string) (models.Task, error)
	updateFn  func(models.Task) (models.Task, error)
	gotOwner  *uuid.UUID
}

func (s stubTaskRepo) recordOwner(owner uuid.UUID) {
	if s.gotOwner != nil {
		*s.gotOwner = owner
	}
}

func (s stubTaskRepo) GetByID(_ context.Context, owner uuid.UUID, id string) (models.Task, error) {
	s.recordOwner(owner)
	return s.getByIDFn(id)
}

func (s stubTaskRepo) GetAll(_ context.Context, owner uuid.UUID, p repository.ListTasksParams) ([]models.Task, error) {
	s.recordOwner(owner)
	return s.getAllFn(p)
}

func (s stubTaskRepo) Count(_ context.Context, owner uuid.UUID, p repository.ListTasksParams) (int, error) {
	s.recordOwner(owner)
	return s.count(p)
}

func (s stubTaskRepo) Create(_ context.Context, owner uuid.UUID, t models.Task) (models.Task, error) {
	s.recordOwner(owner)
	return s.createFn(t)
}

func (s stubTaskRepo) Update(_ context.Context, owner uuid.UUID, t models.Task) (models.Task, error) {
	s.recordOwner(owner)
	return s.updateFn(t)
}

func (s stubTaskRepo) Delete(_ context.Context, owner uuid.UUID, id string) (models.Task, error) {
	s.recordOwner(owner)
	return s.deleteFn(id)
}

// testOwner is the authenticated user used by task-handler tests that don't
// care which user they are.
var testOwner = uuid.New()

// withOwner returns r carrying owner as its authenticated user, the way the
// Authenticate middleware would have set it.
func withOwner(r *http.Request, owner uuid.UUID) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ownerIDKey, owner))
}
