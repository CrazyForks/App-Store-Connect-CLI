package preorders

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the pre-orders command group.
func Command() *ffcli.Command {
	return PreOrdersCommand()
}
