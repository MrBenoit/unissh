<div align="right"><b>English</b> · <a href="README.ru.md">Русский</a></div>

<h1 align="center">unissh</h1>

<p align="center">
  <b>Your whole VPS fleet, one keystroke away.</b><br>
  A fast, colourful terminal UI for SSH — open it, pick a server, you're in.
</p>

<p align="center">
  <a href="https://github.com/MrBenoit/unissh/actions/workflows/ci.yml"><img src="https://github.com/MrBenoit/unissh/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/MrBenoit/unissh/releases/latest"><img src="https://img.shields.io/github/v/release/MrBenoit/unissh" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/MrBenoit/unissh"><img src="https://goreportcard.com/badge/github.com/MrBenoit/unissh" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License"></a>
</p>

```
 ██╗   ██╗███╗   ██╗██╗███████╗███████╗██╗  ██╗
 ██║   ██║████╗  ██║██║██╔════╝██╔════╝██║  ██║
 ██║   ██║██╔██╗ ██║██║███████╗███████╗███████║
 ██║   ██║██║╚██╗██║██║╚════██║╚════██║██╔══██║
 ╚██████╔╝██║ ╚████║██║███████║███████║██║  ██║
  ╚═════╝ ╚═╝  ╚═══╝╚═╝╚══════╝╚══════╝╚═╝  ╚═╝
```

<p align="center">
  <img src="demo.gif" alt="unissh in action: server list with folders, live stats, search, batch actions" width="900">
</p>

---

## Why unissh?

You already have `~/.ssh/config`. You already have a text file with IP addresses somewhere.
This replaces both with something you can actually look at.

- **It shows you a fleet, not a list of strings.** Live status, ping, CPU, memory, disk and
  running tunnels — right next to the server your cursor is on.
- **It sets up new boxes for you.** Point it at a fresh VPS with a root password: it creates
  a sudo user, installs your key, hardens `sshd`, configures the firewall and fail2ban —
  *and verifies that key login works before it turns passwords off.*
- **It doesn't leak your passwords.** No `sshpass`. Nothing on disk. Passwords live in your
  system keyring — the macOS Keychain or the Secret Service on Linux. The clipboard
  auto-clears after 45 seconds, and only a SHA-256 hash of the secret ever reaches `argv`.
- **It stays out of your way.** Everything is one keystroke. The mouse works too, if you want it.

---

## The main screen

Four panels, lazygit style. Arrows or mouse; `Tab` moves between panels.

```
┌─ SERVERS ────────────┬─ ACTIONS ────────────┐
│ ▾ prod               │ ❯ [•] Connect        │
│     ● web-1        ★ │   [ ] Server info    │
│     ● db-1           │   [ ] Run a command  │
│ ▸ staging          3 │   [ ] Port forwarding│
│   ✗ old-box          ├─ STATS ──────────────┤
│                      │  Ping  ● 12 ms       │
├─ MENU ───────────────┤   CPU  ████░░░░  42% │
│ ❯ [•] Add a server   │   RAM  ███░░░░░  31% │
│   [ ] Fleet health   │  Disk  ██░░░░░░  24% │
│   [ ] Journal        │   Net  ↓ 1.2M ↑ 88K  │
│   [ ] Settings       │ Tunnels ● 1 up       │
└──────────────────────┴──────────────────────┘
 tab — panels · space — select · c — connect · ? — help
```

---

## Features

### Daily driver

- **Connect in one key.** `c` on any server. Opens in a new terminal window — Warp, iTerm2,
  Terminal.app, kitty, gnome-terminal and friends are detected automatically — or inline.
- **Auto password login.** Password-auth servers get their password typed in for you via
  `expect`. An instant, ready session — no clipboard dance.
- **Folders.** Group servers, collapse with `←`/`→`. Press `m` to **grab** a server — it
  travels between folders with your cursor until `Enter` drops it.
- **Multi-select and batch actions.** `space` marks a server (on a folder row — the whole
  group). Run one command everywhere, move them, delete them.
- **Search, sort, favourites.** `/` filters, `s` cycles the sort mode, `f` pins to the top.

### Fleet management

- **Live monitor** built into the main screen: CPU, RAM, disk, load, network throughput,
  uptime and every interface IP — refreshed over a *single persistent* SSH connection,
  so fail2ban never sees a reconnect storm.
- **Fleet health dashboard.** One table, every server, worst-first. Answers "who is on fire
  right now" in a single keystroke.
- **Port forwarding.** `-L`, `-R` and `-D` (SOCKS5) tunnels running in the background, with
  their state shown right in the stats panel — up, stopped, or failed *with the reason*.
- **File transfer** over `scp`, with progress, in both directions.
- **Journal.** Every login, every command with its exit code, every transfer.

### Provisioning

- **Automatic first-time setup** of a fresh VPS: system update, base packages, a sudo user
  with a generated password, your SSH key, UFW/firewalld, fail2ban, sshd hardening,
  a custom SSH port, swap, timezone, hostname, unattended security upgrades.
