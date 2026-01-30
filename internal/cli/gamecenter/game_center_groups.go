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

// GameCenterGroupsCommand returns the groups command group.
func GameCenterGroupsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("groups", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "groups",
		ShortUsage: "asc game-center groups <subcommand> [flags]",
		ShortHelp:  "Manage Game Center groups.",
		LongHelp: `Manage Game Center groups.

Examples:
  asc game-center groups list --app "APP_ID"
  asc game-center groups get --id "GROUP_ID"
  asc game-center groups create --reference-name "Group 1"
  asc game-center groups update --id "GROUP_ID" --reference-name "New Name"
  asc game-center groups delete --id "GROUP_ID" --confirm
  asc game-center groups achievements set --group-id "GROUP_ID" --ids "ACH_1,ACH_2"
  asc game-center groups leaderboards set --group-id "GROUP_ID" --ids "LB_1,LB_2"
  asc game-center groups challenges set --group-id "GROUP_ID" --ids "CH_1,CH_2"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterGroupsListCommand(),
			GameCenterGroupsGetCommand(),
			GameCenterGroupsCreateCommand(),
			GameCenterGroupsUpdateCommand(),
			GameCenterGroupsDeleteCommand(),
			GameCenterGroupAchievementsCommand(),
			GameCenterGroupLeaderboardsCommand(),
			GameCenterGroupChallengesCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterGroupsListCommand returns the groups list subcommand.
func GameCenterGroupsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("list", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID env)")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc game-center groups list [flags]",
		ShortHelp:  "List Game Center groups.",
		LongHelp: `List Game Center groups.

Examples:
  asc game-center groups list --app "APP_ID"
  asc game-center groups list --app "APP_ID" --limit 50
  asc game-center groups list --app "APP_ID" --paginate`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *limit != 0 && (*limit < 1 || *limit > 200) {
				return fmt.Errorf("game-center groups list: --limit must be between 1 and 200")
			}
			if err := validateNextURL(*next); err != nil {
				return fmt.Errorf("game-center groups list: %w", err)
			}

			resolvedAppID := resolveAppID(*appID)
			nextURL := strings.TrimSpace(*next)
			if resolvedAppID == "" && nextURL == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups list: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			gcDetailID := ""
			if nextURL == "" {
				var err error
				gcDetailID, err = client.GetGameCenterDetailID(requestCtx, resolvedAppID)
				if err != nil {
					return fmt.Errorf("game-center groups list: failed to get Game Center detail: %w", err)
				}
			}

			opts := []asc.GCGroupsOption{
				asc.WithGCGroupsLimit(*limit),
				asc.WithGCGroupsNextURL(*next),
			}
			if gcDetailID != "" {
				opts = append(opts, asc.WithGCGroupsGameCenterDetailIDs([]string{gcDetailID}))
			}

			if *paginate {
				paginateOpts := append(opts, asc.WithGCGroupsLimit(200))
				firstPage, err := client.GetGameCenterGroups(requestCtx, paginateOpts...)
				if err != nil {
					return fmt.Errorf("game-center groups list: failed to fetch: %w", err)
				}

				resp, err := asc.PaginateAll(requestCtx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
					return client.GetGameCenterGroups(ctx, asc.WithGCGroupsNextURL(nextURL))
				})
				if err != nil {
					return fmt.Errorf("game-center groups list: %w", err)
				}

				return printOutput(resp, *output, *pretty)
			}

			resp, err := client.GetGameCenterGroups(requestCtx, opts...)
			if err != nil {
				return fmt.Errorf("game-center groups list: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterGroupsGetCommand returns the groups get subcommand.
func GameCenterGroupsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	groupID := fs.String("id", "", "Game Center group ID")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc game-center groups get --id \"GROUP_ID\"",
		ShortHelp:  "Get a Game Center group by ID.",
		LongHelp: `Get a Game Center group by ID.

Examples:
  asc game-center groups get --id "GROUP_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups get: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetGameCenterGroup(requestCtx, id)
			if err != nil {
				return fmt.Errorf("game-center groups get: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterGroupsCreateCommand returns the groups create subcommand.
func GameCenterGroupsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("create", flag.ExitOnError)

	referenceName := fs.String("reference-name", "", "Reference name for the group")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "asc game-center groups create [flags]",
		ShortHelp:  "Create a Game Center group.",
		LongHelp: `Create a Game Center group.

Examples:
  asc game-center groups create --reference-name "Group 1"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			var ref *string
			if strings.TrimSpace(*referenceName) != "" {
				value := strings.TrimSpace(*referenceName)
				ref = &value
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups create: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.CreateGameCenterGroup(requestCtx, ref)
			if err != nil {
				return fmt.Errorf("game-center groups create: failed to create: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterGroupsUpdateCommand returns the groups update subcommand.
func GameCenterGroupsUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("update", flag.ExitOnError)

	groupID := fs.String("id", "", "Game Center group ID")
	referenceName := fs.String("reference-name", "", "Reference name for the group")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "asc game-center groups update --id \"GROUP_ID\" [flags]",
		ShortHelp:  "Update a Game Center group.",
		LongHelp: `Update a Game Center group.

Examples:
  asc game-center groups update --id "GROUP_ID" --reference-name "New Name"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}

			if strings.TrimSpace(*referenceName) == "" {
				fmt.Fprintln(os.Stderr, "Error: --reference-name is required")
				return flag.ErrHelp
			}
			value := strings.TrimSpace(*referenceName)

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups update: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.UpdateGameCenterGroup(requestCtx, id, &value)
			if err != nil {
				return fmt.Errorf("game-center groups update: failed to update: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}

// GameCenterGroupsDeleteCommand returns the groups delete subcommand.
func GameCenterGroupsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)

	groupID := fs.String("id", "", "Game Center group ID")
	confirm := fs.Bool("confirm", false, "Confirm deletion")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "asc game-center groups delete --id \"GROUP_ID\" --confirm",
		ShortHelp:  "Delete a Game Center group.",
		LongHelp: `Delete a Game Center group.

Examples:
  asc game-center groups delete --id "GROUP_ID" --confirm`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return flag.ErrHelp
			}
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups delete: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			if err := client.DeleteGameCenterGroup(requestCtx, id); err != nil {
				return fmt.Errorf("game-center groups delete: failed to delete: %w", err)
			}

			result := &asc.GameCenterGroupDeleteResult{
				ID:      id,
				Deleted: true,
			}

			return printOutput(result, *output, *pretty)
		},
	}
}

