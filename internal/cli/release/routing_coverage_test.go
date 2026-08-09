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
	routingcoveragecli "github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/routingcoverage"
)

const validReleaseRoutingCoverageGeoJSON = `{"type":"MultiPolygon","coordinates":[[[[77.5,12.9],[77.7,12.9],[77.7,13.1],[77.5,12.9]]]]}`

func prepareReleaseRoutingCoverage(t *testing.T, path string) routingcoveragecli.PreparedRoutingCoverageFile {
	t.Helper()
	t.Chdir(filepath.Dir(path))
	prepared, err := routingcoveragecli.PrepareRoutingCoverageFile(path)
	if err != nil {
		t.Fatalf("PrepareRoutingCoverageFile() error: %v", err)
	}
	return prepared
}

func TestApplyRoutingCoverageStepReusesMatchingCompleteAsset(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
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

	outcome, err := applyPreparedRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", prepareReleaseRoutingCoverage(t, coveragePath), false)
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
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
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

	outcome, err := applyPreparedRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", prepareReleaseRoutingCoverage(t, coveragePath), true)
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
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
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

	outcome, err := applyPreparedRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", prepareReleaseRoutingCoverage(t, coveragePath), true)
	if err != nil {
		t.Fatalf("applyRoutingCoverageStep() error: %v", err)
	}
	details, ok := outcome.Details.(routingCoverageStepDetails)
	if !ok || details.Action != "create" || details.CoverageID != "" {
		t.Fatalf("unexpected null-relationship plan: %#v", outcome.Details)
	}
}

func TestApplyRoutingCoverageStepCleansFailedReservation(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}

	originalTransport := http.DefaultTransport
	deleted := false
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/routingAppCoverage":
			return releaseJSONResponse(http.StatusOK, `{"data":null}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/routingAppCoverages":
			return releaseJSONResponse(http.StatusCreated, `{"data":{"type":"routingAppCoverages","id":"COVERAGE_NEW","attributes":{"uploadOperations":[]}}}`)
		case req.Method == http.MethodDelete && req.URL.Path == "/v1/routingAppCoverages/COVERAGE_NEW":
			deleted = true
			return releaseJSONResponse(http.StatusNoContent, "")
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	_, err := applyPreparedRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", prepareReleaseRoutingCoverage(t, coveragePath), false)
	if err == nil || !strings.Contains(err.Error(), "no upload operations returned") {
		t.Fatalf("applyRoutingCoverageStep() error = %v, want missing upload operations", err)
	}
	if !deleted {
		t.Fatal("expected failed routing coverage reservation to be deleted")
	}
}

func TestApplyRoutingCoverageStepRevalidatesBeforeDeletingExistingCoverage(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}
	prepared := prepareReleaseRoutingCoverage(t, coveragePath)
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON+"\n"), 0o600); err != nil {
		t.Fatalf("change routing coverage fixture: %v", err)
	}

	originalTransport := http.DefaultTransport
	deleted := false
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/routingAppCoverage":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"routingAppCoverages","id":"COVERAGE_OLD","attributes":{"sourceFileChecksum":"old-checksum","assetDeliveryState":{"state":"COMPLETE"}}}}`)
		case req.Method == http.MethodDelete && req.URL.Path == "/v1/routingAppCoverages/COVERAGE_OLD":
			deleted = true
			return releaseJSONResponse(http.StatusNoContent, "")
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	_, err := applyPreparedRoutingCoverageStep(context.Background(), newReleaseTestClient(t), "VERSION_123", prepared, false)
	if err == nil || !strings.Contains(err.Error(), "file changed after validation") {
		t.Fatalf("applyPreparedRoutingCoverageStep() error = %v, want changed-file diagnostic", err)
	}
	if deleted {
		t.Fatal("existing routing coverage was deleted before the prepared file was revalidated")
	}
}

func TestUploadPreparedRoutingCoverageFileDoesNotDeleteAfterAmbiguousCommitResponse(t *testing.T) {
	coveragePath := filepath.Join(t.TempDir(), "coverage.geojson")
	if err := os.WriteFile(coveragePath, []byte(validReleaseRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write routing coverage fixture: %v", err)
	}
	prepared := prepareReleaseRoutingCoverage(t, coveragePath)

	originalTransport := http.DefaultTransport
	deleted := false
	http.DefaultTransport = releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v1/routingAppCoverages":
			return releaseJSONResponse(http.StatusCreated, fmt.Sprintf(`{"data":{"type":"routingAppCoverages","id":"COVERAGE_NEW","attributes":{"uploadOperations":[{"method":"PUT","url":"https://upload.example/coverage","length":%d,"offset":0}]}}}`, len(validReleaseRoutingCoverageGeoJSON)))
		case req.Method == http.MethodPut && req.URL.Host == "upload.example":
			return releaseJSONResponse(http.StatusOK, `{}`)
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/routingAppCoverages/COVERAGE_NEW":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"routingAppCoverages","id":"","attributes":{"assetDeliveryState":{"state":"UPLOAD_COMPLETE"}}}}`)
		case req.Method == http.MethodDelete && req.URL.Path == "/v1/routingAppCoverages/COVERAGE_NEW":
			deleted = true
			return releaseJSONResponse(http.StatusNoContent, "")
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	_, err := routingcoveragecli.UploadPreparedRoutingCoverageFile(context.Background(), newReleaseTestClient(t), "VERSION_123", prepared)
	if err == nil || !strings.Contains(err.Error(), "committed routing coverage response is missing an ID") {
		t.Fatalf("UploadPreparedRoutingCoverageFile() error = %v, want missing-ID diagnostic", err)
	}
	if deleted {
		t.Fatal("routing coverage was deleted after an ambiguous successful commit response")
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
