package bundleids

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the bundle-ids command group.
func Command() *ffcli.Command {
	return BundleIDsCommand()
}
