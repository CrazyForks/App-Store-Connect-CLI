package versions

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the versions command group.
func Command() *ffcli.Command {
	return VersionsCommand()
}
