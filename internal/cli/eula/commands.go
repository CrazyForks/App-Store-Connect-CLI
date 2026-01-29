package eula

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the eula command group.
func Command() *ffcli.Command {
	return EULACommand()
}
