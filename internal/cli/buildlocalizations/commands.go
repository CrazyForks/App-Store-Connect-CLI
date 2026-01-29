package buildlocalizations

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the build-localizations command group.
func Command() *ffcli.Command {
	return BuildLocalizationsCommand()
}
