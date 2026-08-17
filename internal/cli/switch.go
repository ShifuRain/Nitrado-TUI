package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <game_id>",
		Short: "Install/switch the active game on the selected server",
		Long: "Install/switch the active game on the server chosen with `nitui select`.\n" +
			"<game_id> is the short game slug (e.g. \"valheim\", \"enshrouded\") — run `nitui games list` to see valid ids for this server.\n" +
			"If the server has reached its installed-games limit, Nitrado's error is surfaced with guidance on freeing up space.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := selectedServerID()
			if err != nil {
				return err
			}

			client, err := requireClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := client.SwitchGame(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Switching server %d to game %q.\n", id, args[0])
			return nil
		},
	}
	return cmd
}
