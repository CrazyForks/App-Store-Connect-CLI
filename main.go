package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rudrankriyam/App-Store-Connect-CLI/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	versionInfo := fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	root := cmd.RootCommand(versionInfo)
	defer cmd.CleanupTempPrivateKeys()

	if err := root.Parse(os.Args[1:]); err != nil {
		cmd.CleanupTempPrivateKeys()
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		log.Fatalf("error parsing flags: %v\n", err)
	}

	if err := root.Run(context.Background()); err != nil {
		var reported cmd.ReportedError
		if errors.As(err, &reported) {
			cmd.CleanupTempPrivateKeys()
			os.Exit(1)
		}
		if errors.Is(err, flag.ErrHelp) {
			cmd.CleanupTempPrivateKeys()
			os.Exit(1)
		}
		cmd.CleanupTempPrivateKeys()
		log.Fatalf("error executing command: %v\n", err)
	}
}
