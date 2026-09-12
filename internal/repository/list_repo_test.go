package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// testListRepositoryContract runs the behavioural checks that every
// ListRepository implementation must satisfy. mk must return a fresh, empty
// repository each time it is called.
func testListRepositoryContract(t *testing.T, mk func(t *testing.T) ListRepository) {
	t.Helper()

	ctx := context.Background()

	t.Run("CreateList assigns id and membership for the owner", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()

		list, err := repo.CreateList(ctx, owner, "Groceries")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if list.ID == uuid.Nil {
			t.Error("expected a generated id")
		}
		if list.OwnerID != owner {
			t.Errorf("OwnerID = %s, want %s", list.OwnerID, owner)
		}
		if list.IsInbox {
			t.Error("CreateList must not produce an inbox list")
		}

		if _, err := repo.GetListByID(ctx, owner, list.ID.String()); err != nil {
			t.Errorf("owner cannot fetch their own list: %v", err)
		}
		if member, err := repo.IsMember(ctx, owner, list.ID); err != nil || !member {
			t.Errorf("IsMember(owner) = %v, %v, want true, nil", member, err)
		}
	})

	t.Run("GetListByID hides lists the user is not a member of", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()
		stranger := uuid.New()

		list, err := repo.CreateList(ctx, owner, "Private")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		if _, err := repo.GetListByID(ctx, stranger, list.ID.String()); !errors.Is(err, ErrListNotFound) {
			t.Errorf("expected ErrListNotFound, got %v", err)
		}
	})

	t.Run("GetByID unknown id returns ErrListNotFound", func(t *testing.T) {
		repo := mk(t)
		if _, err := repo.GetListByID(ctx, uuid.New(), uuid.NewString()); !errors.Is(err, ErrListNotFound) {
			t.Errorf("expected ErrListNotFound, got %v", err)
		}
		if _, err := repo.GetListByID(ctx, uuid.New(), "not-a-uuid"); !errors.Is(err, ErrListNotFound) {
			t.Errorf("expected ErrListNotFound for a malformed id, got %v", err)
		}
	})

	t.Run("GetAllLists returns only lists the user belongs to, in creation order", func(t *testing.T) {
		repo := mk(t)
		me := uuid.New()
		other := uuid.New()

		var ids []uuid.UUID
		for _, name := range []string{"a", "b", "c"} {
			list, err := repo.CreateList(ctx, me, name)
			if err != nil {
				t.Fatalf("CreateList(%q): %v", name, err)
			}
			ids = append(ids, list.ID)
			time.Sleep(2 * time.Millisecond)
		}
		if _, err := repo.CreateList(ctx, other, "not mine"); err != nil {
			t.Fatalf("CreateList for other owner: %v", err)
		}

		got, err := repo.GetAllLists(ctx, me)
		if err != nil {
			t.Fatalf("GetAllLists: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 lists, got %d", len(got))
		}
		for i, list := range got {
			if list.ID != ids[i] {
				t.Errorf("position %d = %s, want %s", i, list.ID, ids[i])
			}
		}
	})

	t.Run("UpdateList renames for the owner and persists", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()
		list, err := repo.CreateList(ctx, owner, "old name")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		updated, err := repo.UpdateList(ctx, owner, list.ID.String(), "new name")
		if err != nil {
			t.Fatalf("UpdateList: %v", err)
		}
		if updated.Name != "new name" {
			t.Errorf("Name = %q, want %q", updated.Name, "new name")
		}

		got, err := repo.GetListByID(ctx, owner, list.ID.String())
		if err != nil {
			t.Fatalf("GetListByID after update: %v", err)
		}
		if got.Name != "new name" {
			t.Errorf("persisted name = %q, want %q", got.Name, "new name")
		}
	})

	t.Run("UpdateList forbidden for a non-owner member", func(t *testing.T) {
		repo := mk(t)
		owner, member := uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := repo.AddMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		if _, err := repo.UpdateList(ctx, member, list.ID.String(), "renamed"); !errors.Is(err, ErrListForbidden) {
			t.Errorf("expected ErrListForbidden, got %v", err)
		}
	})

	t.Run("DeleteList removes the list for the owner", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()
		list, err := repo.CreateList(ctx, owner, "goner")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		if err := repo.DeleteList(ctx, owner, list.ID.String()); err != nil {
			t.Fatalf("DeleteList: %v", err)
		}
		if _, err := repo.GetListByID(ctx, owner, list.ID.String()); !errors.Is(err, ErrListNotFound) {
			t.Errorf("expected ErrListNotFound after delete, got %v", err)
		}
	})

	t.Run("DeleteList forbidden for a non-owner member", func(t *testing.T) {
		repo := mk(t)
		owner, member := uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := repo.AddMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		if err := repo.DeleteList(ctx, member, list.ID.String()); !errors.Is(err, ErrListForbidden) {
			t.Errorf("expected ErrListForbidden, got %v", err)
		}
	})

	t.Run("DeleteList refuses the inbox even for its owner", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()
		inbox, err := repo.InboxFor(ctx, owner)
		if err != nil {
			t.Fatalf("InboxFor: %v", err)
		}

		if err := repo.DeleteList(ctx, owner, inbox.ID.String()); !errors.Is(err, ErrCannotDeleteInbox) {
			t.Errorf("expected ErrCannotDeleteInbox, got %v", err)
		}
	})

	t.Run("AddMember lets the new member see the list, duplicate is rejected", func(t *testing.T) {
		repo := mk(t)
		owner, friend := uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		if err := repo.AddMember(ctx, owner, list.ID.String(), friend); err != nil {
			t.Fatalf("AddMember: %v", err)
		}
		if _, err := repo.GetListByID(ctx, friend, list.ID.String()); err != nil {
			t.Errorf("new member cannot see the list: %v", err)
		}

		if err := repo.AddMember(ctx, owner, list.ID.String(), friend); !errors.Is(err, ErrAlreadyMember) {
			t.Errorf("expected ErrAlreadyMember, got %v", err)
		}
	})

	t.Run("AddMember forbidden for a non-owner member", func(t *testing.T) {
		repo := mk(t)
		owner, member, stranger := uuid.New(), uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := repo.AddMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		if err := repo.AddMember(ctx, member, list.ID.String(), stranger); !errors.Is(err, ErrListForbidden) {
			t.Errorf("expected ErrListForbidden, got %v", err)
		}
	})

	t.Run("RemoveMember: owner removes another member", func(t *testing.T) {
		repo := mk(t)
		owner, member := uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := repo.AddMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		if err := repo.RemoveMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("RemoveMember: %v", err)
		}
		if _, err := repo.GetListByID(ctx, member, list.ID.String()); !errors.Is(err, ErrListNotFound) {
			t.Errorf("removed member should lose access, got %v", err)
		}
	})

	t.Run("RemoveMember: a member can leave on their own", func(t *testing.T) {
		repo := mk(t)
		owner, member := uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := repo.AddMember(ctx, owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		if err := repo.RemoveMember(ctx, member, list.ID.String(), member); err != nil {
			t.Fatalf("self-removal should be allowed: %v", err)
		}
	})

	t.Run("RemoveMember: a member cannot remove someone else", func(t *testing.T) {
		repo := mk(t)
		owner, a, b := uuid.New(), uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		for _, m := range []uuid.UUID{a, b} {
			if err := repo.AddMember(ctx, owner, list.ID.String(), m); err != nil {
				t.Fatalf("AddMember: %v", err)
			}
		}

		if err := repo.RemoveMember(ctx, a, list.ID.String(), b); !errors.Is(err, ErrListForbidden) {
			t.Errorf("expected ErrListForbidden, got %v", err)
		}
	})

	t.Run("RemoveMember: the owner cannot be removed, not even by themselves", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		if err := repo.RemoveMember(ctx, owner, list.ID.String(), owner); !errors.Is(err, ErrCannotRemoveOwner) {
			t.Errorf("expected ErrCannotRemoveOwner, got %v", err)
		}
	})

	t.Run("ListMembers returns the owner and every added member", func(t *testing.T) {
		repo := mk(t)
		owner, a, b := uuid.New(), uuid.New(), uuid.New()
		list, err := repo.CreateList(ctx, owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		for _, m := range []uuid.UUID{a, b} {
			if err := repo.AddMember(ctx, owner, list.ID.String(), m); err != nil {
				t.Fatalf("AddMember: %v", err)
			}
		}

		members, err := repo.ListMembers(ctx, owner, list.ID.String())
		if err != nil {
			t.Fatalf("ListMembers: %v", err)
		}
		want := map[uuid.UUID]bool{owner: true, a: true, b: true}
		if len(members) != len(want) {
			t.Fatalf("got %d members, want %d", len(members), len(want))
		}
		for _, id := range members {
			if !want[id] {
				t.Errorf("unexpected member %s", id)
			}
		}
	})

	t.Run("InboxFor creates once and is stable across calls", func(t *testing.T) {
		repo := mk(t)
		owner := uuid.New()

		first, err := repo.InboxFor(ctx, owner)
		if err != nil {
			t.Fatalf("InboxFor (first): %v", err)
		}
		if !first.IsInbox || first.OwnerID != owner {
			t.Errorf("unexpected inbox: %+v", first)
		}

		second, err := repo.InboxFor(ctx, owner)
		if err != nil {
			t.Fatalf("InboxFor (second): %v", err)
		}
		if second.ID != first.ID {
			t.Errorf("InboxFor created a second list: %s != %s", second.ID, first.ID)
		}

		if member, err := repo.IsMember(ctx, owner, first.ID); err != nil || !member {
			t.Errorf("owner should be a member of their own inbox: %v, %v", member, err)
		}
	})
}

func TestListRepositoryContract_InMemory(t *testing.T) {
	testListRepositoryContract(t, func(*testing.T) ListRepository {
		return newInMemoryListRepo()
	})
}

func TestListRepositoryContract_SQLite(t *testing.T) {
	testListRepositoryContract(t, func(t *testing.T) ListRepository {
		return newSQLiteListRepo(newTestDB(t))
	})
}
