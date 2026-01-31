package appclips

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// AppClipDefaultExperienceHeaderImageCommand returns the default experience header image command group.
func AppClipDefaultExperienceHeaderImageCommand() *ffcli.Command {
	fs := flag.NewFlagSet("header-image", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "header-image",
		ShortUsage: "asc app-clips default-experiences header-image <subcommand> [flags]",
		ShortHelp:  "Manage default experience header images.",
		LongHelp: `Manage default experience header images.

Examples:
  asc app-clips default-experiences header-image get --localization-id "LOCALIZATION_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AppClipDefaultExperienceHeaderImageGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// AppClipDefaultExperienceHeaderImageGetCommand retrieves the header image for a localization.
func AppClipDefaultExperienceHeaderImageGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("header-image get", flag.ExitOnError)

	localizationID := fs.String("localization-id", "", "Default experience localization ID")
	output := fs.String("output", "json", "Output format: json (default), table, markdown")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc app-clips default-experiences header-image get --localization-id \"LOCALIZATION_ID\"",
		ShortHelp:  "Get the header image for a localization.",
		LongHelp: `Get the header image for a localization.

Examples:
  asc app-clips default-experiences header-image get --localization-id "LOCALIZATION_ID"`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			locValue := strings.TrimSpace(*localizationID)
			if locValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --localization-id is required")
				return flag.ErrHelp
			}

			client, err := getASCClient()
			if err != nil {
				return fmt.Errorf("app-clips default-experiences header-image get: %w", err)
			}

			requestCtx, cancel := contextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetAppClipDefaultExperienceLocalizationHeaderImage(requestCtx, locValue)
			if err != nil {
				return fmt.Errorf("app-clips default-experiences header-image get: failed to fetch: %w", err)
			}

			return printOutput(resp, *output, *pretty)
		},
	}
}
