package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
)

// registerAccountTx creates the user, their inbox list and the corresponding
// membership row in a single transaction, so a failure partway through never
// leaves a user without an inbox or an inbox without its owner as a member.
func registerAccountTx(ctx context.Context, db *sql.DB, user models.User) (models.User, error) {
	user.ID = uuid.New()
	user.CreatedAt = time.Now().UTC()

	inbox := models.List{
		ID:        uuid.New(),
		OwnerID:   user.ID,
		Name:      "Inbox",
		IsInbox:   true,
		CreatedAt: user.CreatedAt,
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return models.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO users ("+userColumns+") VALUES (?, ?, ?, ?)",
		user.ID.String(), user.Email, user.PasswordHash, formatTime(user.CreatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO lists ("+listColumns+") VALUES (?, ?, ?, ?, ?)",
		inbox.ID.String(), inbox.OwnerID.String(), inbox.Name, inbox.IsInbox, formatTime(inbox.CreatedAt),
	)
	if err != nil {
		return models.User{}, fmt.Errorf("insert inbox list: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO list_members (list_id, user_id) VALUES (?, ?)", inbox.ID.String(), user.ID.String(),
	)
	if err != nil {
		return models.User{}, fmt.Errorf("insert inbox membership: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return models.User{}, fmt.Errorf("commit: %w", err)
	}
	return user, nil
}
