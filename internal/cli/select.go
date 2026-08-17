package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"nitui/internal/state"
)

func newSelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select <server_id>",
		Short: "Choose the server that follow-up commands (switch, server ...) act on",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("server_id must be a number, got %q", args[0])
			}

			client, err := requireClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			services, err := client.ListServices(ctx)
			if err != nil {
				return apiErr(err)
			}

			found := false
			for _, s := range services {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("server %d isn't on your account, run `nitui servers list` to see valid ids", id)
			}

			if err := state.SetSelectedServer(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Selected server %d.\n", id)
			return nil
		},
	}
	return cmd
}
