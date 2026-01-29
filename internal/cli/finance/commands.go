package finance

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the finance command group.
func Command() *ffcli.Command {
	return FinanceCommand()
}
