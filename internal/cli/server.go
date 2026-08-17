package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"nitui/internal/api"
	"nitui/internal/state"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Control the lifecycle of the selected server",
	}
	cmd.AddCommand(newServerStatusCmd())
	cmd.AddCommand(newServerActionCmd("stop", "Stop the selected server", (*api.Client).Stop))
	cmd.AddCommand(newServerActionCmd("restart", "Restart the selected server (also starts it if currently stopped)", (*api.Client).Restart))
	return cmd
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

func newServerStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the selected server's current status",
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
			gs, err := client.GetGameServer(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Server %d: %s — game: %s (%s), address: %s, slots: %d\n",
				id, gs.Status, gs.GameHuman, gs.Game, gs.Address(), gs.Slots)
			return nil
		},
	}
}

func newServerActionCmd(use, short string, action func(*api.Client, context.Context, int) error) *cobra.Command {
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
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := action(client, ctx, id); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: server %d.\n", use, id)
			return nil
		},
	}
}
