package accessibility

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the accessibility command group.
func Command() *ffcli.Command {
	return AccessibilityCommand()
}
