package signing

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the signing command group.
func Command() *ffcli.Command {
	return SigningCommand()
}
