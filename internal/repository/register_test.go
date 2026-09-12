package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// testRegisterAccount runs the behavioural checks RegisterAccount must satisfy
// regardless of backend. mk must return a fresh store each time it is called.
func testRegisterAccount(t *testing.T, mk func(t *testing.T) *Store) {
	t.Helper()

	ctx := context.Background()

	t.Run("creates the user and their inbox together", func(t *testing.T) {
		store := mk(t)

		user, err := store.RegisterAccount(ctx, models.User{Email: "a@example.com", PasswordHash: "hash"})
		if err != nil {
			t.Fatalf("RegisterAccount: %v", err)
		}
		if user.ID == uuid.Nil {
			t.Fatal("expected a generated id")
		}

		fetched, err := store.Users.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if fetched.Email != "a@example.com" {
			t.Errorf("email = %q, want %q", fetched.Email, "a@example.com")
		}

		inbox, err := store.Lists.InboxFor(ctx, user.ID)
		if err != nil {
			t.Fatalf("InboxFor: %v", err)
		}
		if !inbox.IsInbox || inbox.OwnerID != user.ID {
			t.Errorf("unexpected inbox: %+v", inbox)
		}
		if member, err := store.Lists.IsMember(ctx, user.ID, inbox.ID); err != nil || !member {
			t.Errorf("user should be a member of their own inbox: %v, %v", member, err)
		}
	})

	t.Run("duplicate email returns ErrEmailTaken", func(t *testing.T) {
		store := mk(t)

		if _, err := store.RegisterAccount(ctx, models.User{Email: "dup@example.com", PasswordHash: "h"}); err != nil {
			t.Fatalf("first RegisterAccount: %v", err)
		}
		if _, err := store.RegisterAccount(ctx, models.User{Email: "dup@example.com", PasswordHash: "h2"}); !errors.Is(err, ErrEmailTaken) {
			t.Errorf("expected ErrEmailTaken, got %v", err)
		}
	})
}

func TestRegisterAccount_InMemory(t *testing.T) {
	testRegisterAccount(t, func(t *testing.T) *Store {
		store, err := NewStore(Config{Type: TaskRepoInMemory})
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		return store
	})
}

func TestRegisterAccount_SQLite(t *testing.T) {
	testRegisterAccount(t, func(t *testing.T) *Store {
		store, err := NewStore(Config{Type: TaskRepoSQLite, DSN: t.TempDir() + "/register.db"})
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })
		return store
	})
}
