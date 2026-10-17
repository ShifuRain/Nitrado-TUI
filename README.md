# nitui

[![ci](https://github.com/ShifuRain/Nitrado-TUI/actions/workflows/ci.yml/badge.svg)](https://github.com/ShifuRain/Nitrado-TUI/actions/workflows/ci.yml)
[![release](https://github.com/ShifuRain/Nitrado-TUI/actions/workflows/release.yml/badge.svg)](https://github.com/ShifuRain/Nitrado-TUI/actions/workflows/release.yml)

A cross-platform CLI + TUI for controlling [Nitrado](https://www.nitrado.net/) game servers. Every action works as a scriptable one-shot command, and running `nitui` with no arguments launches an interactive terminal UI, styled from a YAML config file.

> **Note:** This project's code was written by an AI coding assistant (Claude), working with a human directing the design, verifying behavior against a real Nitrado account, and reviewing changes.

## Status

Early. The original command surface (servers, select, switch, server status/stop/restart, games) works end-to-end against the real Nitrado API — base URL, auth header, error envelope, and the `Service`/`GameServer`/`Game` field mappings in `internal/api/types.go` have all been confirmed against a live account's actual responses. A couple of things are still unverified — see [Known gaps](#known-gaps).

## Installing a release

Prebuilt binaries are published on the [Releases page](https://github.com/ShifuRain/Nitrado-TUI/releases) for:

- **Linux**: amd64, arm64, armv6, armv7
- **Windows**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

Download the archive for your OS/architecture, extract it, and run the `nitui` binary (`nitui.exe` on Windows). Verify a download against `checksums.txt` in the same release if you want to confirm integrity.

**Package managers** (winget, AUR) are wired up in the release pipeline but not live yet, pending one-time maintainer setup. Once live: `winget install ShifuRain.nitui` on Windows, or `yay -S nitui-bin` (or any AUR helper) on Arch.

## Getting started (devcontainer)

This project is meant to be developed inside the provided devcontainer so you don't need Go installed locally.

1. Install Docker Desktop or Podman, and the VS Code "Dev Containers" extension.
2. Open this folder in VS Code and choose "Reopen in Container".
3. The container has no OS keychain available (no D-Bus Secret Service), so `NITUI_TOKEN_STORE=file` is set automatically inside it — this stores the token in a plaintext file under the config dir purely so `auth login`/`select`/etc. are testable there. The real binary you ship always defaults to the OS keychain.

Build and run:

```sh
go build -o nitui ./cmd/nitui
./nitui --help
```

Run tests:

```sh
go test ./...              # add -cover for a coverage summary
```

Tests cover the HTTP client and endpoint construction (`internal/api`, against `httptest` servers — no real network calls), the token stores (`internal/auth`), config/state persistence (`internal/config`, `internal/state`), CLI command wiring including error paths like the games-limit message (`internal/cli`), and the TUI's state machine (`internal/tui`). Rendering (`view.go`, `styles.go`) isn't covered — it's lower-value to test and changes often.

Two testing-only seams worth knowing about if you add commands: `NITUI_CONFIG_DIR` overrides where config/state live (tests point it at a temp dir instead of touching your real profile), and `api.WithBaseURL` / the package-level `apiBaseURL` var in `internal/cli` and `internal/tui` let tests point the API client at an `httptest.Server` instead of the real `api.nitrado.net`.

## Releasing

Releases are built by [GoReleaser](https://goreleaser.com/) (config: `.goreleaser.yaml`) via `.github/workflows/release.yml`, triggered by pushing a tag matching `v*` (e.g. `v1.0.0`):

```sh
git tag v1.0.1
git push origin v1.0.1
```

That builds every OS/arch combination listed above, archives them, generates checksums, and publishes a GitHub Release with an auto-generated changelog. `.github/workflows/ci.yml` runs `gofmt`/`go vet`/`staticcheck`/`go test` (the latter across Linux, Windows, and macOS runners) plus [`govulncheck`](https://go.dev/blog/govulncheck) — the official Go vulnerability scanner, which checks dependencies against the Go vulnerability database and only flags ones your code actually calls — on every push and pull request.

To sanity-check the release config locally without publishing anything:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

## Commands

```
nitui                            launch the interactive TUI
nitui auth login [--token]       save a Nitrado long-life API token
nitui auth logoff                remove the saved token (alias: logout)
nitui auth status                show whether you're logged in
nitui servers list               list every server on your account
nitui select <server_id>         choose the server follow-up commands act on
nitui switch <game_id>           switch the active game (starts it if installed, installs it otherwise)

nitui server status [--watch]    show the selected server's status (--watch follows changes live)
nitui server stop|restart        [--wait] [--message "..."] [--timeout 5m] [--yes]
nitui server update              restart so Nitrado applies a pending game update (see Known gaps)
nitui server stats [--hours N]   recent resource-usage statistics (raw JSON)

nitui settings list [category]   list game settings; * marks values changed from the default
nitui settings get <cat> <key>   print one setting
nitui settings set <cat> <key> <value>
nitui settings reset [--yes]     reset ALL settings to their defaults

nitui games list|installed       list games available / installed on the selected server
nitui games start <game_id>      make an installed game the active one
nitui games uninstall <game_id>  uninstall a game (frees a game slot)

nitui backups list|create        list / create backups
nitui backups restore|delete <name> [--yes]

nitui files ls [dir]             browse the server's files (same files as FTP)
nitui files cat <path>           print a remote file
nitui files get <remote> [local] download a file
nitui files put <local> <remote> upload a file (overwrites)
nitui files mkdir|rm <path>      create a directory / delete a file or directory

nitui command <console command>  send a command to the running game server
nitui logs                       show the admin log

nitui config path                print the config file location
nitui config init                write a commented example config file
```

`<game_id>` everywhere above means the short slug shown in `nitui games list` (e.g. `valheim`, `enshrouded`), not the numeric catalog id.

**Global flags:** `--server <id>` acts on a specific server for one command instead of the one saved by `nitui select`, and `--json` prints machine-readable output for `servers list`, `server status`, `games list|installed`, `settings list`, `backups list`, `files ls`, and `logs`.

**Confirmations:** destructive commands (`settings reset`, `backups restore|delete`, `files rm`, `games uninstall`) ask before acting and require `--yes` when there's no terminal. `server stop|restart` only prompt when run interactively, so existing scripts and cron jobs keep working.

### Live status

After you restart or stop a server (in the TUI, or with `--wait` on the command line), nitui polls Nitrado until the server has *actually* reached the new state — for a restart it first waits to see the server leave `started` so a stale "started" doesn't count as done. The TUI detail screen also refreshes itself every few seconds.

### The TUI

Open a server, then: `r` restart, `x` stop (both offer an optional message to players), `u` update, `s` games (`enter` switch/start, `u` uninstall), `e` settings (category → key → edit, with a restart offer after saving), `b` backups, `f` files (browse, preview, edit with `ctrl+s`, delete), `c` console (`ctrl+l` loads the admin log).

### Update notifications

nitui checks GitHub for a newer release at most once a day (cached next to your config, 3-second timeout, silent on any failure) and mentions it in the TUI and after interactive CLI commands. Turn it off with `update_check: false` in `config.yaml` or `NITUI_NO_UPDATE_CHECK=1`. Development builds never check.

### Authentication

Generate a token at nitrado.net: **My Account → Developer Portal → Long-life tokens**. `nitui auth login` prompts for it (masked input) and stores it in your OS's credential store — Windows Credential Manager, macOS Keychain, or the Linux Secret Service.

### Theming

`nitui config init` writes an example `config.yaml` to your platform's config directory (`nitui config path` prints the exact location). Every field is optional:

```yaml
theme:
  border_style: rounded  # rounded | normal | thick | double | hidden
  colors:
    primary: "#7D56F4"
    secondary: "#5A5A7A"
    accent: "#F25D94"
    text: "#E4E4E7"
    muted: "#71717A"
    success: "#4ADE80"
    warning: "#FBBF24"
    error: "#F87171"
    background: ""       # leave empty to use the terminal's own background

update_check: true       # set to false to disable the new-version notice
```

## Known gaps

These are called out in code comments where relevant too:

- **Games-per-server limit error text is unverified.** Nitrado's product pages document a 5-games-per-server cap, and the live `games` list confirms per-game `installed`/`enough_slots`/`too_many_slots` flags, but the exact error message/status code returned when you actually hit the 5-game cap hasn't been observed yet. `Error.FriendlyMessage()` currently uses a substring heuristic (any error message containing "limit") to append guidance — this should be tightened once we've triggered the real error.
- **`switch <game_id>` semantics are partially unverified.** An installed game is started with `games/start`, anything else is installed with `games/install` (the slug is the confirmed identifier — see `internal/api/types.go`). Neither call has been confirmed live for the installed-but-inactive case.
- **No generic "start the whole server" endpoint was found** in Nitrado's SDKs, only `stop` and `restart` — so there's no `nitui server start` command. `restart` starts a stopped server too.
- **Game-file updates have no dedicated API.** Nitrado updates games automatically, and the SDK has no "update now" call. `server update` (TUI: `u`) just restarts the server, on the assumption that it applies a pending update on start; that's unconfirmed. `server status` shows the `update_status` Nitrado reports, when there is one.
- **Several response shapes were written from the official Nitrapi-PHP SDK and are not yet confirmed against a live account.** Parsing is deliberately tolerant (a surprise degrades to "nothing shown", never a crash), but check these with `nitui debug api` before relying on them: settings values (always sent as strings; booleans/numbers unverified), `backups` list/restore, `admin_logs`, `stats`, and the file-server calls (`list`, `download`/`upload` token handoff, `delete`).
