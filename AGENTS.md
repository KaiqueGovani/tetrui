# AGENTS.md

Operational guide for coding agents working in `tetrui`.

## Project Snapshot
- Language: Go (`go 1.25` in `go.mod`)
- Type: terminal UI game (Bubble Tea + Lipgloss)
- Entrypoint: `cmd/tetrui/main.go`
- Module path: `tetrui`
- CI workflow: `.github/workflows/release.yml`

## Build / Run / Lint / Test Commands

### Install dependencies
```bash
go mod download
```

Optional cleanup:
```bash
go mod tidy
```

### Run locally (dev)
```bash
go run ./cmd/tetrui
```

Run with debug log output:
```bash
go run ./cmd/tetrui --debug
```

### Build locally
```bash
go build -o tetrui ./cmd/tetrui
```

### Build with CI-like ldflags
```bash
TETRUI_SCORE_API_URL="https://example" TETRUI_SCORE_API_KEY="key" \
go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.defaultScoreAPIURL=${TETRUI_SCORE_API_URL} -X main.defaultScoreAPIKey=${TETRUI_SCORE_API_KEY}" \
  -o dist/tetrui-$(go env GOOS)-$(go env GOARCH) ./cmd/tetrui
```

### Lint / static checks
```bash
go vet ./...
```

### Formatting
```bash
gofmt -w cmd/tetrui/*.go
```

### Tests (current and future)
Current state: no `_test.go` files in repository.

Run all tests (when tests are added):
```bash
go test ./...
```

Run a single package:
```bash
go test ./path/to/package
```

Run a single test function:
```bash
go test -run '^TestName$' ./path/to/package
```

Run one test file indirectly (Go runs by package):
```bash
go test -run 'TestNamesFromThatFile' ./path/to/package
```

Run all test names matching a regex:
```bash
go test -run 'TestNamePrefix|TestOtherName' ./path/to/package
```

## CI Notes
- CI builds on Ubuntu, macOS, Windows.
- Linux CI installs ALSA headers (`libasound2-dev`) before building.
- Release workflow publishes a `nightly` prerelease from `master`.
- There is no CI test step yet; only build + artifact/release.

## Coding Style (Observed + Required)

### Package and file layout
- Keep package as `main` for this repo unless explicitly restructuring.
- Keep concerns split by file domain (`game.go`, `render.go`, `sync.go`, etc.).
- Prefer adding to existing domain files before creating new files when possible.

### Imports
- Use standard Go grouping:
  1) stdlib
  2) blank line
  3) third-party imports
- Use alias only when common/needed for clarity (example: Bubble Tea as `tea`).
- Avoid unnecessary aliases.

### Formatting and whitespace
- `gofmt` is required.
- Tabs for indentation (Go default).
- Keep lines readable; avoid dense one-liners unless idiomatic.

### Types and data modeling
- Use explicit structs for state and payloads (`Model`, `Game`, `ScoreEntry`, API DTOs).
- Use typed enums with `iota` for bounded state (`Screen`, `SoundEvent`, `MusicMode`).
- Keep exported fields/types only where needed; default to unexported internals.

### Naming conventions
- Types: PascalCase (`ScoreSync`, `LockResult`).
- Functions/methods: camelCase (`loadConfig`, `updateGame`, `FetchScoresCmd`).
- Constants: camelCase for internal constants (`boardWidth`, `scoresPageSize`).
- Acronyms follow Go conventions where practical (`API`, `URL` in identifiers are acceptable as used).

### Error handling
- Return errors from lower-level functions; handle at call sites.
- Bubble Tea async commands should return message structs carrying `err error` when needed.
- For optional/best-effort operations (debug logging, config writes), ignoring errors with `_ = ...` is acceptable only when failure is non-fatal.
- Never panic for expected runtime/network/config failures.
- Prefer graceful degradation (missing score API URL disables sync; audio init failures log and continue).

### Control flow and state updates
- Follow Bubble Tea update pattern: pure-ish `Update` switch on message type, return `(model, cmd)`.
- Keep command batching explicit with `tea.Batch`.
- Keep screen transitions via helper (`setScreen`) to centralize side effects (music sync).
- Clamp user-configurable values (`scale`, `volume`) before storing/using.

### Concurrency and shared state
- Protect shared mutable state with mutexes (`sync.Mutex`, `sync.RWMutex`) as in audio/music/debug.
- Use `sync.Once` for one-time initialization (`initAudioContext`).
- Ensure goroutines have clear stop conditions/channels.

### Persistence and compatibility
- Config and scores live under `os.UserConfigDir()/tetrui`.
- Maintain backward compatibility for config by applying defaults if fields are missing.
- Keep JSON tags stable for persisted and API structs.

### Networking and API integration
- Use `http.Client` with timeout.
- Treat non-2xx as errors (custom status error is already used).
- Include `X-Api-Key` only when configured.
- Keep API request/response mapping isolated (`apiScore`, `uploadScore`).

## Environment and Secrets
- Supported env vars:
  - `TETRUI_SCORE_API_URL`
  - `TETRUI_SCORE_API_KEY`
- CI injects these via workflow secrets and ldflags defaults.
- Do not commit real API keys.

## Cursor / Copilot Rules
- No `.cursorrules` file found.
- No `.cursor/rules/` directory found.
- No `.github/copilot-instructions.md` found.

## Agent Execution Checklist
- Read touched files fully before editing.
- Keep edits minimal and in existing style.
- Run `gofmt -w` on changed Go files.
- Run `go vet ./...` after edits.
- Run `go test ./...` when tests exist or when adding tests.
- Do not commit unless explicitly requested.
