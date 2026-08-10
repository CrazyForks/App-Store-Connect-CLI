package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProfilesLocalDefaultDiscoveryFailuresAreRuntimeErrors(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("active Xcode profile directory is macOS-specific")
	}

	binDir := t.TempDir()
	xcodebuildPath := filepath.Join(binDir, "xcodebuild")
	script := "#!/bin/sh\nprintf 'xcode-select: active developer directory is a command line tools instance\\n' >&2\nexit 1\n"
	if err := os.WriteFile(xcodebuildPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake xcodebuild: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))

	tests := []struct {
		name    string
		args    []string
		context string
	}{
		{name: "install", args: []string{"profiles", "local", "install", "--path", filepath.Join(t.TempDir(), "profile.mobileprovision"), "--output", "json"}, context: "profiles local install"},
		{name: "list", args: []string{"profiles", "local", "list", "--output", "json"}, context: "profiles local list"},
		{name: "clean", args: []string{"profiles", "local", "clean", "--expired", "--dry-run", "--output", "json"}, context: "profiles local clean"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetReportFlags(t)
			stdout, stderr := captureCommandOutput(t, func() {
				if code := Run(test.args, "test"); code != ExitError {
					t.Fatalf("Run() exit code = %d, want %d", code, ExitError)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			for _, want := range []string{
				"Error: " + test.context + ": resolve install directory:",
				"xcodebuild -version failed",
				"active developer directory is a command line tools instance",
			} {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr missing %q: %q", want, stderr)
				}
			}
			if strings.Contains(stderr, "Usage:") {
				t.Fatalf("runtime discovery error must not print usage: %q", stderr)
			}
		})
	}
}
