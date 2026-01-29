package submit

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the submit command group.
func Command() *ffcli.Command {
	return SubmitCommand()
}
