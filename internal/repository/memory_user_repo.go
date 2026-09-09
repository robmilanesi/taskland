package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

type inMemoryUserRepository struct {
	users map[uuid.UUID]models.User
}

func newInMemoryUserRepo() *inMemoryUserRepository {
	return &inMemoryUserRepository{users: map[uuid.UUID]models.User{}}
}

func (r *inMemoryUserRepository) CreateUser(_ context.Context, user models.User) (models.User, error) {
	for _, existing := range r.users {
		if existing.Email == user.Email {
			return models.User{}, ErrEmailTaken
		}
	}

	user.ID = uuid.New()
	user.CreatedAt = time.Now().UTC()
	r.users[user.ID] = user
	return user, nil
}

func (r *inMemoryUserRepository) GetUserByEmail(_ context.Context, email string) (models.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return models.User{}, ErrUserNotFound
}

func (r *inMemoryUserRepository) GetUserByID(_ context.Context, id uuid.UUID) (models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return models.User{}, ErrUserNotFound
	}
	return user, nil
}
