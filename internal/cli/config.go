package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"nitui/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage the nitui config file (theming, etc.)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the config file's location",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Write a commented example config file you can edit",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.WriteDefault()
			if errors.Is(err, os.ErrExist) {
				fmt.Fprintf(cmd.OutOrStdout(), "Config already exists at %s, leaving it untouched.\n", path)
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote example config to %s\n", path)
			return nil
		},
	})
	return cmd
}
