-- +goose Up
CREATE TABLE lists (
    id         TEXT    NOT NULL PRIMARY KEY,
    owner_id   TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    is_inbox   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL
);

-- One inbox per owner, enforced by the database rather than just the app.
CREATE UNIQUE INDEX idx_lists_one_inbox_per_owner ON lists (owner_id) WHERE is_inbox = 1;

CREATE TABLE list_members (
    list_id TEXT NOT NULL REFERENCES lists (id),
    user_id TEXT NOT NULL,
    PRIMARY KEY (list_id, user_id)
);

CREATE INDEX idx_list_members_user ON list_members (user_id);

-- +goose Down
DROP TABLE list_members;
DROP TABLE lists;
