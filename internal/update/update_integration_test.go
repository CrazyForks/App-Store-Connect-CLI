package update

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

const updateIntegrationEnvVar = "ASC_UPDATE_INTEGRATION"

func TestCheckAndUpdate_HomebrewSkipsDownload_Integration(t *testing.T) {
	raw := strings.TrimSpace(os.Getenv(updateIntegrationEnvVar))
	if raw == "" {
		t.Skipf("set %s=true to run integration update test", updateIntegrationEnvVar)
	}
	enabled, err := parseBool(raw)
	if err != nil || !enabled {
		t.Skipf("set %s=true to run integration update test", updateIntegrationEnvVar)
	}
	if envBool(noUpdateEnvVar) || envBool(skipUpdateEnvVar) {
		t.Skip("update checks are disabled via env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := CheckAndUpdate(ctx, Options{
		CurrentVersion:  "0.0.0",
		AutoUpdate:      true,
		DownloadBaseURL: "http://127.0.0.1:1",
		Output:          io.Discard,
		ExecutablePath:  "/opt/homebrew/bin/asc",
		EvalSymlinks: func(string) (string, error) {
			return "/opt/homebrew/Cellar/asc/1.0.0/bin/asc", nil
		},
	})
	if err != nil {
		t.Fatalf("CheckAndUpdate() error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
	if result.Updated {
		t.Fatal("expected homebrew install to skip auto-update")
	}
	if result.InstallMethod != InstallMethodHomebrew {
		t.Fatalf("expected install method homebrew, got %q", result.InstallMethod)
	}
}
