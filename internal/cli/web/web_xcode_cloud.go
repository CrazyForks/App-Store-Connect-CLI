package web

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
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
			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage summary failed: session has no public provider ID")
			}

			client := newCIClientFn(session)
			result, err := client.GetCIUsageSummary(requestCtx, teamID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage summary")
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderCIUsageSummaryTable(result) },
				func() error { return renderCIUsageSummaryMarkdown(result) },
			)
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

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage months failed: session has no public provider ID")
			}

			client := newCIClientFn(session)
			result, err := client.GetCIUsageMonths(requestCtx, teamID, *startMonth, *startYear, *endMonth, *endYear)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage months")
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderCIUsageMonthsTable(result) },
				func() error { return renderCIUsageMonthsMarkdown(result) },
			)
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

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud usage days failed: session has no public provider ID")
			}

			client := newCIClientFn(session)
			result, err := client.GetCIUsageDays(requestCtx, teamID, pid, *start, *end)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud usage days")
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderCIUsageDaysTable(result) },
				func() error { return renderCIUsageDaysMarkdown(result) },
			)
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
			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, sessionFlags)
			if err != nil {
				return err
			}
			teamID := strings.TrimSpace(session.PublicProviderID)
			if teamID == "" {
				return fmt.Errorf("xcode-cloud products failed: session has no public provider ID")
			}

			client := newCIClientFn(session)
			result, err := client.ListCIProducts(requestCtx, teamID)
			if err != nil {
				return withWebAuthHint(err, "xcode-cloud products")
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderCIProductsTable(result) },
				func() error { return renderCIProductsMarkdown(result) },
			)
		},
	}
}

func renderCIUsageSummaryTable(result *webcore.CIUsageSummary) error {
	asc.RenderTable(
		[]string{"Plan", "Used", "Available", "Total", "Reset Date", "Reset Date Time", "Manage URL"},
		buildCIUsageSummaryRows(result),
	)
	return nil
}

func renderCIUsageSummaryMarkdown(result *webcore.CIUsageSummary) error {
	asc.RenderMarkdown(
		[]string{"Plan", "Used", "Available", "Total", "Reset Date", "Reset Date Time", "Manage URL"},
		buildCIUsageSummaryRows(result),
	)
	return nil
}

func buildCIUsageSummaryRows(result *webcore.CIUsageSummary) [][]string {
	if result == nil {
		result = &webcore.CIUsageSummary{}
	}
	return [][]string{
		{
			valueOrNA(result.Plan.Name),
			fmt.Sprintf("%d", result.Plan.Used),
			fmt.Sprintf("%d", result.Plan.Available),
			fmt.Sprintf("%d", result.Plan.Total),
			valueOrNA(result.Plan.ResetDate),
			valueOrNA(result.Plan.ResetDateTime),
			valueOrNA(result.Links["manage"]),
		},
	}
}

func renderCIUsageMonthsTable(result *webcore.CIUsageMonths) error {
	if result == nil {
		result = &webcore.CIUsageMonths{}
	}

	fmt.Printf("Range: %s\n", formatCIMonthRange(result.Usage, result.Info))
	fmt.Printf("Current: %d minutes (%d builds), avg30=%d\n", result.Info.Current.Used, result.Info.Current.Builds, result.Info.Current.Average30Days)
	fmt.Printf("Previous: %d minutes (%d builds), avg30=%d\n\n", result.Info.Previous.Used, result.Info.Previous.Builds, result.Info.Previous.Average30Days)
	asc.RenderTable([]string{"Year", "Month", "Minutes", "Builds"}, buildCIMonthUsageRows(result.Usage))

	if len(result.ProductUsage) > 0 {
		fmt.Println()
		asc.RenderTable(
			[]string{"Product ID", "Product Name", "Bundle ID", "Minutes", "Builds", "Prev Minutes", "Prev Builds"},
			buildCIProductUsageSummaryRows(result.ProductUsage),
		)
	}

	return nil
}