// GameCenterGroupAchievementsCommand returns the group achievements command group.
func GameCenterGroupAchievementsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("achievements", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "achievements",
		ShortUsage: "asc game-center groups achievements set --group-id \"GROUP_ID\" --ids \"ACH_1,ACH_2\"",
		ShortHelp:  "Manage group achievements relationships.",
		LongHelp: `Manage group achievements relationships.

Examples:
  asc game-center groups achievements set --group-id "GROUP_ID" --ids "ACH_1,ACH_2"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterGroupAchievementsSetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterGroupAchievementsSetCommand returns the group achievements set subcommand.
func GameCenterGroupAchievementsSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("set", flag.ExitOnError)

	groupID := fs.String("group-id", "", "Game Center group ID")
	ids := fs.String("ids", "", "Comma-separated achievement IDs")
	v2 := fs.Bool("v2", false, "Use v2 relationships endpoint")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "asc game-center groups achievements set --group-id \"GROUP_ID\" --ids \"ACH_1,ACH_2\"",
		ShortHelp:  "Replace group achievements relationships.",
		LongHelp: `Replace group achievements relationships.

Examples:
  asc game-center groups achievements set --group-id "GROUP_ID" --ids "ACH_1,ACH_2"
  asc game-center groups achievements set --group-id "GROUP_ID" --ids "ACH_1,ACH_2" --v2`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --group-id is required")
				return flag.ErrHelp
			}
			idsValue := splitCSV(*ids)
			if len(idsValue) == 0 {
				fmt.Fprintln(os.Stderr, "Error: --ids is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups achievements set: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			if *v2 {
				if err := client.UpdateGameCenterGroupAchievementsV2(requestCtx, id, idsValue); err != nil {
					return fmt.Errorf("game-center groups achievements set: failed to update: %w", err)
				}
			} else {
				if err := client.UpdateGameCenterGroupAchievements(requestCtx, id, idsValue); err != nil {
					return fmt.Errorf("game-center groups achievements set: failed to update: %w", err)
				}
			}

			result := &asc.LinkagesResponse{Data: resourceDataList(asc.ResourceTypeGameCenterAchievements, idsValue)}
			return printOutput(result, *output, *pretty)
		},
	}
}

// GameCenterGroupLeaderboardsCommand returns the group leaderboards command group.
func GameCenterGroupLeaderboardsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("leaderboards", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "leaderboards",
		ShortUsage: "asc game-center groups leaderboards set --group-id \"GROUP_ID\" --ids \"LB_1,LB_2\"",
		ShortHelp:  "Manage group leaderboards relationships.",
		LongHelp: `Manage group leaderboards relationships.

Examples:
  asc game-center groups leaderboards set --group-id "GROUP_ID" --ids "LB_1,LB_2"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterGroupLeaderboardsSetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterGroupLeaderboardsSetCommand returns the group leaderboards set subcommand.
func GameCenterGroupLeaderboardsSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("set", flag.ExitOnError)

	groupID := fs.String("group-id", "", "Game Center group ID")
	ids := fs.String("ids", "", "Comma-separated leaderboard IDs")
	v2 := fs.Bool("v2", false, "Use v2 relationships endpoint")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "asc game-center groups leaderboards set --group-id \"GROUP_ID\" --ids \"LB_1,LB_2\"",
		ShortHelp:  "Replace group leaderboards relationships.",
		LongHelp: `Replace group leaderboards relationships.

Examples:
  asc game-center groups leaderboards set --group-id "GROUP_ID" --ids "LB_1,LB_2"
  asc game-center groups leaderboards set --group-id "GROUP_ID" --ids "LB_1,LB_2" --v2`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --group-id is required")
				return flag.ErrHelp
			}
			idsValue := splitCSV(*ids)
			if len(idsValue) == 0 {
				fmt.Fprintln(os.Stderr, "Error: --ids is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups leaderboards set: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			if *v2 {
				if err := client.UpdateGameCenterGroupLeaderboardsV2(requestCtx, id, idsValue); err != nil {
					return fmt.Errorf("game-center groups leaderboards set: failed to update: %w", err)
				}
			} else {
				if err := client.UpdateGameCenterGroupLeaderboards(requestCtx, id, idsValue); err != nil {
					return fmt.Errorf("game-center groups leaderboards set: failed to update: %w", err)
				}
			}

			result := &asc.LinkagesResponse{Data: resourceDataList(asc.ResourceTypeGameCenterLeaderboards, idsValue)}
			return printOutput(result, *output, *pretty)
		},
	}
}

