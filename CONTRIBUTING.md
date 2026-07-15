# Contributing to unissh

Thanks for your interest! unissh is a small, friendly Go/TUI project.

## Development

```bash
git clone https://github.com/MrBenoit/unissh
cd unissh
make build        # fmt-check → vet → test → coverage gate → build
make ci           # everything CI runs: test → race → vet → coverage → build
make test-race    # tests with the race detector
make cover        # coverage report
make lint         # golangci-lint (needs it installed)
./unissh          # try it against a test VPS
```

`make build` refuses to build on failing tests, or when `internal/core` coverage drops below
**90%** (`scripts/coverage.sh`, override with `COVER_MIN`).

One Go module, two packages — `internal/core` (pure logic, tested ≥90%) and `internal/app`
(TUI/SSH/OS), plus a thin `main.go`. Go ≥ 1.26. Five direct dependencies, no cgo.

Before touching anything non-trivial, read **[ARCHITECTURE.md](ARCHITECTURE.md)** — it
explains the one constraint that shapes the whole codebase (you cannot run a second
`tea.Program` on a live terminal) and the invariants that keep bootstrap from locking you
out of your own server. [CLAUDE.md](CLAUDE.md) is a condensed version of the same map.

## Conventions

- **Commits**: conventional commits (`feat:`, `fix:`, `docs:`, `chore:`) — they drive the
  release changelog.
- **i18n**: every user-facing string lives in `i18n.go`. Add new keys to **both** `trRU` and
  `trEN` with matching format verbs (`%s` / `%d` / `%v`). `i18n_test.go` fails the build if
  a string is empty or the placeholders disagree.
- **Comments**: a short doc comment before every function, explaining *why* when the code
  already says *what*.
- **Screens**: user-facing output goes through `page.go` (`showPage` / `showToast` /
  `showError` / `runSpinner`), never `fmt.Println` — otherwise the full-screen UI collapses.
- **Remote output**: anything a server prints must pass through `sanitizeRemote()` before it
  reaches the terminal.
- **Colours**: only from the palette in `theme.go`, so the themes keep working.
- **Tables**: only via `newTable()`. **SSH options**: only via `sshCommonArgs()`.
  **Server passwords**: only via `serverPassword()`.

## Testing without a terminal or a server

- Key functions are declared as `var` rather than `func` so tests can swap them:
  `dialSSH`, `openRemote`, `runSelect`, `runInput`, `runMulti`, `runPage`, `runSpinner`,
  `scpRun`, `tunnelSpawn`, `fetchHealth`.
- `sshserver_test.go` spins up a real in-process SSH server, so the transport layer and
  `bootstrap` are exercised over an actual handshake.
- bubbletea models are driven directly: `m.Update(keyMsg("c"))`, then assert on `m.View()`.

## Pull requests

Keep them focused; describe what and why. CI runs format check, tests, the race detector,
`go vet`, the coverage gate, a build and a smoke test on both Linux and macOS, plus
`golangci-lint` and a `go mod tidy` check.

Found a security issue? Please read [SECURITY.md](SECURITY.md) instead of opening an issue.
