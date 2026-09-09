# Taskland
![CI](https://github.com/robmilanesi/taskland/actions/workflows/ci.yml/badge.svg)
![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/robmilanesi/91679ad8a726a68141479190594bc9cc/raw/taskland-coverage.json)

A task management application, yep another one!

# Features
- [x] Task management
- [ ] Support for projects
- [ ] Priority — *set & validated on create/update; no sorting or filtering yet*
- [ ] Timing set based on time word expressions (Tomorrow, last week, etc...)
- [ ] Kanban view
- [ ] Calendar view
- [ ] Today view
- [ ] Gamification

# API

Base path: `/api/v1`

| Method   | Path          | Description |
|----------|---------------|-------------|
| `GET`    | `/tasks`      | List tasks, paginated (`?page=`, `?size=`, defaults `1` / `20`) |
| `POST`   | `/tasks`      | Create a task. Body: `title` (required), `description`, `priority` (`0`–`3`), `due_date` (RFC 3339) |
| `GET`    | `/tasks/{id}` | Fetch a single task |
| `PATCH`  | `/tasks/{id}` | Partial update. Any of `title`, `description`, `completed`, `priority`, `due_date` |
| `DELETE` | `/tasks/{id}` | Delete a task (`204 No Content`) |

Errors are returned as `{"error": "..."}` with the matching HTTP status.

# Storage

Tasks are persisted in a local SQLite database via a pure-Go driver (no CGO).

- `TASKLAND_DB_PATH` selects the database file; it defaults to `taskland.db`
  in the working directory.
- The schema is created and kept up to date automatically on startup
  (embedded goose migrations) — there is no manual migration step.
- SQLite runs in WAL mode, so `<db>-wal` and `<db>-shm` files appear next to
  the database file.

The test suite runs against a throwaway database under a temp directory, so
`go test` needs no external services.

# Development

Tooling is pinned with [mise](https://mise.jdx.dev):

```sh
mise install      # install go, golangci-lint, lefthook at pinned versions
mise run setup    # install the git hooks (lefthook)
```

Once the hooks are installed, every commit runs `gofmt`/`goimports`,
`golangci-lint`, `go build` and `go test`; `git push` also runs the tests
with the race detector.

Useful tasks:

```sh
mise run fmt      # format the code
mise run lint     # run golangci-lint
mise run check    # run the full pre-commit gate on all files
```

# License
[GNU GPLv3](./LICENSE)