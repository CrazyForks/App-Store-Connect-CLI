package builds

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the builds command group.
func Command() *ffcli.Command {
	return BuildsCommand()
}
