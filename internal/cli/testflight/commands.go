package testflight

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the testflight command group.
func Command() *ffcli.Command {
	return TestFlightCommand()
}
