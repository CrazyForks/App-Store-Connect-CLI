package crashes

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the crashes command group.
func Command() *ffcli.Command {
	return CrashesCommand()
}
