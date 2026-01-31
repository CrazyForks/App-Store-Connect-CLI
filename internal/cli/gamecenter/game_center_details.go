package gamecenter

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

// GameCenterDetailsCommand returns the details command group.
func GameCenterDetailsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("details", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "details",
		ShortUsage: "asc game-center details <subcommand> [flags]",
		ShortHelp:  "Manage Game Center details.",
		LongHelp: `Manage Game Center details.

Examples:
  asc game-center details list --app "APP_ID"
  asc game-center details get --id "DETAIL_ID"
  asc game-center details app-versions list --id "DETAIL_ID"
  asc game-center details group get --id "DETAIL_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterDetailsListCommand(),
			GameCenterDetailsGetCommand(),
			GameCenterDetailsAppVersionsCommand(),
			GameCenterDetailsGroupCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterDetailsListCommand returns the details list subcommand.
func GameCenterDetailsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID env)")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc game-center details list [flags]",
		ShortHelp:  "List Game Center details.",
		LongHelp: `List Game Center details.

Examples:
  asc game-center details list --app "APP_ID"
  asc game-center details list --app "APP_ID" --paginate`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			nextURL := strings.TrimSpace(*next)
			if nextURL != "" {
				return fmt.Errorf("game-center details list: --next is not supported")
			}
			if *paginate {
				return fmt.Errorf("game-center details list: --paginate is not supported")
			}
			if *limit != 0 {
				return fmt.Errorf("game-center details list: --limit is not supported")
			}

			resolvedAppID := resolveAppID(*appID)
			if resolvedAppID == "" && nextURL == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center details list: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			detailID, err := client.GetGameCenterDetailID(requestCtx, resolvedAppID)
			if err != nil {
				return fmt.Errorf("game-center details list: failed to get Game Center detail: %w", err)
			}

			detail, err := client.GetGameCenterDetail(requestCtx, detailID)
			if err != nil {
				return fmt.Errorf("game-center details list: failed to fetch: %w", err)
			}

			resp := &asc.GameCenterDetailsResponse{
				Data:     []asc.Resource[asc.GameCenterDetailAttributes]{detail.Data},
				Links:    detail.Links,
				Included: detail.Included,
				Meta:     detail.Meta,
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterDetailsGetCommand returns the details get subcommand.
func GameCenterDetailsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	detailID := fs.String("id", "", "Game Center detail ID")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc game-center details get --id \"DETAIL_ID\"",
		ShortHelp:  "Get a Game Center detail by ID.",
		LongHelp: `Get a Game Center detail by ID.

Examples:
  asc game-center details get --id "DETAIL_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*detailID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center details get: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetGameCenterDetail(requestCtx, id)
			if err != nil {
				return fmt.Errorf("game-center details get: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterDetailsAppVersionsCommand returns the details app-versions command group.
func GameCenterDetailsAppVersionsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-versions", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "app-versions",
		ShortUsage: "asc game-center details app-versions list --id \"DETAIL_ID\"",
		ShortHelp:  "List Game Center app versions for a detail.",
		LongHelp: `List Game Center app versions for a detail.

Examples:
  asc game-center details app-versions list --id "DETAIL_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterDetailsAppVersionsListCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterDetailsAppVersionsListCommand returns the details app-versions list subcommand.
func GameCenterDetailsAppVersionsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ExitOnError)

	detailID := fs.String("id", "", "Game Center detail ID")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc game-center details app-versions list --id \"DETAIL_ID\"",
		ShortHelp:  "List Game Center app versions for a detail.",
		LongHelp: `List Game Center app versions for a detail.

Examples:
  asc game-center details app-versions list --id "DETAIL_ID"
  asc game-center details app-versions list --id "DETAIL_ID" --limit 50
  asc game-center details app-versions list --id "DETAIL_ID" --paginate`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *limit != 0 && (*limit < 1 || *limit > 200) {
				return fmt.Errorf("game-center details app-versions list: --limit must be between 1 and 200")
			}
			if err := validateNextURL(*next); err != nil {
				return fmt.Errorf("game-center details app-versions list: %w", err)
			}

			id := strings.TrimSpace(*detailID)
			nextURL := strings.TrimSpace(*next)
			if id == "" && nextURL == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center details app-versions list: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			opts := []asc.GCAppVersionsOption{
				asc.WithGCAppVersionsLimit(*limit),
				asc.WithGCAppVersionsNextURL(*next),
			}

			if *paginate {
				paginateOpts := []asc.GCAppVersionsOption{asc.WithGCAppVersionsNextURL(*next)}
				if nextURL == "" {
					paginateOpts = []asc.GCAppVersionsOption{asc.WithGCAppVersionsLimit(200)}
				}
				firstPage, err := client.GetGameCenterDetailGameCenterAppVersions(requestCtx, id, paginateOpts...)
				if err != nil {
					return fmt.Errorf("game-center details app-versions list: failed to fetch: %w", err)
				}

				resp, err := asc.PaginateAll(requestCtx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
					return client.GetGameCenterDetailGameCenterAppVersions(ctx, id, asc.WithGCAppVersionsNextURL(nextURL))
				})
				if err != nil {
					return fmt.Errorf("game-center details app-versions list: %w", err)
				}

				return printOutput(resp, *output, *pretty)
			}

			resp, err := client.GetGameCenterDetailGameCenterAppVersions(requestCtx, id, opts...)
			if err != nil {
				return fmt.Errorf("game-center details app-versions list: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterDetailsGroupCommand returns the details group command group.
func GameCenterDetailsGroupCommand() *ffcli.Command {
	fs := flag.NewFlagSet("group", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "group",
		ShortUsage: "asc game-center details group get --id \"DETAIL_ID\"",
		ShortHelp:  "Get the Game Center group for a detail.",
		LongHelp: `Get the Game Center group for a detail.

Examples:
  asc game-center details group get --id "DETAIL_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterDetailsGroupGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterDetailsGroupGetCommand returns the details group get subcommand.
func GameCenterDetailsGroupGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	detailID := fs.String("id", "", "Game Center detail ID")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc game-center details group get --id \"DETAIL_ID\"",
		ShortHelp:  "Get the Game Center group for a detail.",
		LongHelp: `Get the Game Center group for a detail.

Examples:
  asc game-center details group get --id "DETAIL_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*detailID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center details group get: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetGameCenterDetailGameCenterGroup(requestCtx, id)
			if err != nil {
				return fmt.Errorf("game-center details group get: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}
