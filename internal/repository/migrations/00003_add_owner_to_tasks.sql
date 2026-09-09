-- +goose Up
ALTER TABLE tasks ADD COLUMN owner_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_tasks_owner_created ON tasks (owner_id, created_at);

-- +goose Down
DROP INDEX idx_tasks_owner_created;
ALTER TABLE tasks DROP COLUMN owner_id;
