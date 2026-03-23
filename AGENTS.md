# Repository Guidelines

## Project Structure & Module Organization
- `cmd/clash-tui/main.go`: program entrypoint; bootstraps config and starts Bubble Tea.
- `internal/config/`: local config load/save (`~/.config/clash-tui/config.yaml`), defaults, env overrides.
- `internal/mihomo/`: Mihomo API client (REST + `/logs` websocket) and response types.
- `internal/ui/`: TUI model/update/view, key/mouse interactions, layout and styles.
- `docs/`: implementation notes and file index (`IMPLEMENTATION.md`, `FILE_INDEX.md`).

Keep business logic in `internal/*`; keep `cmd/*` thin.

## Build, Test, and Development Commands
- `go mod tidy`: sync and clean module dependencies.
- `go build ./...`: compile all packages; run before every PR.
- `go run ./cmd/clash-tui`: run locally in terminal UI mode.
- `gofmt -w ./...` (or targeted files): format Go code.

Typical loop:
```bash
gofmt -w internal/ui/app.go
go build ./...
go run ./cmd/clash-tui
```

## Coding Style & Naming Conventions
- Language: Go (`go1.22+` in this repo setup).
- Formatting: always `gofmt`; do not submit unformatted code.
- Indentation: tabs (Go default), not manual alignment.
- Naming:
  - exported symbols: `PascalCase`
  - internal helpers: `camelCase`
  - files: short lowercase names (e.g., `client.go`, `types.go`)
- Keep UI event handlers small and explicit (`handleProxyKeys`, `handleProxyMouse`).

## Testing Guidelines
- No full test suite is present yet. Minimum requirement is successful build:
  - `go build ./...`
- When adding tests, use Go `testing` package and place `*_test.go` next to source files.
- Prefer table-driven tests for API parsing and state transitions.

## Commit & Pull Request Guidelines
- Use clear, scoped commit messages (recommended Conventional Commits):
  - `feat(ui): add clickable test-all button in proxies`
  - `fix(mihomo): parse providers.proxies object array`
- PRs should include:
  - what changed and why
  - impacted modules/files
  - verification steps (`go build ./...`, manual run flow)
  - TUI screenshots or short terminal recordings for UI changes

## Security & Configuration Tips
- Never commit real controller secrets or local profile data.
- Use env vars for local overrides: `MIHOMO_CONTROLLER`, `MIHOMO_SECRET`.
- Treat logs and exported configs as sensitive if they contain proxy/provider metadata.
