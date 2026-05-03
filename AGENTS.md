# Repository Guidelines

## Project Structure & Module Organization

This is the `pooly/services/poold` Go module. It builds two binaries:

- `cmd/poold`: the daemon that polls the Intex spa, persists state, runs schedules, and exposes the HTTP API.
- `cmd/poolctl`: a client CLI for status, commands, and event watching.

Shared code lives under `internal/`: `config` for settings, `httpapi` for handlers and service logic, `pool` for domain types, `protocol/intex` for spa protocol parsing, `scheduler` for plan logic, and `store` for SQLite persistence. OpenWrt service files are in `packaging/openwrt/`. Build artifacts go in `dist/` and should not be edited by hand.

## Build, Test, and Development Commands

- `go test ./...`: run all unit tests.
- `go run ./cmd/poold`: start the daemon with defaults.
- `go run ./cmd/poolctl -- status`: query a locally running daemon.
- `go build -o dist/poold ./cmd/poold`: build the daemon binary.
- `go build -o dist/poolctl ./cmd/poolctl`: build the CLI binary.
- `mise run poold:test`, `mise run poold:run`, `mise run poold:build:all`: workspace task aliases from `README.md`.

Useful defaults are `POOLD_LISTEN_ADDR=127.0.0.1:8090`, `POOLD_POOL_ADDR=127.0.0.1:8990`, `POOLD_DB_PATH=./var/poold.db`, and `POOLD_TOKEN=dev-token`.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed `.go` files before committing. Keep package names short and lowercase. Document exported identifiers at package boundaries; otherwise prefer unexported helpers. Follow existing organization: handlers in `handler.go`, service behavior in `service.go`, protocol code under `internal/protocol/intex`, and tests beside the code they cover.

## Testing Guidelines

Tests use Go's standard `testing` package and are named `*_test.go` next to the implementation. Prefer table-driven tests for protocol parsing, scheduler decisions, and API cases. Update tests for behavior changes around polling intervals, schedule readiness, SQLite state, or HTTP authorization. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines

Recent commits use short imperative subjects, for example `Embed timezone data` and `Tail poolctl watch by default`. Keep the first line focused on the behavior change; add body text only when context or tradeoffs matter.

Pull requests should include a short description, test results such as `go test ./...`, and deployment notes for OpenWrt or environment-variable changes. Link related issues when applicable. Include terminal output for CLI changes when it clarifies the review.

## Security & Configuration Tips

Do not commit real bearer tokens, Tailscale addresses, production SQLite databases, or device-specific pool addresses. Use environment variables or local shell configuration for secrets. Document any new required `POOLD_*` setting in `README.md`.
