package assets

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// AssetsCommand returns the assets command with subcommands.
func AssetsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("assets", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "assets",
		ShortUsage: "asc assets <subcommand> [flags]",
		ShortHelp:  "Manage App Store assets (screenshots, previews).",
		LongHelp: `Manage App Store metadata assets (screenshots and app previews).

Examples:
  asc assets screenshots list --version-localization "LOC_ID"
  asc assets screenshots upload --version-localization "LOC_ID" --path "./screenshots" --device-type "IPHONE_65"
  asc assets previews upload --version-localization "LOC_ID" --path "./previews" --device-type "IPHONE_65"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AssetsScreenshotsCommand(),
			AssetsPreviewsCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
