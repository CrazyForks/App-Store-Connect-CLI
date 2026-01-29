package migrate

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the migrate command group.
func Command() *ffcli.Command {
	return MigrateCommand()
}
