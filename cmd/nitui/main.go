// Command nitui controls Nitrado game servers from the terminal, either
// interactively (run with no arguments) or via scriptable subcommands.
package main

import "nitui/internal/cli"

// Set via -ldflags by goreleaser at release time; "dev" for local builds.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date})
}
