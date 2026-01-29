package sandbox

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the sandbox command group.
func Command() *ffcli.Command {
	return SandboxCommand()
}
