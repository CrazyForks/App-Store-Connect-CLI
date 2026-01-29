package assets

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the assets command group.
func Command() *ffcli.Command {
	return AssetsCommand()
}
