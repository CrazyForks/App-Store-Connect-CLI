package offercodes

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the offer-codes command group.
func Command() *ffcli.Command {
	return OfferCodesCommand()
}
