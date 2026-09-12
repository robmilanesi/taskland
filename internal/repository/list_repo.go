package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// ErrListNotFound is returned when a list lookup finds no matching record that
// userID is a member of.
var ErrListNotFound = errors.New("list not found")

// ErrListForbidden is returned when a member who is not the list's owner
// attempts to manage the list itself (rename, delete, membership).
var ErrListForbidden = errors.New("not allowed to manage this list")

// ErrCannotDeleteInbox is returned when the owner tries to delete their inbox.
var ErrCannotDeleteInbox = errors.New("cannot delete the inbox list")

// ErrAlreadyMember is returned by AddMember for a user who already belongs.
var ErrAlreadyMember = errors.New("user is already a member of this list")

// ErrNotAMember is returned by RemoveMember for a user who is not a member.
var ErrNotAMember = errors.New("user is not a member of this list")

// ErrCannotRemoveOwner is returned by RemoveMember for the list's owner: the
// owner is removed by deleting the list, not by leaving it.
var ErrCannotRemoveOwner = errors.New("cannot remove the list owner")

func newErrListNotFound(id string) error {
	return fmt.Errorf("list %q: %w", id, ErrListNotFound)
}

func newErrListForbidden(id string) error {
	return fmt.Errorf("list %q: %w", id, ErrListForbidden)
}

// ListRepository is the storage abstraction for lists and their membership.
type ListRepository interface {
	CreateList(ctx context.Context, ownerID uuid.UUID, name string) (models.List, error)
	GetListByID(ctx context.Context, userID uuid.UUID, id string) (models.List, error)
	GetAllLists(ctx context.Context, userID uuid.UUID) ([]models.List, error)
	UpdateList(ctx context.Context, userID uuid.UUID, id string, name string) (models.List, error)
	DeleteList(ctx context.Context, userID uuid.UUID, id string) error

	// AddMember and RemoveMember require userID to be the list's owner, except
	// that a member may always remove themselves ("leave"). The owner cannot be
	// removed; deleting the list is the deliberate way to give it up.
	AddMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error
	RemoveMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error
	ListMembers(ctx context.Context, userID uuid.UUID, listID string) ([]uuid.UUID, error)

	// IsMember and InboxFor are used internally (by TaskRepository and account
	// registration) and are not scoped the way the methods above are.
	IsMember(ctx context.Context, userID uuid.UUID, listID uuid.UUID) (bool, error)

	// InboxFor returns userID's default list, creating it (name "Inbox",
	// IsInbox true, userID as owner and sole member) on first call.
	InboxFor(ctx context.Context, userID uuid.UUID) (models.List, error)
}
