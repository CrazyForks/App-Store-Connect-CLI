package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printCustomerReviewResponseTable(resp *CustomerReviewResponseResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tState\tLast Modified\tResponse Body")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		resp.Data.ID,
		sanitizeTerminal(resp.Data.Attributes.State),
		sanitizeTerminal(resp.Data.Attributes.LastModified),
		compactWhitespace(resp.Data.Attributes.ResponseBody),
	)
	return w.Flush()
}

func printCustomerReviewResponseMarkdown(resp *CustomerReviewResponseResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | State | Last Modified | Response Body |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s |\n",
		escapeMarkdown(resp.Data.ID),
		escapeMarkdown(resp.Data.Attributes.State),
		escapeMarkdown(resp.Data.Attributes.LastModified),
		escapeMarkdown(resp.Data.Attributes.ResponseBody),
	)
	return nil
}

func printCustomerReviewResponseDeleteResultTable(result *CustomerReviewResponseDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n",
		result.ID,
		result.Deleted,
	)
	return w.Flush()
}

func printCustomerReviewResponseDeleteResultMarkdown(result *CustomerReviewResponseDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(result.ID),
		result.Deleted,
	)
	return nil
}
