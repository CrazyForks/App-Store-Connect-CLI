package localizations

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the localizations command group.
func Command() *ffcli.Command {
	return LocalizationsCommand()
}
