<div align="right"><b>English</b> · <a href="README.ru.md">Русский</a></div>

<h1 align="center">unissh</h1>

<p align="center">A beautiful TUI for managing your VPS fleet over SSH — open it, pick a server, you're in.</p>

<p align="center">
  <a href="https://github.com/MrBenoit/unissh/actions/workflows/ci.yml"><img src="https://github.com/MrBenoit/unissh/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/MrBenoit/unissh/releases/latest"><img src="https://img.shields.io/github/v/release/MrBenoit/unissh" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/MrBenoit/unissh"><img src="https://goreportcard.com/badge/github.com/MrBenoit/unissh" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="License"></a>
</p>

```
 ██╗   ██╗███╗   ██╗██╗███████╗███████╗██╗  ██╗
 ██║   ██║████╗  ██║██║██╔════╝██╔════╝██║  ██║
 ██║   ██║██╔██╗ ██║██║███████╗███████╗███████║
 ██║   ██║██║╚██╗██║██║╚════██║╚════██║██╔══██║
 ╚██████╔╝██║ ╚████║██║███████║███████║██║  ██║
  ╚═════╝ ╚═╝  ╚═══╝╚═╝╚══════╝╚══════╝╚═╝  ╚═╝
```

<p align="center"><img src="demo.gif" alt="unissh demo" width="800"></p>

> **Note:** replace `MrBenoit` with your GitHub handle in `go.mod`, the
> badges above, `.goreleaser.yaml` and the workflows before publishing.

## Features

- **Interactive main screen** (lazygit-style): servers on top, actions below.
  Navigate with **arrows and the mouse** — click a row to select+act, wheel to
  scroll. Live status and ping, auto-refreshed.
- **One-key actions** on the selected server: `c`/Enter connect, `i` info,
  `b` bootstrap, `e` edit, `r` reboot, `d` delete, `m` manage, `f` favorite,
  `s` sort, `/` fuzzy search, `?` keybinding help.
- **Auto password login**: for password servers the password is typed
  automatically (via `expect`) — an instant ready session, no clipboard dance.
  The password never touches disk or `argv` (passed through the environment).
- **Automatic first-time setup (bootstrap)** of a fresh VPS: system update,
  sudo user with a generated password, SSH key, UFW/firewalld, fail2ban, sshd
  hardening (key-only login), custom port, swap, timezone, hostname — with
  **lockout protection** (key login is verified before passwords are disabled;
  on failure the sshd config is rolled back).
- **Passwords in the macOS Keychain** — nothing stored in plain text.
- **Full info panel** on add and inspection: OS, uptime, **all interface IPs**,
  live CPU%, load, memory/disk bars, docker containers.
- **ssh-agent & passphrase keys**, **ProxyJump / bastion**, **snippets** for
  frequent commands, **import from `~/.ssh/config`**, **connection history**,
  **config backups with restore**, **color themes** (dark/light/dracula/gruvbox).
- **Open SSH in a new terminal window** (Warp / iTerm2 / Terminal.app,
  auto-detected) or inline — your choice.
- **Export to `~/.ssh/config`** so `ssh <name>` works from any terminal.
- **Two languages** (English / Russian).

## Installation

```bash
# Homebrew (after you set up a tap)
brew install MrBenoit/tap/unissh

# Go
go install github.com/MrBenoit/unissh@latest

# From source
git clone https://github.com/MrBenoit/unissh
cd unissh && ./install.sh
```

Requires Go ≥ 1.26 to build from source. macOS is a first-class target; Linux
support is in progress.

## Usage

```
unissh              interactive TUI
unissh <name>       quick connect (prefix match)
unissh list         list servers with statuses
unissh add          add a server
unissh version      show version
```

Press `?` inside the app for the full keybinding list.

## Configuration

| What | Where |
|---|---|
| Servers & settings | `~/.config/unissh/servers.json` |
| Host fingerprints | `~/.config/unissh/known_hosts` |
| Backups | `~/.config/unissh/backups/` |
| Passwords | macOS Keychain (service `unissh`) |
| SSH key | `~/.ssh/unissh_ed25519` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Architecture map: [CLAUDE.md](CLAUDE.md).

## License

[MIT](LICENSE)
