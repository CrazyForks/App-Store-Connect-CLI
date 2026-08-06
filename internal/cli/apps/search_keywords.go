package apps

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// AppsSearchKeywordsCommand returns the search keywords command group.
func AppsSearchKeywordsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("search-keywords", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "search-keywords",
		ShortUsage: "asc apps search-keywords <subcommand> [flags]",
		ShortHelp:  "Read search keywords for an app.",
		LongHelp: `Read search keywords for an app.

Apple exposes the app-level App Store Connect ` + "`searchKeywords`" + `
resource as read-only.

To update keyword text, use the canonical version-localization workflow under
` + "`asc metadata keywords ...`" + `.

Examples:
  asc apps search-keywords list --app "APP_ID"
  asc apps search-keywords list --app "APP_ID" --platform IOS --locale "en-US"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AppsSearchKeywordsListCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// AppsSearchKeywordsListCommand returns the search keywords list subcommand.
func AppsSearchKeywordsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apps search-keywords list", flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID env)")
	platform := fs.String("platform", "", "Filter by platform: IOS, MAC_OS, TV_OS, VISION_OS (comma-separated)")
	locale := fs.String("locale", "", "Filter by locale(s), comma-separated")
	limit := fs.Int("limit", 0, "Maximum results per page (1-200)")
	next := fs.String("next", "", "Fetch next page using a links.next URL")
	paginate := fs.Bool("paginate", false, "Automatically fetch all pages (aggregate results)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc apps search-keywords list --app \"APP_ID\"",
		ShortHelp:  "List search keywords for an app.",
		LongHelp: `List search keywords for an app.

Examples:
  asc apps search-keywords list --app "APP_ID"
  asc apps search-keywords list --app "APP_ID" --platform IOS --locale "en-US"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *limit != 0 && (*limit < 1 || *limit > 200) {
				return fmt.Errorf("apps search-keywords list: --limit must be between 1 and 200")
			}
			if err := shared.ValidateNextURL(*next); err != nil {
				return fmt.Errorf("apps search-keywords list: %w", err)
			}

			resolvedAppID := shared.ResolveAppID(*appID)
			if resolvedAppID == "" && strings.TrimSpace(*next) == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return shared.MissingRequiredUsageError()
			}

			platforms, err := shared.NormalizeAppStoreVersionPlatforms(shared.SplitCSVUpper(*platform))
			if err != nil {
				return fmt.Errorf("apps search-keywords list: %w", err)
			}

			locales := shared.SplitCSV(*locale)
			if err := shared.ValidateBuildLocalizationLocales(locales); err != nil {
				return fmt.Errorf("apps search-keywords list: %w", err)
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("apps search-keywords list: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			opts := []asc.AppSearchKeywordsOption{
				asc.WithAppSearchKeywordsLimit(*limit),
				asc.WithAppSearchKeywordsNextURL(*next),
			}
			if len(platforms) > 0 {
				opts = append(opts, asc.WithAppSearchKeywordsPlatforms(platforms))
			}
			if len(locales) > 0 {
				opts = append(opts, asc.WithAppSearchKeywordsLocales(locales))
			}

			if *paginate {
				paginateOpts := append(opts, asc.WithAppSearchKeywordsLimit(200))
				firstPage, err := client.GetAppSearchKeywords(requestCtx, resolvedAppID, paginateOpts...)
				if err != nil {
					return fmt.Errorf("apps search-keywords list: failed to fetch: %w", err)
				}

				resp, err := asc.PaginateAll(requestCtx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
					return client.GetAppSearchKeywords(ctx, resolvedAppID, asc.WithAppSearchKeywordsNextURL(nextURL))
				})
				if err != nil {
					return fmt.Errorf("apps search-keywords list: %w", err)
				}
				return shared.PrintOutput(resp, *output.Output, *output.Pretty)
			}

			resp, err := client.GetAppSearchKeywords(requestCtx, resolvedAppID, opts...)
			if err != nil {
				return fmt.Errorf("apps search-keywords list: failed to fetch: %w", err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}
