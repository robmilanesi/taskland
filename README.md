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

| Method   | Path             | Auth   | Description |
|----------|------------------|--------|-------------|
| `POST`   | `/auth/register` | public | Create an account. Body: `email`, `password` (min 8 chars). Returns `{id, email}` |
| `POST`   | `/auth/login`    | public | Exchange credentials for a token. Body: `email`, `password`. Returns `{token}` |
| `GET`    | `/tasks`         | bearer | List the caller's tasks, paginated (`?page=`, `?size=`, defaults `1` / `20`) |
| `POST`   | `/tasks`         | bearer | Create a task. Body: `title` (required), `description`, `priority` (`0`–`3`), `due_date` (RFC 3339) |
| `GET`    | `/tasks/{id}`    | bearer | Fetch one of the caller's tasks |
| `PATCH`  | `/tasks/{id}`    | bearer | Partial update. Any of `title`, `description`, `completed`, `priority`, `due_date` |
| `DELETE` | `/tasks/{id}`    | bearer | Delete a task (`204 No Content`) |

The `/tasks` endpoints require an `Authorization: Bearer <token>` header, where
`<token>` comes from `/auth/login`. Tasks are scoped to the authenticated user:
another user's task is reported as `404`.

Errors are returned as `{"error": "..."}` with the matching HTTP status.

# Configuration

The server reads its settings from the environment:

| Variable              | Default        | Notes |
|-----------------------|----------------|-------|
| `TASKLAND_ADDR`       | `:8080`        | Listen address |
| `TASKLAND_DB_PATH`    | `taskland.db`  | SQLite database file |
| `TASKLAND_JWT_SECRET` | —              | **Required.** HS256 signing key, at least 32 characters |
| `TASKLAND_JWT_TTL`    | `24h`          | Token lifetime (Go duration) |

The server refuses to start without a valid `TASKLAND_JWT_SECRET`, and shuts
down gracefully on `SIGINT` / `SIGTERM`.

# Storage

Tasks and accounts are persisted in a local SQLite database via a pure-Go
driver (no CGO).

- The schema is created and kept up to date automatically on startup (embedded
  goose migrations) — there is no manual migration step.
- SQLite runs in WAL mode, so `<db>-wal` and `<db>-shm` files appear next to the
  database file.

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