package certificates

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the certificates command group.
func Command() *ffcli.Command {
	return CertificatesCommand()
}
