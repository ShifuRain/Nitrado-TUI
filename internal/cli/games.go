package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"nitui/internal/api"
)

func newGamesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "games",
		Short: "List games available on, or installed on, the selected server",
	}
	cmd.AddCommand(newGamesListCmd(false))
	cmd.AddCommand(newGamesListCmd(true))
	cmd.AddCommand(newGamesStartCmd())
	cmd.AddCommand(newGamesUninstallCmd())
	return cmd
}

func newGamesListCmd(installedOnly bool) *cobra.Command {
	use, short := "list", "List every game available to install/switch to on the selected server"
	if installedOnly {
		use, short = "installed", "List games currently installed on the selected server"
	}

	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 15*time.Second)
			defer cancel()
			games, err := client.ListGames(ctx, id)
			if err != nil {
				return apiErr(err)
			}

			if installedOnly {
				filtered := games[:0]
				for _, g := range games {
					if g.Installed {
						filtered = append(filtered, g)
					}
				}
				games = filtered
			}
			if wantJSON(cmd) {
				return printJSON(cmd, games)
			}
			if len(games) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No games found.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "GAME ID\tNAME\tINSTALLED\tACTIVE\tFITS SLOTS")
			for _, g := range games {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					g.Slug, g.Name, yesNo(g.Installed), yesNo(g.Active), fitsSlots(g))
			}
			return w.Flush()
		},
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func fitsSlots(g api.Game) string {
	switch {
	case !g.EnoughSlots:
		return "no — not enough slots"
	case g.TooManySlots:
		return "no — too many slots"
	default:
		return "yes"
	}
}

func newGamesStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <game_id>",
		Short: "Make an already-installed game the active one",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.StartGame(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Starting game %q on server %d.\n", args[0], id)
			return nil
		},
	}
}

func newGamesUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <game_id>",
		Short: "Uninstall a game from the server (frees a game slot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if err := confirm(cmd, fmt.Sprintf("Uninstall %q from server %d? Its files on the server may be lost.", args[0], id)); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.UninstallGame(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled game %q from server %d.\n", args[0], id)
			return nil
		},
	}
	addYesFlag(cmd)
	return cmd
}
