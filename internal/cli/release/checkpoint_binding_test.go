package release

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/metadata"
	validatecli "github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/validate"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/validation"
)

func newCheckpointBindingClient(t *testing.T, handler releaseRoundTripFunc) *asc.Client {
	t.Helper()

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	http.DefaultTransport = handler
	return newReleaseTestClient(t)
}

func checkpointBindingOptions() runOptions {
	return runOptions{
		AppID:           "APP_123",
		Version:         "2.4.0",
		BuildID:         "BUILD_123",
		Platform:        "IOS",
		Mode:            releaseModeRun,
		SubmitForReview: true,
	}
}

func TestVerifyResumedCheckpointBindingRejectsVersionOwnedByAnotherApp(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_B":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_B","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_B"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_B",
		Completed: map[string]bool{stepEnsureVersion: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject a version owned by another app")
	}
	if !strings.Contains(err.Error(), "belongs to app") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsVersionStringMismatch(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"9.9.9","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{stepEnsureVersion: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject a version string mismatch")
	}
	if !strings.Contains(err.Error(), "2.4.0") {
		t.Fatalf("expected error naming the requested version, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsCompletedStepsWithoutVersionBinding(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	checkpoint := runCheckpoint{
		Completed: map[string]bool{stepApplyMetadata: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject completed steps without a bound version")
	}
}

// TestVerifyResumedCheckpointBindingExplainsMissingSubmitForReviewFlag proves
// that resuming a checkpoint with a completed submit_review step but without
// --submit-for-review reports the flag mismatch instead of claiming the
// checkpoint records an unknown step.
func TestVerifyResumedCheckpointBindingExplainsMissingSubmitForReviewFlag(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	opts := checkpointBindingOptions()
	opts.SubmitForReview = false
	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed:    map[string]bool{stepSubmitReview: true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, opts, &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject the missing --submit-for-review flag")
	}
	if !strings.Contains(err.Error(), "--submit-for-review") {
		t.Fatalf("expected error naming --submit-for-review, got %v", err)
	}
	if strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected a flag mismatch error, not an unknown-step error, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingRejectsUnknownCompletedStep(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{"publish_everything": true},
	}
	err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil)
	if err == nil {
		t.Fatal("expected checkpoint verification to reject an unknown completed step")
	}
	if !strings.Contains(err.Error(), "publish_everything") {
		t.Fatalf("expected error naming the unknown step, got %v", err)
	}
}

func TestVerifyResumedCheckpointBindingDropsUnprovenAttachBuildCompletion(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_OTHER","attributes":{"version":"41"}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID: "VERSION_123",
		Completed: map[string]bool{stepEnsureVersion: true, stepAttachBuild: true},
	}
	var messages []string
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, func(message string) {
		messages = append(messages, message)
	}); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepAttachBuild] {
		t.Fatal("expected unproven attach_build completion to be discarded")
	}
	if !checkpoint.Completed[stepEnsureVersion] {
		t.Fatal("expected verified ensure_version completion to survive")
	}
	if len(messages) == 0 || !strings.Contains(strings.Join(messages, "\n"), stepAttachBuild) {
		t.Fatalf("expected a diagnostic naming attach_build, got %v", messages)
	}
}

