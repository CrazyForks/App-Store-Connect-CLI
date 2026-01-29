package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/registry"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// RootCommand returns the root command
func RootCommand(version string) *ffcli.Command {
	root := &ffcli.Command{
		Name:        "asc",
		ShortUsage:  "asc <subcommand> [flags]",
		ShortHelp:   "A fast, AI-agent friendly CLI for App Store Connect.",
		LongHelp:    "ASC is a lightweight CLI for App Store Connect. Built for developers and AI agents.",
		FlagSet:     flag.NewFlagSet("asc", flag.ExitOnError),
		UsageFunc:   DefaultUsageFunc,
		Subcommands: registry.Subcommands(version),
	}

	versionFlag := root.FlagSet.Bool("version", false, "Print version and exit")
	shared.BindRootFlags(root.FlagSet)

	root.Exec = func(ctx context.Context, args []string) error {
		if *versionFlag {
			fmt.Fprintln(os.Stdout, version)
			return nil
		}
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		}
		return flag.ErrHelp
	}

	return root
}