- **Lockout protection.** Key-based login is verified *before* passwords are disabled. If the
  check after the `sshd` restart fails, the config is rolled back over the still-open root
  session. The full list of invariants is in
  [ARCHITECTURE.md](ARCHITECTURE.md#12-bootstrap-инварианты-против-потери-доступа).

### Comfort

**ssh-agent** and passphrase keys · **ProxyJump / bastion** · **snippets** for frequent
commands · **import from `~/.ssh/config`** · **export back into it**, so `ssh <name>` works
from any terminal · **config backups with restore** · **five colour themes** ·
**two languages** (English / Russian).

---

## Install

```bash
# Go (recommended)
go install github.com/MrBenoit/unissh@latest

# From source
git clone https://github.com/MrBenoit/unissh
cd unissh && ./install.sh
```

> Homebrew (`brew install MrBenoit/tap/unissh`) is planned — uncomment the `brews`
> block in `.goreleaser.yaml` and set the tap token to enable it.

**Requirements.** Go ≥ 1.26 to build from source. **macOS and Linux are both supported.**
The **servers** you manage must be Linux — the monitor reads `/proc`, and bootstrap speaks
`apt` and `dnf`.

unissh degrades gracefully when an optional tool is missing, but it is nicer with them:

| Feature | macOS | Linux |
|---|---|---|
| Password storage | Keychain (built in) | `secret-tool` — `libsecret-tools`. Falls back to a `0600` file |
| Clipboard | `pbcopy` (built in) | `wl-clipboard` on Wayland, `xclip` or `xsel` on X11 |
| Auto password login | `expect` (built in) | `expect` — `apt install expect` |
| Open in a new window | Warp / iTerm2 / Terminal.app | kitty, foot, ghostty, wezterm, alacritty, gnome-terminal, konsole, xfce4-terminal, xterm, or `$TERMINAL` |

```bash
# Ubuntu / Debian, everything at once:
sudo apt install expect wl-clipboard libsecret-tools
```

If no terminal emulator is found, unissh quietly connects in the current window instead of
failing. `./install.sh` tells you what's missing.

---

## Usage

```
unissh              interactive TUI
unissh <name>       quick connect (prefix match, e.g. `unissh prod`)
unissh list         list servers with statuses (works without a TTY)
unissh add          add a server
unissh version      show version
```

### Keybindings

Press `?` inside the app to see this list at any time.

| Key | Action |
|---|---|
| `tab` / `shift+tab` | move between panels |
| `↑` `↓` / `k` `j` / mouse | navigate; `g` / `G` jump to top / bottom |
| `enter` / `→` | open actions for the server (or expand a folder) |
| `space` | mark a server — on a folder row, the whole group |
| `c` | connect |
| `t` | port forwarding |
| `f` | toggle favourite |
| `s` | cycle sort mode |
| `/` | search |
| `n` `r` `d` | new / rename / delete folder |
| `m` | grab the server and carry it between folders |
| `shift`+`↑↓` | reorder folders |
| `?` `esc` `q` | help · back · quit |

### Adding a server with bootstrap

1. `unissh` → **Add a server**
2. Name, IP, port → **With initial setup**
3. Give it the root password (the one your hosting provider emailed you)
4. Tick the steps you want — everything safe is on by default
5. Go get a coffee. You come back to a card with the new login and a generated password,
   already in your Keychain and in your clipboard.

> ⚠️ After hardening, password login is **disabled**. Don't delete `~/.ssh/unissh_ed25519`.
> The key is installed for `root` as well, as a spare way in.

---

## Where things live

| What | Where |
|---|---|
| Servers & settings | `~/.config/unissh/servers.json` |
| Host fingerprints | `~/.config/unissh/known_hosts` |
| Config backups | `~/.config/unissh/backups/` |
| Journal (`0600` — commands may contain secrets) | `~/.config/unissh/history.jsonl` |
| Passwords | macOS Keychain, or the Linux Secret Service (`secret-tool`), service `unissh` |
| SSH key | `~/.ssh/unissh_ed25519` |
| Exported ssh config | `~/.ssh/unissh_config` (+ an `Include` line in `~/.ssh/config`) |

`./uninstall.sh` removes the binary and tells you exactly what it left behind.

---

## Under the hood

Two packages. `internal/core` is pure logic — no TUI, no network, no i18n. It is tested to
**99%**, and `make build` refuses to run below 90%. `internal/app` is everything that
touches a terminal, a socket or the OS.

`make build` gates on `fmt` → `vet` → tests → coverage. `make ci` adds the race detector.
You cannot ship a red build.

- **[ARCHITECTURE.md](ARCHITECTURE.md)** — the long read (in Russian): every design decision,
  every trade-off, and a chronicle of the bugs we hit and how they were fixed. Start here if
  you want to understand *why* the code looks the way it does.
- **[SECURITY.md](SECURITY.md)** — threat model and known limitations.
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — how to build, test and send a patch.
- **[CLAUDE.md](CLAUDE.md)** — a condensed map for AI assistants.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea),
[lipgloss](https://github.com/charmbracelet/lipgloss) and
[x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh).
Five direct dependencies, no cgo.

---

## License

[MIT](LICENSE)
