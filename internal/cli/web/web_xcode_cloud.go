package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

var newCIClientFn = webcore.NewCIClient

// WebXcodeCloudCommand returns the xcode-cloud command group.
func WebXcodeCloudCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "xcode-cloud",
		ShortUsage: "asc web xcode-cloud <subcommand> [flags]",
		ShortHelp:  "EXPERIMENTAL: Xcode Cloud compute usage reporting.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Query Xcode Cloud compute usage (plan quota, monthly/daily breakdowns, products)
using Apple's private CI API. Requires a web session.

` + webWarningText + `

Examples:
  asc web xcode-cloud usage summary --apple-id "user@example.com"
  asc web xcode-cloud products --apple-id "user@example.com" --output table
  asc web xcode-cloud usage months --apple-id "user@example.com" --output table
  asc web xcode-cloud usage days --product-id "UUID" --apple-id "user@example.com"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			webXcodeCloudUsageCommand(),
			webXcodeCloudProductsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func webXcodeCloudUsageCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud usage", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "usage",
		ShortUsage: "asc web xcode-cloud usage <subcommand> [flags]",
		ShortHelp:  "EXPERIMENTAL: Xcode Cloud usage queries.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Query Xcode Cloud compute usage: plan summary, monthly history, daily breakdown.

` + webWarningText,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			webXcodeCloudUsageSummaryCommand(),
			webXcodeCloudUsageMonthsCommand(),
			webXcodeCloudUsageDaysCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func webXcodeCloudUsageSummaryCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud usage summary", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "summary",
		ShortUsage: "asc web xcode-cloud usage summary [flags]",
		ShortHelp:  "EXPERIMENTAL: Show Xcode Cloud plan quota.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Show current Xcode Cloud plan usage: used/available/total compute minutes and reset date.

` + webWarningText + `

Examples:
  asc web xcode-cloud usage summary --apple-id "user@example.com"
  asc web xcode-cloud usage summary --apple-id "user@example.com" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			session, err := resolveWebSessionForCommand(ctx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage summary failed: session has no public provider ID")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			client := newCIClientFn(session)
			result, err := client.GetCIUsageSummary(requestCtx, teamID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage summary")
			}
			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func webXcodeCloudUsageMonthsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud usage months", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	now := time.Now()
	defaultEndMonth := int(now.Month())
	defaultEndYear := now.Year()
	past := now.AddDate(-1, 0, 0)
	defaultStartMonth := int(past.Month())
	defaultStartYear := past.Year()

	startMonth := fs.Int("start-month", defaultStartMonth, "Start month (1-12)")
	startYear := fs.Int("start-year", defaultStartYear, "Start year")
	endMonth := fs.Int("end-month", defaultEndMonth, "End month (1-12)")
	endYear := fs.Int("end-year", defaultEndYear, "End year")

	return &ffcli.Command{
		Name:       "months",
		ShortUsage: "asc web xcode-cloud usage months [flags]",
		ShortHelp:  "EXPERIMENTAL: Show monthly Xcode Cloud usage.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Show monthly Xcode Cloud compute usage with per-product breakdown.
Defaults to the last 12 months.

` + webWarningText + `

Examples:
  asc web xcode-cloud usage months --apple-id "user@example.com"
  asc web xcode-cloud usage months --apple-id "user@example.com" --start-month 1 --start-year 2025 --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *startMonth < 1 || *startMonth > 12 {
				fmt.Fprintln(os.Stderr, "Error: --start-month must be between 1 and 12")
				return flag.ErrHelp
			}
			if *endMonth < 1 || *endMonth > 12 {
				fmt.Fprintln(os.Stderr, "Error: --end-month must be between 1 and 12")
				return flag.ErrHelp
			}

			session, err := resolveWebSessionForCommand(ctx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage months failed: session has no public provider ID")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			client := newCIClientFn(session)
			result, err := client.GetCIUsageMonths(requestCtx, teamID, *startMonth, *startYear, *endMonth, *endYear)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage months")
			}
			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func webXcodeCloudUsageDaysCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud usage days", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	now := time.Now()
	defaultEnd := now.Format("2006-01-02")
	defaultStart := now.AddDate(0, 0, -30).Format("2006-01-02")

	productID := fs.String("product-id", "", "Xcode Cloud product ID (required)")
	start := fs.String("start", defaultStart, "Start date (YYYY-MM-DD)")
	end := fs.String("end", defaultEnd, "End date (YYYY-MM-DD)")

	return &ffcli.Command{
		Name:       "days",
		ShortUsage: "asc web xcode-cloud usage days --product-id ID [flags]",
		ShortHelp:  "EXPERIMENTAL: Show daily Xcode Cloud usage for a product.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Show daily Xcode Cloud compute usage for a specific product with per-workflow breakdown.
Defaults to the last 30 days.

` + webWarningText + `

Examples:
  asc web xcode-cloud usage days --product-id "UUID" --apple-id "user@example.com"
  asc web xcode-cloud usage days --product-id "UUID" --start 2025-01-01 --end 2025-01-31 --apple-id "user@example.com" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pid := strings.TrimSpace(*productID)
			if pid == "" {
				fmt.Fprintln(os.Stderr, "Error: --product-id is required")
				return flag.ErrHelp
			}
			if err := validateDateFlag("--start", *start); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return flag.ErrHelp
			}
			if err := validateDateFlag("--end", *end); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return flag.ErrHelp
			}

			session, err := resolveWebSessionForCommand(ctx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage days failed: session has no public provider ID")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			client := newCIClientFn(session)
			result, err := client.GetCIUsageDays(requestCtx, teamID, pid, *start, *end)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage days")
			}
			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func webXcodeCloudProductsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web xcode-cloud products", flag.ExitOnError)
	sessionFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "products",
		ShortUsage: "asc web xcode-cloud products [flags]",
		ShortHelp:  "EXPERIMENTAL: List Xcode Cloud products.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

List Xcode Cloud products (apps) for the authenticated team.
Use the product IDs with 'usage days' for per-product daily breakdowns.

` + webWarningText + `

Examples:
  asc web xcode-cloud products --apple-id "user@example.com"
  asc web xcode-cloud products --apple-id "user@example.com" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			session, err := resolveWebSessionForCommand(ctx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud products failed: session has no public provider ID")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			client := newCIClientFn(session)
			result, err := client.ListCIProducts(requestCtx, teamID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud products")
			}
			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func validateDateFlag(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s must be YYYY-MM-DD (got %q)", name, value)
	}
	return nil
}
