package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printNominationsTable(resp *NominationsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tType\tState\tPublish Start\tPublish End")
	for _, item := range resp.Data {
		attrs := item.Attributes
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			sanitizeTerminal(item.ID),
			compactWhitespace(fallbackValue(attrs.Name)),
			sanitizeTerminal(fallbackValue(string(attrs.Type))),
			sanitizeTerminal(fallbackValue(string(attrs.State))),
			sanitizeTerminal(fallbackValue(attrs.PublishStartDate)),
			sanitizeTerminal(fallbackValue(attrs.PublishEndDate)),
		)
	}
	return w.Flush()
}

func printNominationsMarkdown(resp *NominationsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Name | Type | State | Publish Start | Publish End |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		attrs := item.Attributes
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(fallbackValue(attrs.Name)),
			escapeMarkdown(fallbackValue(string(attrs.Type))),
			escapeMarkdown(fallbackValue(string(attrs.State))),
			escapeMarkdown(fallbackValue(attrs.PublishStartDate)),
			escapeMarkdown(fallbackValue(attrs.PublishEndDate)),
		)
	}
	return nil
}

func printNominationDeleteResultTable(result *NominationDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printNominationDeleteResultMarkdown(result *NominationDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}
