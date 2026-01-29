package buildbundles

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the build-bundles command group.
func Command() *ffcli.Command {
	return BuildBundlesCommand()
}
