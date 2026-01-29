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
	os.Exit(run())
}

func run() int {
	versionInfo := fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	root := cmd.RootCommand(versionInfo)
	defer cmd.CleanupTempPrivateKeys()

	if err := root.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		log.Printf("error parsing flags: %v", err)
		return 1
	}

	if err := root.Run(context.Background()); err != nil {
		var reported cmd.ReportedError
		if errors.As(err, &reported) {
			return 1
		}
		if errors.Is(err, flag.ErrHelp) {
			return 1
		}
		log.Printf("error executing command: %v", err)
		return 1
	}

	return 0
}
