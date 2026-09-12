package repository

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

type inMemoryListRepository struct {
	lists map[uuid.UUID]models.List
	// members maps a list id to the set of user ids that belong to it.
	members map[uuid.UUID]map[uuid.UUID]bool
}

func newInMemoryListRepo() *inMemoryListRepository {
	return &inMemoryListRepository{
		lists:   map[uuid.UUID]models.List{},
		members: map[uuid.UUID]map[uuid.UUID]bool{},
	}
}

func (r *inMemoryListRepository) isMember(listID, userID uuid.UUID) bool {
	return r.members[listID][userID]
}

func (r *inMemoryListRepository) create(ownerID uuid.UUID, name string, isInbox bool) models.List {
	list := models.List{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		IsInbox:   isInbox,
		CreatedAt: time.Now(),
	}
	r.lists[list.ID] = list
	r.members[list.ID] = map[uuid.UUID]bool{ownerID: true}
	return list
}

func (r *inMemoryListRepository) CreateList(ctx context.Context, ownerID uuid.UUID, name string) (models.List, error) {
	if err := ctx.Err(); err != nil {
		return models.List{}, err
	}
	return r.create(ownerID, name, false), nil
}

func (r *inMemoryListRepository) GetListByID(ctx context.Context, userID uuid.UUID, id string) (models.List, error) {
	if err := ctx.Err(); err != nil {
		return models.List{}, err
	}

	listID, err := uuid.Parse(id)
	if err != nil {
		return models.List{}, newErrListNotFound(id)
	}

	list, ok := r.lists[listID]
	if !ok || !r.isMember(listID, userID) {
		return models.List{}, newErrListNotFound(id)
	}
	return list, nil
}

func (r *inMemoryListRepository) GetAllLists(ctx context.Context, userID uuid.UUID) ([]models.List, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	lists := []models.List{}
	for listID, list := range r.lists {
		if r.isMember(listID, userID) {
			lists = append(lists, list)
		}
	}
	slices.SortFunc(lists, func(a, b models.List) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return lists, nil
}

func (r *inMemoryListRepository) UpdateList(ctx context.Context, userID uuid.UUID, id string, name string) (models.List, error) {
	list, err := r.GetListByID(ctx, userID, id)
	if err != nil {
		return models.List{}, err
	}
	if list.OwnerID != userID {
		return models.List{}, newErrListForbidden(id)
	}

	list.Name = name
	r.lists[list.ID] = list
	return list, nil
}

func (r *inMemoryListRepository) DeleteList(ctx context.Context, userID uuid.UUID, id string) error {
	list, err := r.GetListByID(ctx, userID, id)
	if err != nil {
		return err
	}
	if list.OwnerID != userID {
		return newErrListForbidden(id)
	}
	if list.IsInbox {
		return ErrCannotDeleteInbox
	}

	delete(r.lists, list.ID)
	delete(r.members, list.ID)
	return nil
}

func (r *inMemoryListRepository) AddMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error {
	list, err := r.GetListByID(ctx, userID, listID)
	if err != nil {
		return err
	}
	if list.OwnerID != userID {
		return newErrListForbidden(listID)
	}
	if r.isMember(list.ID, memberID) {
		return ErrAlreadyMember
	}

	r.members[list.ID][memberID] = true
	return nil
}

func (r *inMemoryListRepository) RemoveMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error {
	list, err := r.GetListByID(ctx, userID, listID)
	if err != nil {
		return err
	}
	if memberID == list.OwnerID {
		return ErrCannotRemoveOwner
	}
	if userID != memberID && userID != list.OwnerID {
		return newErrListForbidden(listID)
	}
	if !r.isMember(list.ID, memberID) {
		return ErrNotAMember
	}

	delete(r.members[list.ID], memberID)
	return nil
}

func (r *inMemoryListRepository) ListMembers(ctx context.Context, userID uuid.UUID, listID string) ([]uuid.UUID, error) {
	list, err := r.GetListByID(ctx, userID, listID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(r.members[list.ID]))
	for id := range r.members[list.ID] {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return cmp.Compare(a.String(), b.String())
	})
	return ids, nil
}

func (r *inMemoryListRepository) IsMember(ctx context.Context, userID uuid.UUID, listID uuid.UUID) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return r.isMember(listID, userID), nil
}

func (r *inMemoryListRepository) InboxFor(ctx context.Context, userID uuid.UUID) (models.List, error) {
	if err := ctx.Err(); err != nil {
		return models.List{}, err
	}

	for _, list := range r.lists {
		if list.OwnerID == userID && list.IsInbox {
			return list, nil
		}
	}
	return r.create(userID, "Inbox", true), nil
}
