package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"nitui/internal/auth"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication with the Nitrado API",
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoffCmd())
	cmd.AddCommand(newAuthStatusCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var token string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save your Nitrado long-life API token",
		Long: "Save your Nitrado long-life API token.\n\n" +
			"Generate one at nitrado.net: My Account -> Developer Portal -> Long-life tokens.\n" +
			"The token is stored in your OS's credential store (Windows Credential Manager, " +
			"macOS Keychain, or the Linux Secret Service) — never in a plaintext file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := strings.TrimSpace(token)
			if t == "" {
				var err error
				t, err = promptForToken(cmd)
				if err != nil {
					return err
				}
			}
			if t == "" {
				return errors.New("no token provided")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			if _, err := newAPIClient(t).ListServices(ctx); err != nil {
				return fmt.Errorf("that token didn't work: %w", apiErr(err))
			}

			if err := authStore.Save(t); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged in. Token saved to your OS keychain.")
			return nil
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "long-life API token (omit to be prompted, recommended so it doesn't end up in shell history)")
	return cmd
}

func promptForToken(cmd *cobra.Command) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Nitrado long-life token: ")
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("no --token flag given and stdin isn't a terminal to prompt on")
	}
	data, err := term.ReadPassword(fd)
	fmt.Fprintln(cmd.OutOrStdout())
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func newAuthLogoffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logoff",
		Aliases: []string{"logout"},
		Short:   "Remove the saved API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := authStore.Delete(); err != nil {
				if errors.Is(err, auth.ErrNotLoggedIn) {
					fmt.Fprintln(cmd.OutOrStdout(), "Already logged out.")
					return nil
				}
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged off. Token removed from your OS keychain.")
			return nil
		},
	}
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether you're currently logged in",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := authStore.Get()
			if err != nil {
				if errors.Is(err, auth.ErrNotLoggedIn) {
					fmt.Fprintln(cmd.OutOrStdout(), "Not logged in. Run `nitui auth login`.")
					return nil
				}
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			services, err := newAPIClient(token).ListServices(ctx)
			if err != nil {
				return fmt.Errorf("logged in, but the saved token no longer works: %w", apiErr(err))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in. %d server(s) on this account.\n", len(services))
			return nil
		},
	}
	return cmd
}
