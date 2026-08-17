// Command nitui controls Nitrado game servers from the terminal, either
// interactively (run with no arguments) or via scriptable subcommands.
package main

import "nitui/internal/cli"

func main() {
	cli.Execute()
}
