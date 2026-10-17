package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"nitui/internal/api"
)

// pollInterval is how often --wait/--watch re-check the server. A variable
// so tests don't have to sit through real delays.
var pollInterval = 3 * time.Second

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Control the lifecycle of the selected server",
	}
	cmd.AddCommand(newServerStatusCmd())
	cmd.AddCommand(newServerStopCmd())
	cmd.AddCommand(newServerRestartCmd())
	cmd.AddCommand(newServerUpdateCmd())
	cmd.AddCommand(newServerStatsCmd())
	return cmd
}

func formatStatusLine(id int, gs *api.GameServer) string {
	line := fmt.Sprintf("Server %d: %s — game: %s (%s), address: %s, slots: %d",
		id, gs.Status, gs.GameHuman, gs.Game, gs.Address(), gs.Slots)
	if u := gs.UpdateStatus(); u != "" {
		line += ", update: " + u
	}
	return line
}

func newServerStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the selected server's current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if watch, _ := cmd.Flags().GetBool("watch"); watch {
				return watchStatus(cmd.Context(), cmd.OutOrStdout(), id, func(ctx context.Context) (*api.GameServer, error) {
					return client.GetGameServer(ctx, id)
				}, pollInterval)
			}
			ctx, cancel := timeoutCtx(cmd, 15*time.Second)
			defer cancel()
			gs, err := client.GetGameServer(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			if wantJSON(cmd) {
				return printJSON(cmd, gs)
			}
			fmt.Fprintln(cmd.OutOrStdout(), formatStatusLine(id, gs))
			return nil
		},
	}
	cmd.Flags().Bool("watch", false, "keep running and print a line whenever the status changes (Ctrl-C to quit)")
	return cmd
}

// watchStatus prints the status line now and whenever it changes, until ctx
// is cancelled (Ctrl-C).
func watchStatus(ctx context.Context, out io.Writer, id int, fetch func(context.Context) (*api.GameServer, error), interval time.Duration) error {
	last := ""
	for {
		fctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		gs, err := fetch(fctx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Fprintln(out, "(status check failed:", apiErr(err), ")")
		} else if line := formatStatusLine(id, gs); line != last {
			fmt.Fprintf(out, "%s  %s\n", time.Now().Format("15:04:05"), line)
			last = line
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// lifecycleOpts are the flags shared by stop, restart and update.
type lifecycleOpts struct {
	wait    bool
	message string
	timeout time.Duration
}

func addLifecycleFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("wait", false, "wait until the server has actually reached the new state")
	cmd.Flags().String("message", "", "message shown to players before the action")
	cmd.Flags().Duration("timeout", 5*time.Minute, "how long --wait waits before giving up")
	addYesFlag(cmd)
}

func readLifecycleOpts(cmd *cobra.Command) lifecycleOpts {
	w, _ := cmd.Flags().GetBool("wait")
	m, _ := cmd.Flags().GetString("message")
	t, _ := cmd.Flags().GetDuration("timeout")
	return lifecycleOpts{wait: w, message: m, timeout: t}
}

// waitAndReport polls until want holds, printing each new status.
func waitAndReport(cmd *cobra.Command, client *api.Client, id int, opts api.WaitOptions, timeout time.Duration, doneMsg string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout+30*time.Second)
	defer cancel()
	opts.Timeout = timeout
	opts.Interval = pollInterval
	prev := ""
	opts.OnPoll = func(gs *api.GameServer) {
		if gs.Status != prev {
			fmt.Fprintf(cmd.OutOrStdout(), "  status: %s\n", gs.Status)
			prev = gs.Status
		}
	}
	_, err := api.WaitForStatus(ctx, func(c context.Context) (*api.GameServer, error) {
		return client.GetGameServer(c, id)
	}, opts)
	if err != nil {
		return fmt.Errorf("server %d: %w", id, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: server %d.\n", doneMsg, id)
	return nil
}

func isStarted(s string) bool { return s == api.StatusStarted }
func isStopped(s string) bool { return s == api.StatusStopped }

func newServerStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the selected server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			o := readLifecycleOpts(cmd)
			if err := confirmInteractive(cmd, fmt.Sprintf("Stop server %d?", id)); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.StopWithMessage(ctx, id, o.message); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stop: server %d.\n", id)
			if o.wait {
				return waitAndReport(cmd, client, id, api.WaitOptions{Want: isStopped}, o.timeout, "stopped")
			}
			return nil
		},
	}
	addLifecycleFlags(cmd)
	return cmd
}

func newServerRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the selected server (also starts it if currently stopped)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			return restartServer(cmd, client, id, "restart")
		},
	}
	addLifecycleFlags(cmd)
	return cmd
}

// restartServer restarts and (with --wait) waits for the server to be
// running again. Shared by `restart` and `update`.
func restartServer(cmd *cobra.Command, client *api.Client, id int, verb string) error {
	o := readLifecycleOpts(cmd)
	if err := confirmInteractive(cmd, fmt.Sprintf("Restart server %d?", id)); err != nil {
		return err
	}
	ctx, cancel := timeoutCtx(cmd, 30*time.Second)
	defer cancel()
	if err := client.RestartWithMessage(ctx, id, o.message); err != nil {
		return apiErr(err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: server %d.\n", verb, id)
	if o.wait {
		return waitAndReport(cmd, client, id, api.WaitOptions{
			Want:        isStarted,
			LeaveFirst:  api.StatusStarted,
			GracePeriod: 20 * time.Second,
		}, o.timeout, "restarted")
	}
	return nil
}

func newServerUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Restart the server so Nitrado applies any pending game update (unverified)",
		Long: "Nitrado updates game files automatically and exposes no dedicated \"update now\" endpoint.\n" +
			"This restarts the server, which is expected to pick up a pending update on start — that behavior has not been confirmed for every game.\n" +
			"`nitui server status` shows the last update status Nitrado reported, when there is one.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			return restartServer(cmd, client, id, "update (restart)")
		},
	}
	addLifecycleFlags(cmd)
	return cmd
}

func newServerStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show recent resource-usage statistics (raw JSON; shape unverified)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			hours, _ := cmd.Flags().GetInt("hours")
			ctx, cancel := timeoutCtx(cmd, 15*time.Second)
			defer cancel()
			raw, err := client.GetStats(ctx, id, hours)
			if err != nil {
				return apiErr(err)
			}
			var v any
			if json.Unmarshal(raw, &v) != nil {
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return nil
			}
			return printJSON(cmd, v)
		},
	}
	cmd.Flags().Int("hours", 24, "how many hours of history to fetch")
	return cmd
}
