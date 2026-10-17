package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"nitui/internal/api"
	"nitui/internal/state"
)

// addGlobalFlags registers the flags every command understands.
func addGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Int("server", 0, "act on this server id instead of the one saved by 'nitui select'")
	cmd.PersistentFlags().Bool("json", false, "print machine-readable JSON instead of text (where supported)")
}

// resolveServer returns the server id to act on: --server if given,
// otherwise the one saved by `nitui select`.
func resolveServer(cmd *cobra.Command) (int, error) {
	if f := cmd.Flags().Lookup("server"); f != nil && f.Changed {
		id, err := cmd.Flags().GetInt("server")
		if err != nil || id <= 0 {
			return 0, errors.New("--server must be a positive server id")
		}
		return id, nil
	}
	return selectedServerID()
}

func selectedServerID() (int, error) {
	idStr, err := state.SelectedServer()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("stored selected server id %q is invalid, re-run `nitui select`: %w", idStr, err)
	}
	return id, nil
}

// prepare resolves the target server and builds an authenticated client.
func prepare(cmd *cobra.Command) (*api.Client, int, error) {
	id, err := resolveServer(cmd)
	if err != nil {
		return nil, 0, err
	}
	client, err := requireClient()
	if err != nil {
		return nil, 0, err
	}
	return client, id, nil
}

func timeoutCtx(cmd *cobra.Command, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(cmd.Context(), d)
}

// wantJSON reports whether --json was passed.
func wantJSON(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// printJSON writes v as indented JSON to the command's stdout.
func printJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func addYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip the confirmation prompt")
}

// stdinIsTerminal is a variable so tests can simulate an interactive session.
var stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// confirm asks for a y/N answer before a destructive action. With --yes it
// proceeds silently; without a terminal it refuses rather than guessing.
func confirm(cmd *cobra.Command, prompt string) error {
	return ask(cmd, prompt, true)
}

// confirmInteractive prompts only when a human is at a terminal. Scripts and
// cron jobs (no terminal) proceed, so existing automation like
// `nitui server restart` keeps working.
func confirmInteractive(cmd *cobra.Command, prompt string) error {
	return ask(cmd, prompt, false)
}

func ask(cmd *cobra.Command, prompt string, requireYes bool) error {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return nil
	}
	if !stdinIsTerminal() {
		if requireYes {
			return fmt.Errorf("%s — re-run with --yes to confirm (no terminal to ask on)", prompt)
		}
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N] ", prompt)
	line, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	}
	return errors.New("cancelled")
}
