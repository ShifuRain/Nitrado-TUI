package cli

import (
	"fmt"
	"os"
	pathpkg "path"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Browse and manage files on the selected server",
		Long:  "Browse and manage files on the selected server's file server (the same files you would reach over FTP). Paths use forward slashes, e.g. /gameserver/config.json.",
	}
	cmd.AddCommand(newFilesLsCmd(), newFilesCatCmd(), newFilesGetCmd(), newFilesPutCmd(), newFilesRmCmd(), newFilesMkdirCmd())
	return cmd
}

func newFilesLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [dir]",
		Short: "List a directory (default: the root)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			dir := "/"
			if len(args) == 1 {
				dir = args[0]
			}
			ctx, cancel := timeoutCtx(cmd, 20*time.Second)
			defer cancel()
			entries, err := client.ListFiles(ctx, id, dir)
			if err != nil {
				return apiErr(err)
			}
			if wantJSON(cmd) {
				return printJSON(cmd, entries)
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(empty)")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tSIZE\tNAME")
			for _, e := range entries {
				name := e.Name
				size := fmt.Sprint(e.Size)
				if e.IsDir() {
					name += "/"
					size = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.Type, size, name)
			}
			return w.Flush()
		},
	}
}

func newFilesCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat <path>",
		Short: "Print a remote file to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 2*time.Minute)
			defer cancel()
			data, err := client.ReadFile(ctx, id, args[0])
			if err != nil {
				return apiErr(err)
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func newFilesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <remote_path> [local_path]",
		Short: "Download a remote file (default: into the current directory)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			local := pathpkg.Base(args[0])
			if len(args) == 2 {
				local = args[1]
			}
			ctx, cancel := timeoutCtx(cmd, 5*time.Minute)
			defer cancel()
			data, err := client.ReadFile(ctx, id, args[0])
			if err != nil {
				return apiErr(err)
			}
			if err := os.WriteFile(local, data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s (%d bytes).\n", local, len(data))
			return nil
		},
	}
}

func newFilesPutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "put <local_path> <remote_path>",
		Short: "Upload a local file to the full remote path (overwrites)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			dir, name := pathpkg.Split(args[1])
			if name == "" {
				return fmt.Errorf("remote_path %q must include a file name", args[1])
			}
			if dir == "" {
				dir = "/"
			}
			ctx, cancel := timeoutCtx(cmd, 5*time.Minute)
			defer cancel()
			if err := client.WriteFile(ctx, id, dir, name, data); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Uploaded %s to %s (%d bytes).\n", args[0], args[1], len(data))
			return nil
		},
	}
}

func newFilesRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <path>",
		Short: "Delete a remote file or directory (recursively)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			if err := confirm(cmd, fmt.Sprintf("Delete %s from server %d?", args[0], id)); err != nil {
				return err
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.DeleteFile(ctx, id, args[0]); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s.\n", args[0])
			return nil
		},
	}
	addYesFlag(cmd)
	return cmd
}

func newFilesMkdirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir <path>",
		Short: "Create a remote directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, id, err := prepare(cmd)
			if err != nil {
				return err
			}
			dir, name := pathpkg.Split(pathpkg.Clean(args[0]))
			if name == "" || name == "." || name == "/" {
				return fmt.Errorf("path %q must name a directory to create", args[0])
			}
			if dir == "" {
				dir = "/"
			}
			ctx, cancel := timeoutCtx(cmd, 30*time.Second)
			defer cancel()
			if err := client.MakeDir(ctx, id, dir, name); err != nil {
				return apiErr(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s.\n", args[0])
			return nil
		},
	}
}
