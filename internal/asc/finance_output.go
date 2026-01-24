package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// FinanceReportResult represents CLI output for finance report downloads.
type FinanceReportResult struct {
	VendorNumber      string `json:"vendorNumber"`
	ReportType        string `json:"reportType"`
	RegionCode        string `json:"regionCode"`
	ReportDate        string `json:"reportDate"`
	FilePath          string `json:"file"`
	Bytes             int64  `json:"bytes"`
	Decompressed      bool   `json:"decompressed"`
	DecompressedPath  string `json:"decompressedFile,omitempty"`
	DecompressedBytes int64  `json:"decompressedBytes,omitempty"`
}

func printFinanceReportResultTable(result *FinanceReportResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Vendor\tType\tRegion\tDate\tCompressed File\tCompressed Size\tDecompressed File\tDecompressed Size")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%d\n",
		result.VendorNumber,
		result.ReportType,
		result.RegionCode,
		result.ReportDate,
		result.FilePath,
		result.Bytes,
		result.DecompressedPath,
		result.DecompressedBytes,
	)
	return w.Flush()
}

func printFinanceReportResultMarkdown(result *FinanceReportResult) error {
	fmt.Fprintln(os.Stdout, "| Vendor | Type | Region | Date | Compressed File | Compressed Size | Decompressed File | Decompressed Size |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %d | %s | %d |\n",
		escapeMarkdown(result.VendorNumber),
		escapeMarkdown(result.ReportType),
		escapeMarkdown(result.RegionCode),
		escapeMarkdown(result.ReportDate),
		escapeMarkdown(result.FilePath),
		result.Bytes,
		escapeMarkdown(result.DecompressedPath),
		result.DecompressedBytes,
	)
	return nil
}
