package prerelease

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the pre-release versions command group.
func Command() *ffcli.Command {
	return PreReleaseVersionsCommand()
}
