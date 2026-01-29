package performance

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the performance command group.
func Command() *ffcli.Command {
	return PerformanceCommand()
}
