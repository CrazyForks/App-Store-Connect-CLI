package profiles

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the profiles command group.
func Command() *ffcli.Command {
	return ProfilesCommand()
}
