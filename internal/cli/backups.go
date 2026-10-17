package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newBackupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backups",
		Short: "List, create, restore and delete server backups",
	}
	cmd.AddCommand(newBackupsListCmd(), newBackupsCreateCmd(), newBackupsRestoreCmd(), newBackupsDeleteCmd())
	return cmd
}

func newBackupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List backups of the selected server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			backups, err := client.ListBackups(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			if wantJSON(cmd) {
				return printJSON(cmd, backups)
			}
			if len(backups) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No backups found.")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSIZE\tCREATED")
			for _, b := range backups {
				fmt.Fprintf(w, "%s\t%s\t%s\n", b.Name, b.Size, b.Created)
			}
			return w.Flush()
		},
	}
}

func newBackupsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup now",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			kind, _ := cmd.Flags().GetString("type")
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.CreateBackup(ctx, id, kind); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Backup requested for server %d. It may take a few minutes to appear in `nitui backups list`.\n", id)
			return nil
		},
	}
	cmd.Flags().String("type", "game", "backup type")
	return cmd
}

func newBackupsRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore a backup (overwrites the current server files)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if err := confirm(cmd, fmt.Sprintf("Restore backup %q onto server %d, overwriting its current files?", args[0], id)); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.RestoreBackup(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restoring %q on server %d.\n", args[0], id)
			return nil
		},
	}
	addYesFlag(cmd)
	return cmd
}

func newBackupsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if err := confirm(cmd, fmt.Sprintf("Delete backup %q?", args[0])); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.DeleteBackup(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted backup %q.\n", args[0])
			return nil
		},
	}
	addYesFlag(cmd)
	return cmd
}
