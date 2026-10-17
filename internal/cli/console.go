package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "command <console command...>",
		Short: "Send a console command to the running game server",
		Long:  "Send a console command to the running game server, e.g. `nitui command say hello`. The server must be running and the game must support remote commands.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			line := strings.Join(args, " ")
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			if err := client.SendCommand(ctx, id, line); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sent to server %d: %s\n", id, line)
			return nil
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Show the server's admin log",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			lines, err := client.AdminLogs(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			if wantJSON(cmd) {
				return printJSON(cmd, lines)
			}
			if len(lines) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No log entries.")
				return nil
			}
			for _, l := range lines {
				fmt.Fprintln(cmd.OutOrStdout(), l)
			}
			return nil
		},
	}
}
