package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// newDebugCmd exists to inspect real Nitrado API responses while we
// finalize internal/api's field mappings against a live account. Not
// something end users need day-to-day, but it's cheap to keep around for
// diagnosing "why does field X look wrong" reports later too.
func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Short:  "Low-level tools for inspecting the raw Nitrado API",
		Hidden: true,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "api <method> <path>",
		Short: "Make an authenticated request and print the raw (pretty-printed) response",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			body, err := client.Raw(ctx, args[0], args[1])
			if len(body) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
			}
			return err
		},
	})
	return cmd
}