// GameCenterGroupChallengesCommand returns the group challenges command group.
func GameCenterGroupChallengesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("challenges", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "challenges",
		ShortUsage: "asc game-center groups challenges set --group-id \"GROUP_ID\" --ids \"CH_1,CH_2\"",
		ShortHelp:  "Manage group challenges relationships.",
		LongHelp: `Manage group challenges relationships.

Examples:
  asc game-center groups challenges set --group-id "GROUP_ID" --ids "CH_1,CH_2"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GameCenterGroupChallengesSetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GameCenterGroupChallengesSetCommand returns the group challenges set subcommand.
func GameCenterGroupChallengesSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("set", flag.ExitOnError)

	groupID := fs.String("group-id", "", "Game Center group ID")
	ids := fs.String("ids", "", "Comma-separated challenge IDs")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "asc game-center groups challenges set --group-id \"GROUP_ID\" --ids \"CH_1,CH_2\"",
		ShortHelp:  "Replace group challenges relationships.",
		LongHelp: `Replace group challenges relationships.

Examples:
  asc game-center groups challenges set --group-id "GROUP_ID" --ids "CH_1,CH_2"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			id := strings.TrimSpace(*groupID)
			if id == "" {
				fmt.Fprintln(os.Stderr, "Error: --group-id is required")
				return flag.ErrHelp
			}
			idsValue := splitCSV(*ids)
			if len(idsValue) == 0 {
				fmt.Fprintln(os.Stderr, "Error: --ids is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("game-center groups challenges set: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			if err := client.UpdateGameCenterGroupChallenges(requestCtx, id, idsValue); err != nil {
				return fmt.Errorf("game-center groups challenges set: failed to update: %w", err)
			}

			result := &asc.LinkagesResponse{Data: resourceDataList(asc.ResourceTypeGameCenterChallenges, idsValue)}
			return printOutput(result, *output, *pretty)
		},
	}
}
