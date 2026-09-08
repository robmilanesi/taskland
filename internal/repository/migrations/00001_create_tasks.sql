-- +goose Up
CREATE TABLE tasks (
    id           TEXT    PRIMARY KEY,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    completed    INTEGER NOT NULL DEFAULT 0,
    completed_at TEXT,
    priority     INTEGER NOT NULL DEFAULT 0,
    due_date     TEXT,
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);

-- +goose Down
DROP TABLE tasks;
