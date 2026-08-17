package cli

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newServersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers",
		Short: "List and inspect the servers on your account",
	}
	cmd.AddCommand(newServersListCmd())
	return cmd
}

func newServersListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every server on your account",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if len(services) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No servers on this account.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTATUS\tGAME\tADDRESS\tSLOTS")
			for _, s := range services {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", s.ID, s.Status, s.Details.Game, s.Details.Address, s.Details.Slots)
			}
			return w.Flush()
		},
	}
	return cmd
}