func TestVerifyResumedCheckpointBindingDropsSubmissionNotBoundToVersion(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_OTHER":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_OTHER","attributes":{"state":"SUBMITTED","platform":"IOS"},"relationships":{"appStoreVersionForReview":{"data":{"type":"appStoreVersions","id":"VERSION_OTHER"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_OTHER",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepAttachBuild:   true,
			stepSubmitReview:  true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if checkpoint.Completed[stepSubmitReview] {
		t.Fatal("expected unproven submit_review completion to be discarded")
	}
	if checkpoint.SubmissionID != "" {
		t.Fatalf("expected unproven submission ID to be cleared, got %q", checkpoint.SubmissionID)
	}
}

func TestVerifyResumedCheckpointBindingKeepsProvenCheckpoint(t *testing.T) {
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_123","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_123"}}}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_123/build":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"builds","id":"BUILD_123","attributes":{"version":"42"}}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/reviewSubmissions/SUBMISSION_123":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"reviewSubmissions","id":"SUBMISSION_123","attributes":{"state":"SUBMITTED","platform":"IOS"},"relationships":{"appStoreVersionForReview":{"data":{"type":"appStoreVersions","id":"VERSION_123"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})

	checkpoint := runCheckpoint{
		VersionID:    "VERSION_123",
		SubmissionID: "SUBMISSION_123",
		Completed: map[string]bool{
			stepEnsureVersion:     true,
			stepApplyMetadata:     true,
			stepAttachBuild:       true,
			stepValidateReadiness: true,
			stepSubmitReview:      true,
		},
	}
	if err := verifyResumedCheckpointBinding(context.Background(), client, checkpointBindingOptions(), &checkpoint, nil); err != nil {
		t.Fatalf("verifyResumedCheckpointBinding error: %v", err)
	}
	if len(checkpoint.Completed) != 5 {
		t.Fatalf("expected all verified completions to survive, got %#v", checkpoint.Completed)
	}
	if checkpoint.SubmissionID != "SUBMISSION_123" {
		t.Fatalf("expected verified submission ID to survive, got %q", checkpoint.SubmissionID)
	}
}

// TestExecuteRun_RejectsForgedCheckpointVersionBeforeMutation proves a modified
// checkpoint cannot substitute VersionID and have the pipeline act on it.
func TestExecuteRun_RejectsForgedCheckpointVersionBeforeMutation(t *testing.T) {
	origClientFactory := releaseClientFactory
	origMetadataExecutor := metadataPushExecutor
	origReadinessBuilder := readinessReportBuilder
	t.Cleanup(func() {
		releaseClientFactory = origClientFactory
		metadataPushExecutor = origMetadataExecutor
		readinessReportBuilder = origReadinessBuilder
	})

	metadataPushExecutor = func(context.Context, metadata.PushExecutionOptions) (metadata.PushPlanResult, error) {
		t.Fatal("metadata executor must not run for an unverifiable checkpoint")
		return metadata.PushPlanResult{}, nil
	}
	readinessReportBuilder = func(context.Context, validatecli.ReadinessOptions) (validation.Report, error) {
		t.Fatal("readiness builder must not run for an unverifiable checkpoint")
		return validation.Report{}, nil
	}

	var mutations []string
	client := newCheckpointBindingClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			mutations = append(mutations, req.Method+" "+req.URL.Path)
		}
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/VERSION_FORGED":
			return releaseJSONResponse(http.StatusOK, `{"data":{"type":"appStoreVersions","id":"VERSION_FORGED","attributes":{"versionString":"2.4.0","platform":"IOS"},"relationships":{"app":{"data":{"type":"apps","id":"APP_OTHER"}}}}}`)
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
	})
	releaseClientFactory = func() (*asc.Client, error) { return client, nil }

	checkpointPath := filepath.Join(t.TempDir(), "release-checkpoint.json")
	if err := saveCheckpoint(checkpointPath, runCheckpoint{
		AppID:       "APP_123",
		Version:     "2.4.0",
		BuildID:     "BUILD_123",
		MetadataDir: "./metadata/version/2.4.0",
		Platform:    "IOS",
		Mode:        releaseModeRun,
		VersionID:   "VERSION_FORGED",
		Completed: map[string]bool{
			stepEnsureVersion: true,
			stepApplyMetadata: true,
			stepAttachBuild:   true,
		},
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	result, err := executeRun(context.Background(), runOptions{
		AppID:          "APP_123",
		Version:        "2.4.0",
		BuildID:        "BUILD_123",
		MetadataDir:    "./metadata/version/2.4.0",
		Platform:       "IOS",
		Timeout:        releaseRunTimeout,
		Confirm:        true,
		CheckpointFile: checkpointPath,
	})
	if err == nil {
		t.Fatal("expected executeRun to fail for a checkpoint version owned by another app")
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	if len(mutations) != 0 {
		t.Fatalf("expected no mutating requests, got %v", mutations)
	}
}
