package cli

import (
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
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()

			// A game that's already installed is started; anything else is
			// installed. If the lookup fails we fall back to install, which
			// is what `switch` always did.
			if games, err := client.ListGames(ctx, id); err == nil {
				for _, g := range games {
					if g.Slug == args[0] && g.Installed {
						if g.Active {
							fmt.Fprintf(cmd.OutOrStdout(), "Game %q is already the active game on server %d.\n", args[0], id)
							return nil
						}
						if err := client.StartGame(ctx, id, args[0]); err != nil {
							return apiErr(err)
						}
						fmt.Fprintf(cmd.OutOrStdout(), "Switching server %d to game %q.\n", id, args[0])
						return nil
					}
				}
			}
			if err := client.SwitchGame(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Switching server %d to game %q.\n", id, args[0])
			return nil
		},
	}
	return cmd
}
