package xcodecloud

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the Xcode Cloud command group.
func Command() *ffcli.Command {
	return XcodeCloudCommand()
}
