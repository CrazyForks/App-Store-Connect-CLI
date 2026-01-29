package marketplace

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the marketplace command group.
func Command() *ffcli.Command {
	return MarketplaceCommand()
}