func renderCIUsageMonthsMarkdown(result *webcore.CIUsageMonths) error {
	if result == nil {
		result = &webcore.CIUsageMonths{}
	}

	fmt.Printf("**Range:** %s\n\n", formatCIMonthRange(result.Usage, result.Info))
	fmt.Printf("**Current:** %d minutes (%d builds), avg30=%d\n\n", result.Info.Current.Used, result.Info.Current.Builds, result.Info.Current.Average30Days)
	fmt.Printf("**Previous:** %d minutes (%d builds), avg30=%d\n\n", result.Info.Previous.Used, result.Info.Previous.Builds, result.Info.Previous.Average30Days)
	asc.RenderMarkdown([]string{"Year", "Month", "Minutes", "Builds"}, buildCIMonthUsageRows(result.Usage))

	if len(result.ProductUsage) > 0 {
		fmt.Println()
		asc.RenderMarkdown(
			[]string{"Product ID", "Product Name", "Bundle ID", "Minutes", "Builds", "Prev Minutes", "Prev Builds"},
			buildCIProductUsageSummaryRows(result.ProductUsage),
		)
	}

	return nil
}

func buildCIMonthUsageRows(usage []webcore.CIMonthUsage) [][]string {
	rows := make([][]string, 0, len(usage))
	for _, monthUsage := range usage {
		rows = append(rows, []string{
			fmt.Sprintf("%d", monthUsage.Year),
			fmt.Sprintf("%d", monthUsage.Month),
			fmt.Sprintf("%d", monthUsage.Duration),
			fmt.Sprintf("%d", monthUsage.NumberOfBuilds),
		})
	}
	return rows
}

func buildCIProductUsageSummaryRows(productUsage []webcore.CIProductUsage) [][]string {
	rows := make([][]string, 0)
	for _, product := range productUsage {
		minutes := product.UsageInMinutes
		builds := product.NumberOfBuilds
		if minutes == 0 && len(product.Usage) > 0 {
			for _, monthUsage := range product.Usage {
				minutes += monthUsage.Duration
				builds += monthUsage.NumberOfBuilds
			}
		}
		rows = append(rows, []string{
			valueOrNA(product.ProductID),
			valueOrNA(product.ProductName),
			valueOrNA(product.BundleID),
			fmt.Sprintf("%d", minutes),
			fmt.Sprintf("%d", builds),
			fmt.Sprintf("%d", product.PreviousUsageInMinutes),
			fmt.Sprintf("%d", product.PreviousNumberOfBuilds),
		})
	}
	return rows
}

func renderCIUsageDaysTable(result *webcore.CIUsageDays) error {
	if result == nil {
		result = &webcore.CIUsageDays{}
	}

	fmt.Printf("Range: %s\n", formatCIDayRange(result.Usage, result.Info))
	fmt.Printf("Current: %d minutes (%d builds), avg30=%d\n", result.Info.Current.Used, result.Info.Current.Builds, result.Info.Current.Average30Days)
	fmt.Printf("Previous: %d minutes (%d builds), avg30=%d\n\n", result.Info.Previous.Used, result.Info.Previous.Builds, result.Info.Previous.Average30Days)
	asc.RenderTable([]string{"Date", "Minutes", "Builds"}, buildCIDayUsageRows(result.Usage))

	if len(result.WorkflowUsage) > 0 {
		fmt.Println()
		asc.RenderTable(
			[]string{"Workflow ID", "Workflow Name", "Minutes", "Builds", "Prev Minutes", "Prev Builds"},
			buildCIWorkflowUsageRows(result.WorkflowUsage),
		)
	}

	return nil
}

