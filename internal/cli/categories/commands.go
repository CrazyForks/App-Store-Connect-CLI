package categories

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the categories command group.
func Command() *ffcli.Command {
	return CategoriesCommand()
}
