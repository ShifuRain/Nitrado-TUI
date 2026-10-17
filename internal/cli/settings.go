package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"nitui/internal/api"
)

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "View and change the selected server's game settings",
		Long: "View and change the selected server's game settings.\n" +
			"Settings are per game: the categories and keys depend on the installed game. Most changes only take effect after a restart (`nitui server restart`).",
	}
	cmd.AddCommand(newSettingsListCmd())
	cmd.AddCommand(newSettingsGetCmd())
	cmd.AddCommand(newSettingsSetCmd())
	cmd.AddCommand(newSettingsResetCmd())
	return cmd
}

func newSettingsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [category]",
		Short: "List settings (a * marks values changed from the default)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			gs, err := client.GetGameServer(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			defaults, _ := client.GetSettingsDefaults(ctx, id) // best-effort

			cats := gs.Settings.Categories()
			if len(args) == 1 {
				if _, ok := gs.Settings[args[0]]; !ok {
					return fmt.Errorf("no settings category %q — run `nitui settings list` to see them", args[0])
				}
				cats = []string{args[0]}
			}
			if len(cats) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No settings reported for this server.")
				return nil
			}

			if wantJSON(cmd) {
				out := api.Settings{}
				for _, c := range cats {
					out[c] = gs.Settings[c]
				}
				return printJSON(cmd, out)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "CATEGORY\tKEY\tVALUE\tDEFAULT")
			for _, c := range cats {
				for _, k := range gs.Settings.Keys(c) {
					val, _ := gs.Settings.Get(c, k)
					def, hasDef := defaults.Get(c, k)
					mark := ""
					if hasDef && def != val {
						mark = " *"
					}
					fmt.Fprintf(w, "%s\t%s\t%s%s\t%s\n", c, k, val, mark, def)
				}
			}
			return w.Flush()
		},
	}
}

func newSettingsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <category> <key>",
		Short: "Print one setting's current value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			gs, err := client.GetGameServer(ctx, id)
			if err != nil {
				return apiErr(err)
			}
			val, ok := gs.Settings.Get(args[0], args[1])
			if !ok {
				return fmt.Errorf("no setting %s/%s — run `nitui settings list` to see valid ones", args[0], args[1])
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newSettingsSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <category> <key> <value>",
		Short: "Change one setting",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			if err := client.SetSetting(ctx, id, args[0], args[1], args[2]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s/%s on server %d. Restart the server for it to take effect (`nitui server restart`).\n", args[0], args[1], id)
			return nil
		},
	}
	return cmd
}

func newSettingsResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset ALL settings to their defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if err := confirm(cmd, fmt.Sprintf("Reset every setting on server %d to its default?", id)); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			if err := client.ResetSettings(ctx, id); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Reset settings on server %d.\n", id)
			return nil
		},
	}
	addYesFlag(cmd)
	return cmd
}
