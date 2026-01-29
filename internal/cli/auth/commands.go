package auth

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the auth command group.
func Command() *ffcli.Command {
	return AuthCommand()
}
