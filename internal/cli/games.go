package cli

import (
	"context"
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
			id, err := selectedServerID()
			if err != nil {
				return err
			}
			client, err := requireClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
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
