package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type processedMaximumOutput struct {
	LatestProcessedBuildNumber *string  `json:"latestProcessedBuildNumber"`
	LatestUploadBuildNumber    *string  `json:"latestUploadBuildNumber"`
	LatestObservedBuildNumber  *string  `json:"latestObservedBuildNumber"`
	NextBuildNumber            string   `json:"nextBuildNumber"`
	SourcesConsidered          []string `json:"sourcesConsidered"`
}

func runProcessedMaximumCommand(t *testing.T) (string, string, error) {
	t.Helper()

	root := RootCommand("test")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "next-build-number", "--app", "100000001", "--output", "json"}); err != nil {
			t.Fatalf("parse: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func decodeProcessedMaximumOutput(t *testing.T, stdout string) processedMaximumOutput {
	t.Helper()

	var output processedMaximumOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode output: %v; stdout=%s", err, stdout)
	}
	return output
}

func requireProcessedMaximumValue(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func TestBuildsNextBuildNumberUsesHighestProcessedNumber(t *testing.T) {
	tests := []struct {
		name              string
		chronologicalBody string
		maximumBody       string
		wantLatest        string
		wantObserved      string
		wantNext          string
	}{
		{
			name: "older numeric maximum",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-51","attributes":{"version":"51","uploadedDate":"2026-02-02T00:00:00Z"}},
				{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-01T00:00:00Z"}}
			]}`,
			wantLatest:   "50",
			wantObserved: "100",
			wantNext:     "101",
		},
		{
			name: "dotted numeric maximum",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-1-9","attributes":{"version":"1.9","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-1-10","attributes":{"version":"1.10","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`,
			wantLatest:   "1.9",
			wantObserved: "1.10",
			wantNext:     "1.11",
		},
		{
			name:              "chronological seed survives lower second read",
			chronologicalBody: `{"data":[{"type":"builds","id":"build-new-100","attributes":{"version":"100","uploadedDate":"2026-02-03T00:00:00Z"}}]}`,
			maximumBody:       `{"data":[{"type":"builds","id":"build-old-50","attributes":{"version":"50","uploadedDate":"2026-02-02T00:00:00Z"}}]}`,
			wantLatest:        "100",
			wantObserved:      "100",
			wantNext:          "101",
		},
		{
			name: "older non-positive placeholders",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-zero","attributes":{"version":"0","uploadedDate":"2026-02-02T00:00:00Z"}},
				{"type":"builds","id":"build-old-zero-dot","attributes":{"version":"0.1","uploadedDate":"2026-02-01T00:00:00Z"}}
			]}`,
			wantLatest:   "50",
			wantObserved: "50",
			wantNext:     "51",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			originalTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = originalTransport })
			buildRequests := 0
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v1/builds":
					buildRequests++
					query := req.URL.Query()
					if query.Get("filter[app]") != "100000001" || query.Get("sort") != "-uploadedDate" {
						t.Fatalf("unexpected builds query: %s", req.URL.RawQuery)
					}
					body := test.chronologicalBody
					if buildRequests > 1 && test.maximumBody != "" {
						body = test.maximumBody
					}
					return jsonHTTPResponse(http.StatusOK, body), nil
				case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/100000001/buildUploads":
					return jsonHTTPResponse(http.StatusOK, `{"data":[],"links":{"next":""}}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})

			stdout, stderr, runErr := runProcessedMaximumCommand(t)
			if runErr != nil {
				t.Fatalf("run: %v", runErr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			output := decodeProcessedMaximumOutput(t, stdout)
			requireProcessedMaximumValue(t, "latestProcessedBuildNumber", output.LatestProcessedBuildNumber, test.wantLatest)
			requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, test.wantObserved)
			if output.NextBuildNumber != test.wantNext {
				t.Fatalf("nextBuildNumber = %q, want %q", output.NextBuildNumber, test.wantNext)
			}
			if len(output.SourcesConsidered) != 1 || output.SourcesConsidered[0] != "processed_builds" {
				t.Fatalf("sourcesConsidered = %v, want [processed_builds]", output.SourcesConsidered)
			}
			if buildRequests != 2 {
				t.Fatalf("build requests = %d, want 2 chronological and maximum reads", buildRequests)
			}
		})
	}
}

func TestBuildsNextBuildNumberScansEveryProcessedPageForMaximum(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	const page2 = "https://api.appstoreconnect.apple.com/v1/builds?cursor=page-2"
	const page3 = "https://api.appstoreconnect.apple.com/v1/builds?cursor=page-3"
	page3Requests := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.String() == page2:
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"builds","id":"build-old-40","attributes":{"version":"40","uploadedDate":"2026-02-02T00:00:00Z"}}],"links":{"next":"`+page3+`"}}`), nil
		case req.URL.String() == page3:
			page3Requests++
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-01T00:00:00Z"}}],"links":{"next":""}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/builds":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}}],"links":{"next":"`+page2+`"}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/100000001/buildUploads":
			return jsonHTTPResponse(http.StatusOK, `{"data":[],"links":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, stderr, runErr := runProcessedMaximumCommand(t)
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	output := decodeProcessedMaximumOutput(t, stdout)
	requireProcessedMaximumValue(t, "latestProcessedBuildNumber", output.LatestProcessedBuildNumber, "50")
	requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, "100")
	if output.NextBuildNumber != "101" {
		t.Fatalf("nextBuildNumber = %q, want 101", output.NextBuildNumber)
	}
	if page3Requests != 1 {
		t.Fatalf("page 3 requests = %d, want 1", page3Requests)
	}
}

func TestBuildsNextBuildNumberRejectsMalformedOlderProcessedNumber(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	uploadRequests := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/builds":
			return jsonHTTPResponse(http.StatusOK, `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-bad","attributes":{"version":"not-a-number","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/100000001/buildUploads":
			uploadRequests++
			return jsonHTTPResponse(http.StatusOK, `{"data":[],"links":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	stdout, _, runErr := runProcessedMaximumCommand(t)
	if runErr == nil {
		t.Fatal("expected malformed processed build number error")
	}
	if !strings.Contains(runErr.Error(), `processed build build-old-bad build number "not-a-number" is not numeric`) {
		t.Fatalf("run error = %v, want malformed processed build context", runErr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if uploadRequests != 0 {
		t.Fatalf("upload requests = %d, want 0 after processed history failure", uploadRequests)
	}
}
