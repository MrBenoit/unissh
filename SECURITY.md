# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Use GitHub's [private vulnerability reporting](https://github.com/MrBenoit/unissh/security/advisories/new)
instead. You should get a first response within a few days.

If you prefer email, write to the address on the maintainer's GitHub profile with
`unissh security` in the subject.

## Supported versions

Only the latest release receives fixes.

## Threat model

unissh holds SSH credentials for your whole fleet, so it is worth being explicit
about what it does and does not protect against.

### What unissh does

| Area | Measure |
|---|---|
| Passwords at rest | System keyring: macOS Keychain, or the Secret Service on Linux via `secret-tool` (service `unissh`, account `<serverID>/<username>`). If no keyring daemon is reachable: `~/.config/unissh/secrets.json`, mode `0600` |
| Passwords in flight | Passed to `expect` through the environment (`UNISSH_SSHPASS`), never through `argv`, never written to disk. For a new terminal window the password is embedded in a `0600` script that deletes itself on first run |
| Clipboard | A copied password is auto-cleared after 45 s. Only its **SHA-256 hash** appears in the cleanup command's `argv`; the clipboard is only wiped if its content still matches. Uses `pbcopy`, `wl-copy`, `xclip` or `xsel` — whichever is present |
| Host keys | Trust on first use. An unknown host is recorded; a **changed** fingerprint is a hard error. One `known_hosts` shared by the Go client and the system `ssh` |
| Remote output | Everything printed to your terminal passes through `sanitizeRemote()`, which strips ANSI and control sequences — a compromised server cannot move your cursor or spoof prompts |
| Remote command injection | User input reaching a remote shell is either validated by regex, quoted (`shellQuote`), or passed via **stdin** instead of `argv` |
| SSH config export | Values written to `~/.ssh/unissh_config` are escaped and newline-stripped, so a server name cannot inject a `ProxyCommand` directive |
| Lockout | Bootstrap verifies key-based login **before** disabling password auth, and rolls the `sshd` config back over the still-open root session if the post-restart login check fails |
| Supply chain | Five direct dependencies. `go mod tidy` is verified in CI. `gosec` runs on every push |

### Known limitations

These are deliberate trade-offs, not oversights.

1. **On macOS, Keychain writes pass the password in `argv`.** The `security` CLI has no
   non-interactive way to accept a secret on stdin, so `security add-generic-password -w <pw>`
   briefly exposes the password to `ps` for the **local user**. Acceptable on a
   single-user Mac; fixing it properly requires cgo and the Security framework.
   Linux does not have this problem: `secret-tool store` reads the secret from stdin.

   If no keyring is reachable (a headless box, a locked wallet), passwords fall back to
   `~/.config/unissh/secrets.json` — **plaintext, mode `0600`**. That is a real downgrade.
   Install `libsecret-tools` and run a keyring daemon if it matters to you.

2. **The journal may contain secrets.** `~/.config/unissh/history.jsonl` records the
   commands you ran. If you type `mysql -pPASSWORD`, that string is stored. The file is
   `0600`, the UI says so on screen, and it can be wiped from the journal viewer.
   It is never copied into config backups.

3. **`StrictHostKeyChecking=accept-new`** is passed to the system `ssh`. This is TOFU,
   not verification: the very first connection to a host is trusted. If you can obtain
   fingerprints out of band, add them to `~/.config/unissh/known_hosts` beforehand.

4. **Encrypted keys need an agent.** A passphrase-protected key works through `ssh-agent`
   or a passphrase stored in the Keychain. There is no interactive passphrase prompt in
   the programmatic client.

5. **Tunnels are child processes.** They run in their own process group and are killed
   when unissh exits. They are not daemonised and do not survive a crash of the app.

### What unissh does not defend against

- A compromised local account. Anything that can read your Keychain, your `~/.ssh`, or
  attach to your processes has already won.
- A malicious server you deliberately connect to, beyond sanitising its output.
- Physical access to an unlocked machine.
