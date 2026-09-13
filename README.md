# Taskland
![CI](https://github.com/robmilanesi/taskland/actions/workflows/ci.yml/badge.svg)
![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/robmilanesi/91679ad8a726a68141479190594bc9cc/raw/taskland-coverage.json)

A task management application, yep another one!

# Features
- [x] Task management
- [x] Lists — grouping and sharing, see below
- [ ] Priority — *set & validated on create/update; no sorting or filtering yet*
- [x] Timing set based on time word expressions (Tomorrow, next monday, etc...)
- [ ] Kanban view
- [ ] Calendar view
- [ ] Today view
- [ ] Gamification

# API

Base path: `/api/v1`

| Method   | Path             | Auth   | Description |
|----------|------------------|--------|-------------|
| `POST`   | `/auth/register` | public | Create an account. Body: `email`, `password` (min 8 chars). Returns `{id, email}`. Also creates the account's `Inbox` list |
| `POST`   | `/auth/login`    | public | Exchange credentials for a token. Body: `email`, `password`. Returns `{token}` |
| `GET`    | `/tasks`         | bearer | List the caller's tasks, paginated (`?page=`, `?size=`, defaults `1` / `20`) |
| `POST`   | `/tasks`         | bearer | Create a task. Body: `title` (required), `description`, `priority` (`0`–`3`), `due_date` (RFC 3339 or natural language, see below), `list_id` (defaults to the caller's `Inbox`) |
| `GET`    | `/tasks/{id}`    | bearer | Fetch a task from any list the caller belongs to |
| `PATCH`  | `/tasks/{id}`    | bearer | Partial update. Any of `title`, `description`, `completed`, `priority`, `due_date`, `list_id` |
| `DELETE` | `/tasks/{id}`    | bearer | Delete a task (`204 No Content`) |
| `GET`    | `/lists`         | bearer | List every list the caller belongs to (owned or shared) |
| `POST`   | `/lists`         | bearer | Create a list. Body: `name` (required) |
| `GET`    | `/lists/{id}`    | bearer | Fetch a list the caller belongs to |
| `PATCH`  | `/lists/{id}`    | bearer | Rename a list. Body: `name`. Owner only |
| `DELETE` | `/lists/{id}`    | bearer | Delete a list (`204 No Content`). Owner only; the `Inbox` list can't be deleted |
| `GET`    | `/lists/{id}/members`           | bearer | List a list's members as `[{id, email}]`. Any member |
| `POST`   | `/lists/{id}/members`           | bearer | Share the list with another user. Body: `email`. Owner only |
| `DELETE` | `/lists/{id}/members/{userId}`  | bearer | Remove a member (`204 No Content`). Owner may remove anyone but themselves; a member may only remove themselves |

Every endpoint but `/auth/*` requires an `Authorization: Bearer <token>`
header, where `<token>` comes from `/auth/login`. A list or task the caller
can't see (wrong list, not a member) is reported as `404`; a list-management
action the caller isn't allowed to take (rename, delete, add/remove a member)
as `403`.

Errors are returned as `{"error": "..."}` with the matching HTTP status.

# Due dates

`due_date` accepts either an RFC 3339 timestamp or an English natural-language
expression, evaluated at request time:

- `today`, `tomorrow`
- a weekday name (`monday`) — the next occurrence of that day, never today itself
- `next <weekday>` (`next monday`) — a full week after the plain weekday above
- `in N days` (`in 5 days`)

Expressions without an explicit time resolve to midnight UTC of the resulting
date. An expression that matches none of the above and isn't valid RFC 3339 is
rejected with `400`.

# Lists

Every task belongs to exactly one list — there is no such thing as a "loose"
task. Every account gets an auto-created `Inbox` list at registration, which
can't be deleted, and is where a task lands if `list_id` is omitted on create.

A list has one owner (its creator) and any number of members. The permission
model is intentionally binary, with no roles:

- Only the owner can rename or delete the list, or add/remove members.
- Any member — owner or not — has full read/write access to the list's tasks.

This makes a list the natural unit for sharing: e.g. a "Spesa" (shopping)
list shared between two people, where both can freely add, edit and complete
tasks, but only the person who created it can rename it, delete it, or decide
who else is on it.

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