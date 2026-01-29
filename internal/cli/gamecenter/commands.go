package gamecenter

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the game-center command group.
func Command() *ffcli.Command {
	return GameCenterCommand()
}
