package metadata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMetadataApproveWritesToRequestedReviewDir(t *testing.T) {
	reviewDir := filepath.Join(t.TempDir(), "relative-review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	plan := metadataReviewTestPlan(t, reviewDir)
	plan.PlanPath = filepath.Join("relative-review", metadataPlanFileName)
	plan.ApprovalPath = filepath.Join("relative-review", metadataApprovalFileName)
	if err := writeMetadataReviewJSON(filepath.Join(reviewDir, metadataPlanFileName), plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	otherCWD := t.TempDir()
	if err := os.Mkdir(filepath.Join(otherCWD, "relative-review"), 0o755); err != nil {
		t.Fatalf("mkdir stale relative review dir: %v", err)
	}
	t.Chdir(otherCWD)

	approval, err := ExecuteMetadataApprove(MetadataApproveOptions{
		ReviewDir: reviewDir,
		All:       true,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approval.ApprovalPath != filepath.Join(reviewDir, metadataApprovalFileName) {
		t.Fatalf("expected approval path under requested review dir, got %q", approval.ApprovalPath)
	}
	if _, err := os.Stat(filepath.Join(reviewDir, metadataApprovalFileName)); err != nil {
		t.Fatalf("expected approval in requested review dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(otherCWD, "relative-review", metadataApprovalFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected no approval in stale cwd-relative dir, err=%v", err)
	}
}

func TestReadMetadataPlanArtifactRejectsMismatchedHash(t *testing.T) {
	reviewDir := t.TempDir()
	plan := metadataReviewTestPlan(t, reviewDir)
	plan.Plan.Updates[0].To = "Edited after review"
	planPath := filepath.Join(reviewDir, metadataPlanFileName)
	if err := writeMetadataReviewJSON(planPath, plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	_, err := readMetadataPlanArtifact(planPath)
	if err == nil {
		t.Fatal("expected mismatched hash error")
	}
	if !strings.Contains(err.Error(), "planHash does not match") {
		t.Fatalf("expected planHash mismatch error, got %v", err)
	}
}

func metadataReviewTestPlan(t *testing.T, reviewDir string) MetadataPlanArtifact {
	t.Helper()
	result := PushPlanResult{
		AppID:     "app-1",
		AppInfoID: "appinfo-1",
		Version:   "1.2.3",
		VersionID: "version-1",
		Dir:       "./metadata",
		DryRun:    true,
		Includes:  []string{includeLocalizations},
		Updates: []PlanItem{{
			Key:    "app-info:en-US:subtitle",
			Scope:  appInfoDirName,
			Locale: "en-US",
			Field:  "subtitle",
			Reason: "field differs",
			From:   "Sleep tracker",
			To:     "Sleep scores",
		}},
	}
	options := metadataPlanOptionsFromPush(PushExecutionOptions{
		AppID:        result.AppID,
		Version:      result.Version,
		Dir:          result.Dir,
		Include:      includeLocalizations,
		DryRun:       true,
		AllowDeletes: false,
	}, result)
	planHash, err := hashMetadataPlan(options, result)
	if err != nil {
		t.Fatalf("hash plan: %v", err)
	}
	return MetadataPlanArtifact{
		SchemaVersion: metadataReviewSchemaV1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Command:       "metadata plan",
		ReviewDir:     reviewDir,
		PlanPath:      filepath.Join(reviewDir, metadataPlanFileName),
		ApprovalPath:  filepath.Join(reviewDir, metadataApprovalFileName),
		PlanHash:      planHash,
		Options:       options,
		Plan:          result,
	}
}
