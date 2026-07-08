# Contributing to unissh

Thanks for your interest! unissh is a small, friendly Go/TUI project.

## Development

```bash
git clone https://github.com/MrBenoit/unissh
cd unissh
go build -o unissh .   # build
go vet ./...           # required before a PR
gofmt -l .             # must print nothing
go test ./...          # run tests
./unissh               # try it against a test VPS
```

One Go module, one package `main`, Go ≥ 1.26. UI is built on
`charmbracelet/bubbletea` + `lipgloss` (see [CLAUDE.md](CLAUDE.md) for the
architecture map).

## Conventions

- **Commits**: conventional commits (`feat:`, `fix:`, `docs:`, `chore:`) — they
  drive the changelog.
- **i18n**: every user-facing string lives in `i18n.go`. Add new keys to **both**
  `trRU` and `trEN` with matching format verbs (`%s`/`%d`/`%v`).
- **Comments**: a short doc comment before each function.
- **Colors**: only from the palette in `theme.go` (so themes keep working).

## Pull requests

Keep them focused; describe what and why. CI runs build/vet/test/lint on
Linux and macOS.
