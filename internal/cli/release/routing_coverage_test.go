package release

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

func TestApplyRoutingCoverageStepReusesMatchingCompleteAsset(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(`{"type":"MultiPolygon","coordinates":[]}`), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}
	checksum, err := asc.ComputeFileChecksum(coveragePath, asc.ChecksumAlgorithmMD5)
	if err != nil {
		t.Fatalf("compute routing coverage checksum: %v", err)
	}

	originalTransport := http.DefaultTransport
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/VERSION_123/routingAppCoverage" {
			t.Fatalf("unexpected mutation while reusing routing coverage: %s %s", req.Method, req.URL.String())
		}
		return releaseJSONResponse(http.StatusOK, fmt.Sprintf(`{"data":{"type":"routingAppCoverages","id":"COVERAGE_123","attributes":{"fileName":"coverage.geojson","sourceFileChecksum":%q,"assetDeliveryState":{"state":"COMPLETE"}}}}`, checksum.Hash))
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	outcome, err := applyRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", coveragePath, false)
	if err != nil {
		t.Fatalf("applyRoutingCoverageStep() error: %v", err)
	}
	if outcome.Status != "skipped" || !outcome.Persist {
		t.Fatalf("expected persisted reuse outcome, got %#v", outcome)
	}
	details, ok := outcome.Details.(routingCoverageStepDetails)
	if !ok || details.Action != "reuse" || details.CoverageID != "COVERAGE_123" {
		t.Fatalf("unexpected reuse details: %#v", outcome.Details)
	}
}

func TestApplyRoutingCoverageStepDryRunPlansReplacementWithoutMutation(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(`{"type":"MultiPolygon","coordinates":[]}`), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}

	originalTransport := http.DefaultTransport
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/VERSION_123/routingAppCoverage" {
			t.Fatalf("unexpected mutation during routing coverage dry-run: %s %s", req.Method, req.URL.String())
		}
		return releaseJSONResponse(http.StatusOK, `{"data":{"type":"routingAppCoverages","id":"COVERAGE_OLD","attributes":{"sourceFileChecksum":"different","assetDeliveryState":{"state":"COMPLETE"}}}}`)
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	outcome, err := applyRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", coveragePath, true)
	if err != nil {
		t.Fatalf("applyRoutingCoverageStep() error: %v", err)
	}
	if outcome.Status != "dry-run" || outcome.Persist {
		t.Fatalf("expected non-persisted dry-run outcome, got %#v", outcome)
	}
	details, ok := outcome.Details.(routingCoverageStepDetails)
	if !ok || details.Action != "replace" {
		t.Fatalf("unexpected dry-run details: %#v", outcome.Details)
	}
}

func TestApplyRoutingCoverageStepTreatsNullRelationshipAsMissing(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(`{"type":"MultiPolygon","coordinates":[]}`), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}

	originalTransport := http.DefaultTransport
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/VERSION_123/routingAppCoverage" {
			t.Fatalf("unexpected mutation during routing coverage dry-run: %s %s", req.Method, req.URL.String())
		}
		return releaseJSONResponse(http.StatusOK, `{"data":null}`)
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	outcome, err := applyRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", coveragePath, true)
	if err != nil {
		t.Fatalf("applyRoutingCoverageStep() error: %v", err)
	}
	details, ok := outcome.Details.(routingCoverageStepDetails)
	if !ok || details.Action != "create" || details.CoverageID != "" {
		t.Fatalf("unexpected null-relationship plan: %#v", outcome.Details)
	}
}

func TestVerifyResumedCheckpointRechecksRoutingCoverageInput(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123"}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	opts := checkpointBindingOptions()
	opts.RoutingCoverageFile = "/tmp/coverage.geojson"
	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:        true,
			stepApplyMetadata:        true,
			stepApplyRoutingCoverage: true,
			stepAttachBuild:          true,
			stepValidateReadiness:    true,
		},
	}
	messages := []string{}
	if err := verifyResumedCheckpointBinding(context.Background(), client, opts, &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding() error: %v", err)
	}
	if checkpoint.Completed[stepApplyRoutingCoverage] || checkpoint.Completed[stepValidateReadiness] {
		t.Fatalf("expected routing coverage and readiness to be rechecked, got %#v", checkpoint.Completed)
	}
	if !checkpoint.Completed[stepEnsureVersion] || !checkpoint.Completed[stepAttachBuild] {
		t.Fatalf("expected remotely verified steps to survive, got %#v", checkpoint.Completed)
	}
	if !strings.Contains(strings.Join(messages, "\n"), stepApplyRoutingCoverage) {
		t.Fatalf("expected routing coverage recheck diagnostic, got %v", messages)
	}
}
