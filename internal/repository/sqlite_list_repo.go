package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

const listColumns = "id, owner_id, name, is_inbox, created_at"
const listColumnsQualified = "l.id, l.owner_id, l.name, l.is_inbox, l.created_at"

type sqliteListRepository struct {
	db *sql.DB
}

func newSQLiteListRepo(db *sql.DB) *sqliteListRepository {
	return &sqliteListRepository{db: db}
}

func (r *sqliteListRepository) CreateList(ctx context.Context, ownerID uuid.UUID, name string) (models.List, error) {
	return r.createList(ctx, ownerID, name, false)
}

// createList inserts the list and its owner's membership in one transaction.
func (r *sqliteListRepository) createList(ctx context.Context, ownerID uuid.UUID, name string, isInbox bool) (models.List, error) {
	list := models.List{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		IsInbox:   isInbox,
		CreatedAt: time.Now().UTC(),
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.List{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO lists ("+listColumns+") VALUES (?, ?, ?, ?, ?)",
		list.ID.String(), list.OwnerID.String(), list.Name, list.IsInbox, formatTime(list.CreatedAt),
	)
	if err != nil {
		return models.List{}, fmt.Errorf("insert list: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO list_members (list_id, user_id) VALUES (?, ?)", list.ID.String(), ownerID.String(),
	)
	if err != nil {
		return models.List{}, fmt.Errorf("insert owner membership: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return models.List{}, fmt.Errorf("commit: %w", err)
	}
	return list, nil
}

func (r *sqliteListRepository) GetListByID(ctx context.Context, userID uuid.UUID, id string) (models.List, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+listColumnsQualified+" FROM lists l JOIN list_members m ON m.list_id = l.id WHERE l.id = ? AND m.user_id = ?",
		id, userID.String(),
	)

	list, err := scanList(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.List{}, newErrListNotFound(id)
	}
	if err != nil {
		return models.List{}, fmt.Errorf("get list %q: %w", id, err)
	}
	return list, nil
}

func (r *sqliteListRepository) GetAllLists(ctx context.Context, userID uuid.UUID) ([]models.List, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+listColumnsQualified+" FROM lists l JOIN list_members m ON m.list_id = l.id WHERE m.user_id = ? ORDER BY l.created_at",
		userID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list lists: %w", err)
	}
	defer func() { _ = rows.Close() }()

	lists := []models.List{}
	for rows.Next() {
		list, err := scanList(rows)
		if err != nil {
			return nil, fmt.Errorf("scan list row: %w", err)
		}
		lists = append(lists, list)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate list rows: %w", err)
	}
	return lists, nil
}

func (r *sqliteListRepository) UpdateList(ctx context.Context, userID uuid.UUID, id string, name string) (models.List, error) {
	list, err := r.GetListByID(ctx, userID, id)
	if err != nil {
		return models.List{}, err
	}
	if list.OwnerID != userID {
		return models.List{}, newErrListForbidden(id)
	}

	if _, err := r.db.ExecContext(ctx, "UPDATE lists SET name = ? WHERE id = ?", name, id); err != nil {
		return models.List{}, fmt.Errorf("update list %q: %w", id, err)
	}
	list.Name = name
	return list, nil
}

func (r *sqliteListRepository) DeleteList(ctx context.Context, userID uuid.UUID, id string) error {
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

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM list_members WHERE list_id = ?", id); err != nil {
		return fmt.Errorf("delete members of list %q: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM lists WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete list %q: %w", id, err)
	}
	return tx.Commit()
}

func (r *sqliteListRepository) AddMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error {
	list, err := r.GetListByID(ctx, userID, listID)
	if err != nil {
		return err
	}
	if list.OwnerID != userID {
		return newErrListForbidden(listID)
	}

	_, err = r.db.ExecContext(ctx,
		"INSERT INTO list_members (list_id, user_id) VALUES (?, ?)", listID, memberID.String(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyMember
		}
		return fmt.Errorf("add member to list %q: %w", listID, err)
	}
	return nil
}

func (r *sqliteListRepository) RemoveMember(ctx context.Context, userID uuid.UUID, listID string, memberID uuid.UUID) error {
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

	res, err := r.db.ExecContext(ctx,
		"DELETE FROM list_members WHERE list_id = ? AND user_id = ?", listID, memberID.String(),
	)
	if err != nil {
		return fmt.Errorf("remove member from list %q: %w", listID, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrNotAMember
	}
	return nil
}

func (r *sqliteListRepository) ListMembers(ctx context.Context, userID uuid.UUID, listID string) ([]uuid.UUID, error) {
	if _, err := r.GetListByID(ctx, userID, listID); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT user_id FROM list_members WHERE list_id = ? ORDER BY user_id", listID,
	)
	if err != nil {
		return nil, fmt.Errorf("list members of list %q: %w", listID, err)
	}
	defer func() { _ = rows.Close() }()

	ids := []uuid.UUID{}
	for rows.Next() {
		var idStr string
		if err := rows.Scan(&idStr); err != nil {
			return nil, fmt.Errorf("scan member id: %w", err)
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse member id %q: %w", idStr, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate member rows: %w", err)
	}
	return ids, nil
}

func (r *sqliteListRepository) IsMember(ctx context.Context, userID uuid.UUID, listID uuid.UUID) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM list_members WHERE list_id = ? AND user_id = ?", listID.String(), userID.String(),
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return n > 0, nil
}

func (r *sqliteListRepository) InboxFor(ctx context.Context, userID uuid.UUID) (models.List, error) {
	list, err := r.inbox(ctx, userID)
	if err == nil {
		return list, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.List{}, fmt.Errorf("get inbox for %s: %w", userID, err)
	}

	created, err := r.createList(ctx, userID, "Inbox", true)
	if err != nil {
		if isUniqueViolation(err) {
			// Lost a race with a concurrent InboxFor call for the same user.
			return r.inbox(ctx, userID)
		}
		return models.List{}, err
	}
	return created, nil
}

func (r *sqliteListRepository) inbox(ctx context.Context, userID uuid.UUID) (models.List, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+listColumns+" FROM lists WHERE owner_id = ? AND is_inbox = 1", userID.String(),
	)
	return scanList(row)
}

func scanList(s rowScanner) (models.List, error) {
	var (
		list       models.List
		idStr      string
		ownerIDStr string
		createdAt  string
	)

	err := s.Scan(&idStr, &ownerIDStr, &list.Name, &list.IsInbox, &createdAt)
	if err != nil {
		return models.List{}, err
	}

	if list.ID, err = uuid.Parse(idStr); err != nil {
		return models.List{}, fmt.Errorf("parse list id %q: %w", idStr, err)
	}
	if list.OwnerID, err = uuid.Parse(ownerIDStr); err != nil {
		return models.List{}, fmt.Errorf("parse list owner id %q: %w", ownerIDStr, err)
	}
	if list.CreatedAt, err = time.Parse(sqlTimeLayout, createdAt); err != nil {
		return models.List{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
	}

	return list, nil
}
