package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

func testUserRepositoryContract(t *testing.T, mk func(t *testing.T) UserRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("CreateUser assigns id and timestamp", func(t *testing.T) {
		repo := mk(t)
		before := time.Now().Add(-time.Second)

		u, err := repo.CreateUser(ctx, models.User{Email: "a@example.com", PasswordHash: "hash"})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		after := time.Now().Add(time.Second)

		if u.ID == uuid.Nil {
			t.Error("expected a generated id")
		}
		if u.CreatedAt.Before(before) || u.CreatedAt.After(after) {
			t.Errorf("CreatedAt %v outside [%v, %v]", u.CreatedAt, before, after)
		}
		if u.Email != "a@example.com" || u.PasswordHash != "hash" {
			t.Errorf("fields not preserved: %+v", u)
		}
	})

	t.Run("get by email and by id roundtrip", func(t *testing.T) {
		repo := mk(t)
		created, err := repo.CreateUser(ctx, models.User{Email: "b@example.com", PasswordHash: "h"})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		byEmail, err := repo.GetUserByEmail(ctx, "b@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail: %v", err)
		}
		byID, err := repo.GetUserByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}

		if byEmail.ID != created.ID || byID.ID != created.ID {
			t.Errorf("ids differ: created=%s byEmail=%s byID=%s", created.ID, byEmail.ID, byID.ID)
		}
		if byEmail.PasswordHash != "h" {
			t.Errorf("password hash not returned, got %q", byEmail.PasswordHash)
		}
	})

	t.Run("unknown lookups return ErrUserNotFound", func(t *testing.T) {
		repo := mk(t)
		if _, err := repo.GetUserByEmail(ctx, "missing@example.com"); !errors.Is(err, ErrUserNotFound) {
			t.Errorf("GetUserByEmail unknown: got %v", err)
		}
		if _, err := repo.GetUserByID(ctx, uuid.New()); !errors.Is(err, ErrUserNotFound) {
			t.Errorf("GetUserByID unknown: got %v", err)
		}
	})

	t.Run("duplicate email returns ErrEmailTaken", func(t *testing.T) {
		repo := mk(t)
		if _, err := repo.CreateUser(ctx, models.User{Email: "dup@example.com", PasswordHash: "h"}); err != nil {
			t.Fatalf("first CreateUser: %v", err)
		}
		_, err := repo.CreateUser(ctx, models.User{Email: "dup@example.com", PasswordHash: "h2"})
		if !errors.Is(err, ErrEmailTaken) {
			t.Errorf("second CreateUser: expected ErrEmailTaken, got %v", err)
		}
	})
}

func TestUserRepositoryContract_InMemory(t *testing.T) {
	testUserRepositoryContract(t, func(*testing.T) UserRepository {
		return newInMemoryUserRepo()
	})
}

func TestUserRepositoryContract_SQLite(t *testing.T) {
	testUserRepositoryContract(t, func(t *testing.T) UserRepository {
		return newSQLiteUserRepo(newTestDB(t))
	})
}
