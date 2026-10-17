// Package cli wires up nitui's command-line interface with Cobra. Running
// the binary with no subcommand launches the interactive TUI; every
// subcommand also works standalone for scripting.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/config"
	"nitui/internal/tui"
	"nitui/internal/update"
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

	// Ctrl-C cancels in-flight work (--wait, --watch) cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	notice := startUpdateCheck(ctx, info.Version, os.Args[1:])
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		stop()
		os.Exit(1)
	}
	printUpdateNotice(notice)
}

// startUpdateCheck begins the (cached, time-boxed) new-version lookup in the
// background and returns a channel that yields at most one release. Nothing
// here may ever fail or delay a command: errors are dropped, and the caller
// only waits a moment for the result after the command has finished.
func startUpdateCheck(ctx context.Context, version string, args []string) <-chan *update.Release {
	ch := make(chan *update.Release, 1)
	if !wantsUpdateNotice(args) {
		close(ch)
		return ch
	}
	cfg, err := config.Load()
	if err != nil || update.Disabled(cfg) {
		close(ch)
		return ch
	}
	go func() {
		defer close(ch)
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if r, err := update.Check(cctx, version, update.Options{}); err == nil && r != nil {
			ch <- r
		}
	}()
	return ch
}

// wantsUpdateNotice is false for non-interactive uses where an extra line
// would be noise or break parsing: JSON output, piped stderr, completion,
// version/help, and the no-argument TUI (which shows its own banner).
func wantsUpdateNotice(args []string) bool {
	if len(args) == 0 || !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}
	for _, a := range args {
		switch a {
		case "--json", "--version", "--help", "-h", "completion", "help", "__complete", "--watch":
			return false
		}
	}
	return true
}

func printUpdateNotice(ch <-chan *update.Release) {
	select {
	case r, ok := <-ch:
		if ok && r != nil {
			fmt.Fprintln(os.Stderr, "\n"+update.Notice(r))
		}
	case <-time.After(300 * time.Millisecond):
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
			return tui.Run(cfg, authStore, info.Version)
		},
	}

	addGlobalFlags(cmd)
	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newSettingsCmd())
	cmd.AddCommand(newBackupsCmd())
	cmd.AddCommand(newFilesCmd())
	cmd.AddCommand(newCommandCmd())
	cmd.AddCommand(newLogsCmd())
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
