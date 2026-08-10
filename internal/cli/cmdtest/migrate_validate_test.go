package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/migrate"
)

func TestMigrateValidateReportsUnsupportedCreateLocale(t *testing.T) {
	root := t.TempDir()
	for locale, description := range map[string]string{
		"nl":    "Dutch description",
		"en-US": "English description",
	} {
		localeDir := filepath.Join(root, "metadata", locale)
		if err := os.MkdirAll(localeDir, 0o755); err != nil {
			t.Fatalf("mkdir metadata %s: %v", locale, err)
		}
		writeFile(t, filepath.Join(localeDir, "description.txt"), description)
	}

	result := runMigrateValidate(t, root)

	if result.Valid {
		t.Fatalf("valid = true, want false for a locale migrate import would reject")
	}
	if result.ErrorCount != 1 {
		t.Fatalf("errorCount = %d, want 1", result.ErrorCount)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Locale != "nl" || issue.Field != "locale" {
			continue
		}
		if issue.Severity != "error" {
			t.Fatalf("locale issue severity = %q, want error", issue.Severity)
		}
		if issue.Message != `unsupported locale "nl"; did you mean: nl-NL` {
			t.Fatalf("locale issue message = %q, want the import rejection message", issue.Message)
		}
		found = true
	}
	if !found {
		t.Fatalf("issues = %+v, want a locale issue for \"nl\"", result.Issues)
	}
}

func TestMigrateValidateAcceptsSupportedLocales(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, "metadata", "nl-NL")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}
	writeFile(t, filepath.Join(localeDir, "description.txt"), "Dutch description")

	result := runMigrateValidate(t, root)

	if !result.Valid || result.ErrorCount != 0 {
		t.Fatalf("result = %+v, want a clean validation", result)
	}
}

func runMigrateValidate(t *testing.T, fastlaneDir string) migrate.MigrateValidateResult {
	t.Helper()

	rootCmd := RootCommand("1.2.3")
	rootCmd.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := rootCmd.Parse([]string{
			"migrate", "validate",
			"--fastlane-dir", fastlaneDir,
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := rootCmd.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result migrate.MigrateValidateResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return result
}
