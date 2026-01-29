package apps

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the apps command group.
func Command() *ffcli.Command {
	return AppsCommand()
}
