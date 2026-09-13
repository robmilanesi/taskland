-- +goose Up
ALTER TABLE tasks ADD COLUMN list_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_tasks_list_created ON tasks (list_id, created_at);

-- +goose Down
DROP INDEX idx_tasks_list_created;
ALTER TABLE tasks DROP COLUMN list_id;
