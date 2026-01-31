package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func betaLicenseAgreementAppID(resource BetaLicenseAgreementResource) string {
	if resource.Relationships == nil || resource.Relationships.App == nil {
		return ""
	}
	return resource.Relationships.App.Data.ID
}

func printBetaLicenseAgreementsTable(resp *BetaLicenseAgreementsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tApp ID\tAgreement Text")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			item.ID,
			betaLicenseAgreementAppID(item),
			compactWhitespace(item.Attributes.AgreementText),
		)
	}
	return w.Flush()
}

func printBetaLicenseAgreementsMarkdown(resp *BetaLicenseAgreementsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | App ID | Agreement Text |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(betaLicenseAgreementAppID(item)),
			escapeMarkdown(item.Attributes.AgreementText),
		)
	}
	return nil
}
