# Taskland
A task management application, yep another one!

# Features
- [ ] Task management
- [ ] Support for projects
- [ ] Priority
- [ ] Timing set based on time word expressions (Tomorrow, last week, etc...)
- [ ] Kanban view
- [ ] Calendar view
- [ ] Today view
- [ ] Gamification

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