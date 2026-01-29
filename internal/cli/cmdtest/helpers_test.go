package cmdtest

import (
	"github.com/peterbourgon/ff/v3/ffcli"

	cmd "github.com/rudrankriyam/App-Store-Connect-CLI/cmd"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func RootCommand(version string) *ffcli.Command {
	return cmd.RootCommand(version)
}

func parseCommaSeparatedIDs(input string) []string {
	return shared.SplitCSV(input)
}

type ReportedError = shared.ReportedError
