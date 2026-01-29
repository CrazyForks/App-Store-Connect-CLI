package feedback

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the feedback command group.
func Command() *ffcli.Command {
	return FeedbackCommand()
}
