package sandbox

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// SandboxCommand returns the sandbox testers command with subcommands.
func SandboxCommand() *ffcli.Command {
	fs := flag.NewFlagSet("sandbox", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "sandbox",
		ShortUsage: "asc sandbox <subcommand> [flags]",
		ShortHelp:  "Manage App Store Connect sandbox testers.",
		LongHelp: `Manage sandbox testers for in-app purchase testing.

Examples:
  asc sandbox list
  asc sandbox list --email "tester@example.com"
  asc sandbox get --id "SANDBOX_TESTER_ID"
  asc sandbox update --id "SANDBOX_TESTER_ID" --territory "USA"
  asc sandbox clear-history --id "SANDBOX_TESTER_ID" --confirm
`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			SandboxListCommand(),
			SandboxGetCommand(),
			SandboxUpdateCommand(),
			SandboxClearHistoryCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
