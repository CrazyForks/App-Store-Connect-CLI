package agerating

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the age-rating command group.
func Command() *ffcli.Command {
	return AgeRatingCommand()
}
