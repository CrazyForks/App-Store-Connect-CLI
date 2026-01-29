package encryption

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the encryption command group.
func Command() *ffcli.Command {
	return EncryptionCommand()
}
