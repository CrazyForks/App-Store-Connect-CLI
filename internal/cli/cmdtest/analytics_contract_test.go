package cmdtest

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyticsRequestsUsesAccessTypeFilter(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	requestCount := 0
	installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/analyticsReportRequests" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		if got := req.URL.Query().Get("filter[accessType]"); got != "ONGOING" {
			t.Fatalf("filter[accessType] = %q, want ONGOING", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"type":"analyticsReportRequests","id":"request-1","attributes":{"accessType":"ONGOING","stoppedDueToInactivity":false}}],"links":{"self":"https://api.appstoreconnect.apple.com/v1/apps/app-1/analyticsReportRequests"}}`)),
			Request:    req,
		}, nil
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"analytics", "requests",
			"--app", "app-1",
			"--access-type", "ongoing",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	var response struct {
		Data []struct {
			Attributes struct {
				AccessType string `json:"accessType"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.AccessType != "ONGOING" {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestAnalyticsSalesSupportsNewTypeWithoutDailyDate(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	var payload bytes.Buffer
	writer := gzip.NewWriter(&payload)
	if _, err := writer.Write([]byte("report-data")); err != nil {
		t.Fatalf("write gzip: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	requestCount := 0
	installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodGet || req.URL.Path != "/v1/salesReports" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		query := req.URL.Query()
		if got := query.Get("filter[reportType]"); got != "WIN_BACK_ELIGIBILITY" {
			t.Fatalf("filter[reportType] = %q, want WIN_BACK_ELIGIBILITY", got)
		}
		if got := query.Get("filter[reportSubType]"); got != "SUMMARY" {
			t.Fatalf("filter[reportSubType] = %q, want SUMMARY", got)
		}
		if query.Has("filter[reportDate]") {
			t.Fatalf("unexpected filter[reportDate]: %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/a-gzip"}},
			Body:       io.NopCloser(bytes.NewReader(payload.Bytes())),
			Request:    req,
		}, nil
	}))

	outputPath := filepath.Join(t.TempDir(), "report.tsv.gz")
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"analytics", "sales",
			"--vendor", "12345678",
			"--type", "WIN_BACK_ELIGIBILITY",
			"--subtype", "SUMMARY",
			"--frequency", "DAILY",
			"--output", outputPath,
			"--output-format", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !bytes.Equal(data, payload.Bytes()) {
		t.Fatal("downloaded report does not match response payload")
	}
	var result struct {
		ReportDate string `json:"reportDate"`
		FilePath   string `json:"filePath"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if result.ReportDate != "" || result.FilePath != outputPath {
		t.Fatalf("unexpected output: %s", stdout)
	}
}
