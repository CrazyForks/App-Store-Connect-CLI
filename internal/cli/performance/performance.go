package performance

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// PerformanceCommand returns the performance command group.
func PerformanceCommand() *ffcli.Command {
	fs := flag.NewFlagSet("performance", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "performance",
		ShortUsage: "asc performance <subcommand> [flags]",
		ShortHelp:  "Access performance metrics and diagnostic logs.",
		LongHelp: `Access performance metrics and diagnostic logs.

Examples:
  asc performance metrics list --app "APP_ID"
  asc performance metrics get --build "BUILD_ID"
  asc performance diagnostics list --build "BUILD_ID"
  asc performance diagnostics get --id "SIGNATURE_ID"
  asc performance download --build "BUILD_ID" --output ./metrics.json`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			PerformanceMetricsCommand(),
			PerformanceDiagnosticsCommand(),
			PerformanceDownloadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
