package xcodecloud

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// XcodeCloudTestResultsCommand returns the xcode-cloud test-results command with subcommands.
func XcodeCloudTestResultsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("test-results", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "test-results",
		ShortUsage: "asc xcode-cloud test-results <subcommand> [flags]",
		ShortHelp:  "List Xcode Cloud test results.",
		LongHelp: `List Xcode Cloud test results.

Examples:
  asc xcode-cloud test-results list --action-id "ACTION_ID"
  asc xcode-cloud test-results get --id "TEST_RESULT_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			XcodeCloudTestResultsListCommand(),
			XcodeCloudTestResultsGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// XcodeCloudTestResultsListCommand returns the xcode-cloud test-results list subcommand.
func XcodeCloudTestResultsListCommand() *ffcli.Command {
	return shared.NewPaginatedListCommand(shared.PaginatedListCommandConfig{
		FlagSetName: "list",
		Name:        "list",
		ShortUsage:  "asc xcode-cloud test-results list [flags]",
		ShortHelp:   "List test results for a build action.",
		LongHelp: `List test results for a build action.

Examples:
  asc xcode-cloud test-results list --action-id "ACTION_ID"
  asc xcode-cloud test-results list --action-id "ACTION_ID" --output table
  asc xcode-cloud test-results list --action-id "ACTION_ID" --limit 50
  asc xcode-cloud test-results list --action-id "ACTION_ID" --paginate`,
		ParentFlag:  "action-id",
		ParentUsage: "Build action ID to list test results for",
		LimitMax:    200,
		ErrorPrefix: "xcode-cloud test-results list",
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		FetchPage: func(ctx context.Context, client *asc.Client, actionID string, limit int, next string) (asc.PaginatedResponse, error) {
			opts := []asc.CiTestResultsOption{
				asc.WithCiTestResultsLimit(limit),
				asc.WithCiTestResultsNextURL(next),
			}
			return client.GetCiBuildActionTestResults(ctx, actionID, opts...)
		},
	})
}

// XcodeCloudTestResultsGetCommand returns the xcode-cloud test-results get subcommand.
func XcodeCloudTestResultsGetCommand() *ffcli.Command {
	return shared.NewIDGetCommand(shared.IDGetCommandConfig{
		FlagSetName: "get",
		Name:        "get",
		ShortUsage:  "asc xcode-cloud test-results get --id \"TEST_RESULT_ID\"",
		ShortHelp:   "Get details for a test result.",
		LongHelp: `Get details for a test result.

Examples:
  asc xcode-cloud test-results get --id "TEST_RESULT_ID"
  asc xcode-cloud test-results get --id "TEST_RESULT_ID" --output table`,
		IDFlag:      "id",
		IDUsage:     "Test result ID",
		ErrorPrefix: "xcode-cloud test-results get",
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		Fetch: func(ctx context.Context, client *asc.Client, id string) (any, error) {
			return client.GetCiTestResult(ctx, id)
		},
	})
}
