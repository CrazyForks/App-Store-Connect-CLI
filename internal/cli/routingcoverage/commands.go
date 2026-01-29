package routingcoverage

import "github.com/peterbourgon/ff/v3/ffcli"

// Command returns the routing-coverage command group.
func Command() *ffcli.Command {
	return RoutingCoverageCommand()
}
