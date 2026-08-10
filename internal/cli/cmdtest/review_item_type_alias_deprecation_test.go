package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

const deprecatedExperimentAliasWarning = "Warning: `--item-type appStoreVersionExperimentV2` is deprecated. " +
	"Use `--item-type appStoreVersionExperimentsV2`. The alias will be removed in 5.0.0."

// TestReviewItemsAddAcceptsDeprecatedExperimentItemTypeAlias keeps the 3.7.0
// singular alias working through the 4.0.0 deprecation window while warning on
// stderr and sending the canonical relationship payload.
func TestReviewItemsAddAcceptsDeprecatedExperimentItemTypeAlias(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "flat", args: []string{"review", "items-add"}},
		{name: "nested", args: []string{"review", "items", "add"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)

			var payload asc.ReviewSubmissionItemCreateRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost || req.URL.Path != "/v1/reviewSubmissionItems" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Errorf("decode payload: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"data":{"type":"reviewSubmissionItems","id":"item-1","attributes":{"state":"READY_FOR_REVIEW"}}}`)
			}))
			t.Cleanup(server.Close)
			installDefaultTransportForServer(t, server)

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			args := append(
				append([]string{}, test.args...),
				"--submission", "submission-1",
				"--item-type", "appStoreVersionExperimentV2",
				"--item-id", "experiment-1",
				"--output", "json",
			)

			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				runErr = root.Run(context.Background())
			})
			if runErr != nil {
				t.Fatalf("run: %v; stderr=%q", runErr, stderr)
			}

			if payload.Data.Relationships.AppStoreVersionExperimentV2 == nil {
				t.Fatalf("expected appStoreVersionExperimentV2 relationship, got %#v", payload.Data.Relationships)
			}
			if got := payload.Data.Relationships.AppStoreVersionExperimentV2.Data.ID; got != "experiment-1" {
				t.Fatalf("relationship id = %q, want %q", got, "experiment-1")
			}
			if got := payload.Data.Relationships.AppStoreVersionExperimentV2.Data.Type; got != asc.ResourceTypeAppStoreVersionExperiments {
				t.Fatalf("relationship type = %q, want %q", got, asc.ResourceTypeAppStoreVersionExperiments)
			}

			if !strings.Contains(stderr, deprecatedExperimentAliasWarning) {
				t.Fatalf("expected deprecation warning %q, got %q", deprecatedExperimentAliasWarning, stderr)
			}
			var response asc.ReviewSubmissionItemResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("parse stdout: %v; stdout=%q", err, stdout)
			}
			if response.Data.ID != "item-1" {
				t.Fatalf("response id = %q, want %q", response.Data.ID, "item-1")
			}
		})
	}
}

// TestReviewItemsAddCanonicalExperimentItemTypeDoesNotWarn keeps the
// deprecation warning scoped to the legacy singular spelling.
func TestReviewItemsAddCanonicalExperimentItemTypeDoesNotWarn(t *testing.T) {
	setupAuth(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"type":"reviewSubmissionItems","id":"item-1","attributes":{"state":"READY_FOR_REVIEW"}}}`)
	}))
	t.Cleanup(server.Close)
	installDefaultTransportForServer(t, server)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"review", "items-add",
			"--submission", "submission-1",
			"--item-type", "appStoreVersionExperimentsV2",
			"--item-id", "experiment-1",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run: %v; stderr=%q", runErr, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr for the canonical item type, got %q", stderr)
	}
}
