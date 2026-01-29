package nominations

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the nominations command group.
func Command() *ffcli.Command {
	return NominationsCommand()
}
