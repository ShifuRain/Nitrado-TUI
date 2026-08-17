// Package cli wires up nitui's command-line interface with Cobra. Running
// the binary with no subcommand launches the interactive TUI; every
// subcommand also works standalone for scripting.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/config"
	"nitui/internal/tui"
)

// authStore defaults to the OS keychain. Setting NITUI_TOKEN_STORE=file
// switches to a plaintext file under the config dir instead — this exists
// solely for environments with no OS keychain (this project's own
// devcontainer has no D-Bus Secret Service) and must never be relied on
// for a real install.
var authStore auth.Store = newAuthStore()

func newAuthStore() auth.Store {
	if os.Getenv("NITUI_TOKEN_STORE") == "file" {
		dir, err := config.Dir()
		if err == nil {
			return auth.NewFileStore(dir)
		}
	}
	return auth.NewKeychainStore()
}

// BuildInfo carries version metadata injected at build time via -ldflags
// (see .goreleaser.yaml); main.go's defaults ("dev"/"none"/"unknown") apply
// to plain `go build` runs.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func Execute(info BuildInfo) {
	root := newRootCmd(info)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newRootCmd(info BuildInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "nitui",
		Short:         "Control your Nitrado game servers from the terminal",
		Long:          "nitui is a CLI and TUI for managing Nitrado game servers: authenticate, list servers, switch installed games, and control server lifecycle — interactively or scripted.",
		Version:       fmt.Sprintf("%s (commit %s, built %s)", info.Version, info.Commit, info.Date),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			return tui.Run(cfg, authStore)
		},
	}

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newSelectCmd())
	cmd.AddCommand(newSwitchCmd())
	cmd.AddCommand(newServersCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newGamesCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newDebugCmd())

	return cmd
}

// apiBaseURL overrides the Nitrado API base URL when set. Tests point this
// at an httptest.Server; production code leaves it empty to use the real API.
var apiBaseURL string

func newAPIClient(token string) *api.Client {
	if apiBaseURL != "" {
		return api.New(token, api.WithBaseURL(apiBaseURL))
	}
	return api.New(token)
}

// requireClient loads the saved token and builds an authenticated API
// client, or returns a friendly error telling the user to log in.
func requireClient() (*api.Client, error) {
	token, err := authStore.Get()
	if err != nil {
		return nil, err
	}
	return newAPIClient(token), nil
}

// apiErr renders an error for CLI output, using api.Error's friendly
// message when available.
func apiErr(err error) error {
	var e *api.Error
	if errors.As(err, &e) {
		return fmt.Errorf("%s", e.FriendlyMessage())
	}
	return err
}
