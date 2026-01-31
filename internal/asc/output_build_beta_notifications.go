package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printBuildBetaNotificationTable(resp *BuildBetaNotificationResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID")
	fmt.Fprintf(w, "%s\n", resp.Data.ID)
	return w.Flush()
}

func printBuildBetaNotificationMarkdown(resp *BuildBetaNotificationResponse) error {
	fmt.Fprintln(os.Stdout, "| ID |")
	fmt.Fprintln(os.Stdout, "| --- |")
	fmt.Fprintf(os.Stdout, "| %s |\n", escapeMarkdown(resp.Data.ID))
	return nil
}
