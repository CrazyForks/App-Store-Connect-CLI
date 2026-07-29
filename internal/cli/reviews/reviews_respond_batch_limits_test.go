package reviews

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReviewBatchTargetsAcceptsFileAtSizeLimit(t *testing.T) {
	path := writeSizedReviewBatchFile(t, reviewBatchMaxFileBytes)

	targets, err := loadReviewBatchTargets(path)
	if err != nil {
		t.Fatalf("loadReviewBatchTargets() error: %v", err)
	}
	if len(targets) != 1 || targets[0].ReviewID != "review-1" {
		t.Fatalf("unexpected targets at size limit: %+v", targets)
	}
}

func TestLoadReviewBatchTargetsRejectsFileOneByteOverSizeLimit(t *testing.T) {
	path := writeSizedReviewBatchFile(t, reviewBatchMaxFileBytes+1)

	_, err := loadReviewBatchTargets(path)
	if err == nil {
		t.Fatal("expected oversized --file rejection, got nil")
	}
	want := fmt.Sprintf("--file must not exceed %d bytes", reviewBatchMaxFileBytes)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestLoadReviewBatchTargetsAcceptsMaximumReviewIDs(t *testing.T) {
	path := writeReviewBatchFileWithReviewIDs(t, reviewBatchMaxTargets)

	targets, err := loadReviewBatchTargets(path)
	if err != nil {
		t.Fatalf("loadReviewBatchTargets() error: %v", err)
	}
	if len(targets) != reviewBatchMaxTargets {
		t.Fatalf("expected %d targets, got %d", reviewBatchMaxTargets, len(targets))
	}
}

func TestLoadReviewBatchTargetsRejectsOneReviewIDOverLimit(t *testing.T) {
	path := writeReviewBatchFileWithReviewIDs(t, reviewBatchMaxTargets+1)

	_, err := loadReviewBatchTargets(path)
	if err == nil {
		t.Fatal("expected too-many-review-ids rejection, got nil")
	}
	want := fmt.Sprintf("--file must not contain more than %d review ids", reviewBatchMaxTargets)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

// writeSizedReviewBatchFile writes a valid single-target batch file whose byte
// length is exactly totalSize, padding the response body to reach it.
func writeSizedReviewBatchFile(t *testing.T, totalSize int) string {
	t.Helper()

	envelope := func(response string) []byte {
		body, err := json.Marshal(reviewBatchInput{
			Replies: []reviewBatchReplyInput{{Response: response, ReviewIDs: []string{"review-1"}}},
		})
		if err != nil {
			t.Fatalf("marshal batch file: %v", err)
		}
		return body
	}

	overhead := len(envelope(""))
	if overhead > totalSize {
		t.Fatalf("cannot build a batch file as small as %d bytes", totalSize)
	}
	body := envelope(strings.Repeat("a", totalSize-overhead))
	if len(body) != totalSize {
		t.Fatalf("expected a %d byte batch file, got %d", totalSize, len(body))
	}
	return writeReviewBatchTestFile(t, body)
}

func writeReviewBatchFileWithReviewIDs(t *testing.T, count int) string {
	t.Helper()

	reviewIDs := make([]string, 0, count)
	for i := range count {
		reviewIDs = append(reviewIDs, fmt.Sprintf("review-%d", i))
	}
	body, err := json.Marshal(reviewBatchInput{
		Replies: []reviewBatchReplyInput{{Response: "Thanks for the feedback.", ReviewIDs: reviewIDs}},
	})
	if err != nil {
		t.Fatalf("marshal batch file: %v", err)
	}
	return writeReviewBatchTestFile(t, body)
}

func writeReviewBatchTestFile(t *testing.T, body []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "replies.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write batch file: %v", err)
	}
	return path
}
