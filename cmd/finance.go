package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

// FinanceCommand returns the finance command with subcommands.
func FinanceCommand() *ffcli.Command {
	fs := flag.NewFlagSet("finance", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "finance",
		ShortUsage: "asc finance <subcommand> [flags]",
		ShortHelp:  "Download payments and financial reports.",
		LongHelp: `Download payments and financial reports.

Finance reports are monthly and available through the App Store Connect API.
Requires Account Holder, Admin, or Finance role.

Examples:
  asc finance reports --vendor "12345678" --report-type FINANCIAL --region "US" --date "2025-12"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			FinanceReportsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// FinanceReportsCommand downloads finance reports.
func FinanceReportsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("reports", flag.ExitOnError)

	vendor := fs.String("vendor", "", "Vendor number (or ASC_VENDOR_NUMBER env)")
	reportType := fs.String("report-type", "", "Report type: FINANCIAL or FINANCE_DETAIL")
	region := fs.String("region", "", "Region code (e.g., US)")
	date := fs.String("date", "", "Report date (YYYY-MM)")
	output := fs.String("output", "", "Output file path (default: finance_report_{date}_{type}_{region}.tsv.gz)")
	decompress := fs.Bool("decompress", false, "Decompress gzip output to .tsv")
	outputFormat := fs.String("output-format", "json", "Output format for metadata: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "reports",
		ShortUsage: "asc finance reports [flags]",
		ShortHelp:  "Download financial reports from App Store Connect.",
		LongHelp: `Download financial reports from App Store Connect.

Requires Account Holder, Admin, or Finance role.

Examples:
  asc finance reports --vendor "12345678" --report-type FINANCIAL --region "US" --date "2025-12"
  asc finance reports --vendor "12345678" --report-type FINANCE_DETAIL --region "US" --date "2025-12" --decompress
  asc finance reports --vendor "12345678" --report-type FINANCIAL --region "US" --date "2025-12" --output "reports/finance.tsv.gz"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			vendorNumber := resolveVendorNumber(*vendor)
			if vendorNumber == "" {
				fmt.Fprintln(os.Stderr, "Error: --vendor is required (or set ASC_VENDOR_NUMBER)")
				return flag.ErrHelp
			}
			if strings.TrimSpace(*reportType) == "" {
				fmt.Fprintln(os.Stderr, "Error: --report-type is required")
				return flag.ErrHelp
			}
			if strings.TrimSpace(*region) == "" {
				fmt.Fprintln(os.Stderr, "Error: --region is required")
				return flag.ErrHelp
			}
			if strings.TrimSpace(*date) == "" {
				fmt.Fprintln(os.Stderr, "Error: --date is required")
				return flag.ErrHelp
			}

			normalizedReportType, err := normalizeFinanceReportType(*reportType)
			if err != nil {
				return fmt.Errorf("finance reports: %w", err)
			}
			reportDate, err := normalizeFinanceReportDate(*date)
			if err != nil {
				return fmt.Errorf("finance reports: %w", err)
			}

			regionCode := strings.ToUpper(strings.TrimSpace(*region))
			defaultOutput := fmt.Sprintf("finance_report_%s_%s_%s.tsv.gz", reportDate, string(normalizedReportType), regionCode)
			compressedPath, decompressedPath := resolveReportOutputPaths(*output, defaultOutput, ".tsv", *decompress)

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("finance reports: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			download, err := client.DownloadFinanceReport(requestCtx, asc.FinanceReportParams{
				VendorNumber: vendorNumber,
				ReportType:   normalizedReportType,
				RegionCode:   regionCode,
				ReportDate:   reportDate,
			})
			if err != nil {
				return fmt.Errorf("finance reports: failed to download report: %w", err)
			}
			defer download.Body.Close()

			compressedSize, err := writeStreamToFile(compressedPath, download.Body)
			if err != nil {
				return fmt.Errorf("finance reports: failed to write report: %w", err)
			}

			var decompressedSize int64
			if *decompress {
				decompressedSize, err = decompressGzipFile(compressedPath, decompressedPath)
				if err != nil {
					return fmt.Errorf("finance reports: %w", err)
				}
			}

			result := &asc.FinanceReportResult{
				VendorNumber:      vendorNumber,
				ReportType:        string(normalizedReportType),
				RegionCode:        regionCode,
				ReportDate:        reportDate,
				FilePath:          compressedPath,
				Bytes:             compressedSize,
				Decompressed:      *decompress,
				DecompressedPath:  decompressedPath,
				DecompressedBytes: decompressedSize,
			}

			return printOutput(result, *outputFormat, *pretty)
		},
	}
}
