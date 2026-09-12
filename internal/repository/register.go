package repository

import (
	"context"

	"github.com/robmilanesi/taskland/internal/models"
)

// RegisterAccount creates a new user together with their default Inbox list.
// On the SQLite backend this happens in a single transaction; the in-memory
// backend has no partial-failure to guard against, so it is just two
// sequential calls.
func (s *Store) RegisterAccount(ctx context.Context, user models.User) (models.User, error) {
	return s.registerAccount(ctx, user)
}

func registerAccountSequential(users UserRepository, lists ListRepository) func(context.Context, models.User) (models.User, error) {
	return func(ctx context.Context, user models.User) (models.User, error) {
		created, err := users.CreateUser(ctx, user)
		if err != nil {
			return models.User{}, err
		}
		if _, err := lists.InboxFor(ctx, created.ID); err != nil {
			return models.User{}, err
		}
		return created, nil
	}
}
