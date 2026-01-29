package publish

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the publish command group.
func Command() *ffcli.Command {
	return PublishCommand()
}
