# nitui

A cross-platform CLI + TUI for controlling [Nitrado](https://www.nitrado.net/) game servers. Every action works as a scriptable one-shot command, and running `nitui` with no arguments launches an interactive terminal UI, styled from a YAML config file.

> **Note:** This project's code was written by an AI coding assistant (Claude), working with a human directing the design, verifying behavior against a real Nitrado account, and reviewing changes.

## Status

Early scaffolding. The command surface below works end-to-end against the real Nitrado API — base URL, auth header, error envelope, and the `Service`/`GameServer`/`Game` field mappings in `internal/api/types.go` have all been confirmed against a live account's actual responses. A couple of things are still unverified — see [Known gaps](#known-gaps).

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

## Commands

```
nitui                            launch the interactive TUI
nitui auth login [--token]       save a Nitrado long-life API token
nitui auth logoff                remove the saved token (alias: logout)
nitui auth status                show whether you're logged in
nitui servers list               list every server on your account
nitui select <server_id>         choose the server follow-up commands act on
nitui switch <game_id>           install/switch the active game on the selected server
nitui server status              show the selected server's status
nitui server stop                stop the selected server
nitui server restart             restart the selected server (also starts it if stopped)
nitui games list                 list every game available on the selected server
nitui games installed            list games currently installed on the selected server
nitui config path                print the config file location
nitui config init                write a commented example config file
```

`<game_id>` everywhere above means the short slug shown in `nitui games list` (e.g. `valheim`, `enshrouded`), not the numeric catalog id.

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
```

## Known gaps

These are called out in code comments where relevant too:

- **Games-per-server limit error text is unverified.** Nitrado's product pages document a 5-games-per-server cap, and the live `games` list confirms per-game `installed`/`enough_slots`/`too_many_slots` flags, but the exact error message/status code returned when you actually hit the 5-game cap hasn't been observed yet. `Error.FriendlyMessage()` currently uses a substring heuristic (any error message containing "limit") to append guidance — this should be tightened once we've triggered the real error.
- **`switch <game_id>` semantics are partially unverified.** It calls the `games/install` endpoint with the game's slug (confirmed correct identifier — see `internal/api/types.go`). What's not yet confirmed live is whether `install` is also correct for switching to a game that's already installed-but-inactive, or whether that case needs `games/start` instead.
- **No generic "start the whole server" endpoint was found** in Nitrado's SDKs, only `stop` and `restart` — so there's no `nitui server start` command. `restart` starts a stopped server too.
