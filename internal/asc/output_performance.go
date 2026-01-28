package asc

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// PerformanceDownloadResult represents CLI output for performance downloads.
type PerformanceDownloadResult struct {
	DownloadType          string `json:"downloadType"`
	AppID                 string `json:"appId,omitempty"`
	BuildID               string `json:"buildId,omitempty"`
	DiagnosticSignatureID string `json:"diagnosticSignatureId,omitempty"`
	FilePath              string `json:"filePath"`
	FileSize              int64  `json:"fileSize"`
	Decompressed          bool   `json:"decompressed"`
	DecompressedPath      string `json:"decompressedPath,omitempty"`
	DecompressedSize      int64  `json:"decompressedSize,omitempty"`
}

type perfPowerMetricsSummary struct {
	Version         string
	ProductCount    int
	TrendingUpCount int
	RegressionCount int
}

type diagnosticLogsSummary struct {
	Version      string
	ProductCount int
	LogCount     int
	InsightCount int
}

func summarizePerfPowerMetrics(resp *PerfPowerMetricsResponse) (perfPowerMetricsSummary, error) {
	if resp == nil {
		return perfPowerMetricsSummary{}, fmt.Errorf("perf power metrics response is nil")
	}
	if len(resp.Data) == 0 {
		return perfPowerMetricsSummary{}, fmt.Errorf("perf power metrics response is empty")
	}

	var payload struct {
		Version  string `json:"version"`
		Insights struct {
			TrendingUp  []json.RawMessage `json:"trendingUp"`
			Regressions []json.RawMessage `json:"regressions"`
		} `json:"insights"`
		ProductData []json.RawMessage `json:"productData"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return perfPowerMetricsSummary{}, fmt.Errorf("decode perf power metrics: %w", err)
	}

	return perfPowerMetricsSummary{
		Version:         payload.Version,
		ProductCount:    len(payload.ProductData),
		TrendingUpCount: len(payload.Insights.TrendingUp),
		RegressionCount: len(payload.Insights.Regressions),
	}, nil
}

func summarizeDiagnosticLogs(resp *DiagnosticLogsResponse) (diagnosticLogsSummary, error) {
	if resp == nil {
		return diagnosticLogsSummary{}, fmt.Errorf("diagnostic logs response is nil")
	}
	if len(resp.Data) == 0 {
		return diagnosticLogsSummary{}, fmt.Errorf("diagnostic logs response is empty")
	}

	var payload struct {
		Version     string `json:"version"`
		ProductData []struct {
			DiagnosticLogs     []json.RawMessage `json:"diagnosticLogs"`
			DiagnosticInsights []json.RawMessage `json:"diagnosticInsights"`
		} `json:"productData"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return diagnosticLogsSummary{}, fmt.Errorf("decode diagnostic logs: %w", err)
	}

	logCount := 0
	insightCount := 0
	for _, item := range payload.ProductData {
		logCount += len(item.DiagnosticLogs)
		insightCount += len(item.DiagnosticInsights)
	}

	return diagnosticLogsSummary{
		Version:      payload.Version,
		ProductCount: len(payload.ProductData),
		LogCount:     logCount,
		InsightCount: insightCount,
	}, nil
}

func printPerfPowerMetricsTable(resp *PerfPowerMetricsResponse) error {
	summary, err := summarizePerfPowerMetrics(resp)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Version\tProducts\tTrending Up\tRegressions")
	fmt.Fprintf(w, "%s\t%d\t%d\t%d\n",
		summary.Version,
		summary.ProductCount,
		summary.TrendingUpCount,
		summary.RegressionCount,
	)
	return w.Flush()
}

func printPerfPowerMetricsMarkdown(resp *PerfPowerMetricsResponse) error {
	summary, err := summarizePerfPowerMetrics(resp)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "| Version | Products | Trending Up | Regressions |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %d | %d | %d |\n",
		escapeMarkdown(summary.Version),
		summary.ProductCount,
		summary.TrendingUpCount,
		summary.RegressionCount,
	)
	return nil
}

func printDiagnosticSignaturesTable(resp *DiagnosticSignaturesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tType\tWeight\tInsight\tSignature")
	for _, item := range resp.Data {
		insight := ""
		if item.Attributes.Insight != nil {
			insight = string(item.Attributes.Insight.Direction)
		}
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\t%s\n",
			item.ID,
			item.Attributes.DiagnosticType,
			item.Attributes.Weight,
			insight,
			item.Attributes.Signature,
		)
	}
	return w.Flush()
}

func printDiagnosticSignaturesMarkdown(resp *DiagnosticSignaturesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Type | Weight | Insight | Signature |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		insight := ""
		if item.Attributes.Insight != nil {
			insight = string(item.Attributes.Insight.Direction)
		}
		fmt.Fprintf(os.Stdout, "| %s | %s | %.2f | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(string(item.Attributes.DiagnosticType)),
			item.Attributes.Weight,
			escapeMarkdown(insight),
			escapeMarkdown(item.Attributes.Signature),
		)
	}
	return nil
}

func printDiagnosticLogsTable(resp *DiagnosticLogsResponse) error {
	summary, err := summarizeDiagnosticLogs(resp)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Version\tProducts\tLogs\tInsights")
	fmt.Fprintf(w, "%s\t%d\t%d\t%d\n",
		summary.Version,
		summary.ProductCount,
		summary.LogCount,
		summary.InsightCount,
	)
	return w.Flush()
}

func printDiagnosticLogsMarkdown(resp *DiagnosticLogsResponse) error {
	summary, err := summarizeDiagnosticLogs(resp)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "| Version | Products | Logs | Insights |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %d | %d | %d |\n",
		escapeMarkdown(summary.Version),
		summary.ProductCount,
		summary.LogCount,
		summary.InsightCount,
	)
	return nil
}

func printPerformanceDownloadResultTable(result *PerformanceDownloadResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Type\tApp ID\tBuild ID\tDiagnostic ID\tCompressed File\tCompressed Size\tDecompressed File\tDecompressed Size")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%d\n",
		result.DownloadType,
		result.AppID,
		result.BuildID,
		result.DiagnosticSignatureID,
		result.FilePath,
		result.FileSize,
		result.DecompressedPath,
		result.DecompressedSize,
	)
	return w.Flush()
}

func printPerformanceDownloadResultMarkdown(result *PerformanceDownloadResult) error {
	fmt.Fprintln(os.Stdout, "| Type | App ID | Build ID | Diagnostic ID | Compressed File | Compressed Size | Decompressed File | Decompressed Size |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %d | %s | %d |\n",
		escapeMarkdown(result.DownloadType),
		escapeMarkdown(result.AppID),
		escapeMarkdown(result.BuildID),
		escapeMarkdown(result.DiagnosticSignatureID),
		escapeMarkdown(result.FilePath),
		result.FileSize,
		escapeMarkdown(result.DecompressedPath),
		result.DecompressedSize,
	)
	return nil
}
