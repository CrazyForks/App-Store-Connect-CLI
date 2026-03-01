package xcodecloud

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// XcodeCloudIssuesCommand returns the xcode-cloud issues command with subcommands.
func XcodeCloudIssuesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("issues", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "issues",
		ShortUsage: "asc xcode-cloud issues <subcommand> [flags]",
		ShortHelp:  "List Xcode Cloud build issues.",
		LongHelp: `List Xcode Cloud build issues.

Examples:
  asc xcode-cloud issues list --action-id "ACTION_ID"
  asc xcode-cloud issues get --id "ISSUE_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			XcodeCloudIssuesListCommand(),
			XcodeCloudIssuesGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// XcodeCloudIssuesListCommand returns the xcode-cloud issues list subcommand.
func XcodeCloudIssuesListCommand() *ffcli.Command {
	return shared.NewPaginatedListCommand(shared.PaginatedListCommandConfig{
		FlagSetName: "list",
		Name:        "list",
		ShortUsage:  "asc xcode-cloud issues list [flags]",
		ShortHelp:   "List issues for a build action.",
		LongHelp: `List issues for a build action.

Examples:
  asc xcode-cloud issues list --action-id "ACTION_ID"
  asc xcode-cloud issues list --action-id "ACTION_ID" --output table
  asc xcode-cloud issues list --action-id "ACTION_ID" --limit 50
  asc xcode-cloud issues list --action-id "ACTION_ID" --paginate`,
		ParentFlag:  "action-id",
		ParentUsage: "Build action ID to list issues for",
		LimitMax:    200,
		ErrorPrefix: "xcode-cloud issues list",
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		FetchPage: func(ctx context.Context, client *asc.Client, actionID string, limit int, next string) (asc.PaginatedResponse, error) {
			opts := []asc.CiIssuesOption{
				asc.WithCiIssuesLimit(limit),
				asc.WithCiIssuesNextURL(next),
			}
			return client.GetCiBuildActionIssues(ctx, actionID, opts...)
		},
	})
}

// XcodeCloudIssuesGetCommand returns the xcode-cloud issues get subcommand.
func XcodeCloudIssuesGetCommand() *ffcli.Command {
	return shared.NewIDGetCommand(shared.IDGetCommandConfig{
		FlagSetName: "get",
		Name:        "get",
		ShortUsage:  "asc xcode-cloud issues get --id \"ISSUE_ID\"",
		ShortHelp:   "Get details for a build issue.",
		LongHelp: `Get details for a build issue.

Examples:
  asc xcode-cloud issues get --id "ISSUE_ID"
  asc xcode-cloud issues get --id "ISSUE_ID" --output table`,
		IDFlag:      "id",
		IDUsage:     "Issue ID",
		ErrorPrefix: "xcode-cloud issues get",
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		Fetch: func(ctx context.Context, client *asc.Client, id string) (any, error) {
			return client.GetCiIssue(ctx, id)
		},
	})
}
