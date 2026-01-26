package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printOfferCodesTable(resp *SubscriptionOfferCodeOneTimeUseCodesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCodes\tExpires\tCreated\tActive")
	for _, item := range resp.Data {
		attrs := item.Attributes
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%t\n",
			sanitizeTerminal(item.ID),
			attrs.NumberOfCodes,
			sanitizeTerminal(attrs.ExpirationDate),
			sanitizeTerminal(attrs.CreatedDate),
			attrs.Active,
		)
	}
	return w.Flush()
}

func printOfferCodesMarkdown(resp *SubscriptionOfferCodeOneTimeUseCodesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Codes | Expires | Created | Active |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		attrs := item.Attributes
		fmt.Fprintf(os.Stdout, "| %s | %d | %s | %s | %t |\n",
			escapeMarkdown(item.ID),
			attrs.NumberOfCodes,
			escapeMarkdown(attrs.ExpirationDate),
			escapeMarkdown(attrs.CreatedDate),
			attrs.Active,
		)
	}
	return nil
}