func renderCIUsageDaysMarkdown(result *webcore.CIUsageDays) error {
	if result == nil {
		result = &webcore.CIUsageDays{}
	}

	fmt.Printf("**Range:** %s\n\n", formatCIDayRange(result.Usage, result.Info))
	fmt.Printf("**Current:** %d minutes (%d builds), avg30=%d\n\n", result.Info.Current.Used, result.Info.Current.Builds, result.Info.Current.Average30Days)
	fmt.Printf("**Previous:** %d minutes (%d builds), avg30=%d\n\n", result.Info.Previous.Used, result.Info.Previous.Builds, result.Info.Previous.Average30Days)
	asc.RenderMarkdown([]string{"Date", "Minutes", "Builds"}, buildCIDayUsageRows(result.Usage))

	if len(result.WorkflowUsage) > 0 {
		fmt.Println()
		asc.RenderMarkdown(
			[]string{"Workflow ID", "Workflow Name", "Minutes", "Builds", "Prev Minutes", "Prev Builds"},
			buildCIWorkflowUsageRows(result.WorkflowUsage),
		)
	}

	return nil
}

func buildCIDayUsageRows(usage []webcore.CIDayUsage) [][]string {
	rows := make([][]string, 0, len(usage))
	for _, dayUsage := range usage {
		rows = append(rows, []string{
			valueOrNA(dayUsage.Date),
			fmt.Sprintf("%d", dayUsage.Duration),
			fmt.Sprintf("%d", dayUsage.NumberOfBuilds),
		})
	}
	return rows
}

func buildCIWorkflowUsageRows(workflowUsage []webcore.CIWorkflowUsage) [][]string {
	rows := make([][]string, 0)
	for _, workflow := range workflowUsage {
		minutes := workflow.UsageInMinutes
		builds := workflow.NumberOfBuilds
		if minutes == 0 && len(workflow.Usage) > 0 {
			for _, dayUsage := range workflow.Usage {
				minutes += dayUsage.Duration
				builds += dayUsage.NumberOfBuilds
			}
		}
		rows = append(rows, []string{
			valueOrNA(workflow.WorkflowID),
			valueOrNA(workflow.WorkflowName),
			fmt.Sprintf("%d", minutes),
			fmt.Sprintf("%d", builds),
			fmt.Sprintf("%d", workflow.PreviousUsageInMinutes),
			fmt.Sprintf("%d", workflow.PreviousNumberOfBuilds),
		})
	}
	return rows
}

func renderCIProductsTable(result *webcore.CIProductListResponse) error {
	asc.RenderTable([]string{"Product ID", "Name", "Bundle ID", "Type"}, buildCIProductRows(result))
	return nil
}

func renderCIProductsMarkdown(result *webcore.CIProductListResponse) error {
	asc.RenderMarkdown([]string{"Product ID", "Name", "Bundle ID", "Type"}, buildCIProductRows(result))
	return nil
}

func buildCIProductRows(result *webcore.CIProductListResponse) [][]string {
	if result == nil {
		result = &webcore.CIProductListResponse{}
	}
	rows := make([][]string, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, []string{
			valueOrNA(item.ID),
			valueOrNA(item.Name),
			valueOrNA(item.BundleID),
			valueOrNA(item.Type),
		})
	}
	return rows
}

func formatCIMonthRange(usage []webcore.CIMonthUsage, info webcore.CIUsageInfo) string {
	if info.StartMonth < 1 || info.StartYear < 1 || info.EndMonth < 1 || info.EndYear < 1 {
		if len(usage) > 0 {
			first := usage[0]
			last := usage[len(usage)-1]
			return fmt.Sprintf("%04d-%02d to %04d-%02d", first.Year, first.Month, last.Year, last.Month)
		}
		return "n/a"
	}
	return fmt.Sprintf("%04d-%02d to %04d-%02d", info.StartYear, info.StartMonth, info.EndYear, info.EndMonth)
}

func formatCIDayRange(usage []webcore.CIDayUsage, info webcore.CIUsageInfo) string {
	if info.StartMonth > 0 && info.StartYear > 0 && info.EndMonth > 0 && info.EndYear > 0 {
		return fmt.Sprintf("%04d-%02d to %04d-%02d", info.StartYear, info.StartMonth, info.EndYear, info.EndMonth)
	}
	if len(usage) == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%s to %s", valueOrNA(usage[0].Date), valueOrNA(usage[len(usage)-1].Date))
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
