-- +goose Up
CREATE TABLE field_version_sequence (
    id INTEGER PRIMARY KEY CHECK (id=1),
    value INTEGER NOT NULL
) STRICT;

INSERT INTO field_version_sequence (id, value) VALUES (1, 0);

CREATE TABLE projects (
    id TEXT PRIMARY KEY CHECK(LENGTH(id) == 36),
    title TEXT NOT NULL CHECK(LENGTH(title) <= 100),
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL,
    deleted_at_ms INTEGER NULL,
    CHECK(id != '00000000-0000-0000-0000-000000000000' OR title = 'Inbox'),
    CHECK(id != '00000000-0000-0000-0000-000000000000' OR deleted_at_ms IS NULL)
) STRICT;

INSERT INTO projects (id, title, created_at_ms, updated_at_ms) VALUES ('00000000-0000-0000-0000-000000000000', 'Inbox', CAST(unixepoch('now', 'subsec') * 1000 as INTEGER),  CAST(unixepoch('now', 'subsec') * 1000 as INTEGER));

CREATE TABLE project_field_versions (
    project_id TEXT CHECK(LENGTH(project_id) == 36),
    field TEXT NOT NULL CHECK(field IN ('title', 'deleted_at_ms')),
    time_ms INTEGER NOT NULL,
    device_id TEXT NOT NULL,
    sequence_id INTEGER NOT NULL UNIQUE,
    PRIMARY KEY(project_id, field),
    FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE tasks (
    id TEXT PRIMARY KEY CHECK(LENGTH(id) == 36),
    title TEXT NOT NULL CHECK(LENGTH(title) <= 140),
    description TEXT NULL,
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL,
    priority INTEGER NOT NULL CHECK(priority IN (0, 1, 2, 3)),
    due_at_ms INTEGER NULL,
    completed_at_ms INTEGER NULL,
    deleted_at_ms INTEGER NULL,
    project_id TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id)
) STRICT;

CREATE INDEX idx_project_id ON tasks(project_id);

CREATE TABLE task_field_versions (
    task_id TEXT NOT NULL CHECK(LENGTH(task_id) == 36),
    field TEXT NOT NULL CHECK(field IN ('title', 'description', 'priority', 'due_at_ms', 'completed_at_ms', 'deleted_at_ms', 'project_id')),
    time_ms INTEGER NOT NULL,
    device_id TEXT NOT NULL,
    sequence_id INTEGER NOT NULL UNIQUE,
    PRIMARY KEY(task_id, field),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) STRICT;

-- +goose Down
DROP TABLE task_field_versions;
DROP TABLE tasks;
DROP TABLE project_field_versions;
DROP TABLE projects;
DROP TABLE field_version_sequence;
