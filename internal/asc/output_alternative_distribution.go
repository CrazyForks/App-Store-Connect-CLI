package asc

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func printAlternativeDistributionDomainsTable(resp *AlternativeDistributionDomainsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDomain\tReference Name\tCreated Date")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.Domain),
			compactWhitespace(item.Attributes.ReferenceName),
			item.Attributes.CreatedDate,
		)
	}
	return w.Flush()
}

func printAlternativeDistributionDomainsMarkdown(resp *AlternativeDistributionDomainsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Domain | Reference Name | Created Date |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Domain),
			escapeMarkdown(item.Attributes.ReferenceName),
			escapeMarkdown(item.Attributes.CreatedDate),
		)
	}
	return nil
}

func printAlternativeDistributionKeysTable(resp *AlternativeDistributionKeysResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPublic Key")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.PublicKey),
		)
	}
	return w.Flush()
}

func printAlternativeDistributionKeysMarkdown(resp *AlternativeDistributionKeysResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Public Key |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.PublicKey),
		)
	}
	return nil
}

func printAlternativeDistributionPackageVersionsTable(resp *AlternativeDistributionPackageVersionsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVersion\tState\tFile Checksum\tURL\tURL Expiration Date")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.Version),
			compactWhitespace(string(item.Attributes.State)),
			compactWhitespace(item.Attributes.FileChecksum),
			compactWhitespace(item.Attributes.URL),
			compactWhitespace(item.Attributes.URLExpirationDate),
		)
	}
	return w.Flush()
}

func printAlternativeDistributionPackageVersionsMarkdown(resp *AlternativeDistributionPackageVersionsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Version | State | File Checksum | URL | URL Expiration Date |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Version),
			escapeMarkdown(string(item.Attributes.State)),
			escapeMarkdown(item.Attributes.FileChecksum),
			escapeMarkdown(item.Attributes.URL),
			escapeMarkdown(item.Attributes.URLExpirationDate),
		)
	}
	return nil
}

func printAlternativeDistributionPackageVariantsTable(resp *AlternativeDistributionPackageVariantsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tURL Expiration Date\tKey Blob\tFile Checksum")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.URL),
			compactWhitespace(item.Attributes.URLExpirationDate),
			compactWhitespace(item.Attributes.AlternativeDistributionKeyBlob),
			compactWhitespace(item.Attributes.FileChecksum),
		)
	}
	return w.Flush()
}

func printAlternativeDistributionPackageVariantsMarkdown(resp *AlternativeDistributionPackageVariantsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | URL | URL Expiration Date | Key Blob | File Checksum |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.URL),
			escapeMarkdown(item.Attributes.URLExpirationDate),
			escapeMarkdown(item.Attributes.AlternativeDistributionKeyBlob),
			escapeMarkdown(item.Attributes.FileChecksum),
		)
	}
	return nil
}

func printAlternativeDistributionPackageDeltasTable(resp *AlternativeDistributionPackageDeltasResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tURL Expiration Date\tKey Blob\tFile Checksum")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.URL),
			compactWhitespace(item.Attributes.URLExpirationDate),
			compactWhitespace(item.Attributes.AlternativeDistributionKeyBlob),
			compactWhitespace(item.Attributes.FileChecksum),
		)
	}
	return w.Flush()
}

func printAlternativeDistributionPackageDeltasMarkdown(resp *AlternativeDistributionPackageDeltasResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | URL | URL Expiration Date | Key Blob | File Checksum |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.URL),
			escapeMarkdown(item.Attributes.URLExpirationDate),
			escapeMarkdown(item.Attributes.AlternativeDistributionKeyBlob),
			escapeMarkdown(item.Attributes.FileChecksum),
		)
	}
	return nil
}

func printAlternativeDistributionPackageTable(resp *AlternativeDistributionPackageResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSource File Checksum")
	fmt.Fprintf(w, "%s\t%s\n",
		resp.Data.ID,
		compactWhitespace(formatAlternativeDistributionChecksums(resp.Data.Attributes.SourceFileChecksum)),
	)
	return w.Flush()
}

func printAlternativeDistributionPackageMarkdown(resp *AlternativeDistributionPackageResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Source File Checksum |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s |\n",
		escapeMarkdown(resp.Data.ID),
		escapeMarkdown(formatAlternativeDistributionChecksums(resp.Data.Attributes.SourceFileChecksum)),
	)
	return nil
}

func formatAlternativeDistributionChecksums(checksums *Checksums) string {
	if checksums == nil {
		return ""
	}
	parts := []string{}
	if checksums.File != nil {
		parts = append(parts, formatAlternativeDistributionChecksum("file", checksums.File))
	}
	if checksums.Composite != nil {
		parts = append(parts, formatAlternativeDistributionChecksum("composite", checksums.Composite))
	}
	return strings.Join(parts, ", ")
}

func formatAlternativeDistributionChecksum(label string, checksum *Checksum) string {
	if checksum == nil {
		return ""
	}
	if checksum.Algorithm != "" {
		return fmt.Sprintf("%s:%s (%s)", label, checksum.Hash, checksum.Algorithm)
	}
	return fmt.Sprintf("%s:%s", label, checksum.Hash)
}

func printAlternativeDistributionDomainDeleteResultTable(result *AlternativeDistributionDomainDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printAlternativeDistributionDomainDeleteResultMarkdown(result *AlternativeDistributionDomainDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}

func printAlternativeDistributionKeyDeleteResultTable(result *AlternativeDistributionKeyDeleteResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDeleted")
	fmt.Fprintf(w, "%s\t%t\n", result.ID, result.Deleted)
	return w.Flush()
}

func printAlternativeDistributionKeyDeleteResultMarkdown(result *AlternativeDistributionKeyDeleteResult) error {
	fmt.Fprintln(os.Stdout, "| ID | Deleted |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n", escapeMarkdown(result.ID), result.Deleted)
	return nil
}
