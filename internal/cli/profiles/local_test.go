package profiles

import (
	"context"
	"errors"
	"flag"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveProfilesInstallDirVersionBoundary(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	homeDir := t.TempDir()
	profilesUserHomeDirFn = func() (string, error) { return homeDir, nil }

	tests := []struct {
		major    int
		relative string
	}{
		{major: 15, relative: filepath.Join("Library", "MobileDevice", "Provisioning Profiles")},
		{major: 16, relative: filepath.Join("Library", "Developer", "Xcode", "UserData", "Provisioning Profiles")},
		{major: 27, relative: filepath.Join("Library", "Developer", "Xcode", "UserData", "Provisioning Profiles")},
	}

	for _, test := range tests {
		activeXcodeMajorVersionFn = func(context.Context) (int, error) { return test.major, nil }
		got, err := resolveProfilesInstallDir(context.Background(), "")
		if err != nil {
			t.Fatalf("resolveProfilesInstallDir() for Xcode %d error = %v", test.major, err)
		}
		want := filepath.Join(homeDir, test.relative)
		if got != want {
			t.Fatalf("resolveProfilesInstallDir() for Xcode %d = %q, want %q", test.major, got, want)
		}
	}
}

func TestResolveProfilesInstallDirExplicitOverrideSkipsDiscovery(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "linux"
	profilesUserHomeDirFn = func() (string, error) { panic("home lookup must not run") }
	activeXcodeMajorVersionFn = func(context.Context) (int, error) { panic("Xcode discovery must not run") }

	input := filepath.Join(t.TempDir(), "nested", "..", "profiles")
	got, err := resolveProfilesInstallDir(context.Background(), input)
	if err != nil {
		t.Fatalf("resolveProfilesInstallDir() error = %v", err)
	}
	if want := filepath.Clean(input); got != want {
		t.Fatalf("resolveProfilesInstallDir() = %q, want %q", got, want)
	}
}

func TestResolveProfilesInstallDirFailsClosed(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	discoveryErr := errors.New("full Xcode is not active")
	activeXcodeMajorVersionFn = func(context.Context) (int, error) { return 0, discoveryErr }

	_, err := resolveProfilesInstallDir(context.Background(), "")
	if !errors.Is(err, discoveryErr) {
		t.Fatalf("resolveProfilesInstallDir() error = %v, want discovery error", err)
	}
	if errors.Is(err, errProfilesInstallDirRequired) {
		t.Fatalf("active Xcode discovery error must not be classified as missing --install-dir: %v", err)
	}
}

func TestResolveProfilesInstallDirPreservesCancellation(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	activeXcodeMajorVersionFn = func(ctx context.Context) (int, error) { return 0, ctx.Err() }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolveProfilesInstallDir(ctx, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolveProfilesInstallDir() error = %v, want context.Canceled", err)
	}
}

func TestResolveProfilesInstallDirHomeFailureIsRuntimeError(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	activeXcodeMajorVersionFn = func(context.Context) (int, error) { return 16, nil }
	homeErr := errors.New("home unavailable")
	profilesUserHomeDirFn = func() (string, error) { return "", homeErr }

	_, err := resolveProfilesInstallDirForCommand(context.Background(), "", "profiles local list")
	if !errors.Is(err, homeErr) {
		t.Fatalf("resolveProfilesInstallDirForCommand() error = %v, want home error", err)
	}
	if errors.Is(err, flag.ErrHelp) {
		t.Fatalf("home failure must be a runtime error, got %v", err)
	}
	if !strings.HasPrefix(err.Error(), "profiles local list: resolve install directory:") {
		t.Fatalf("unexpected command context: %v", err)
	}
}

func TestResolveProfilesInstallDirNonMacRequiresOverride(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	t.Cleanup(func() { profilesRuntimeGOOS = originalGOOS })
	profilesRuntimeGOOS = "linux"

	_, err := resolveProfilesInstallDir(context.Background(), "")
	if !errors.Is(err, errProfilesInstallDirRequired) {
		t.Fatalf("resolveProfilesInstallDir() error = %v, want --install-dir requirement", err)
	}
}

func TestProfilesLocalHelpDocumentsVersionedDefaults(t *testing.T) {
	commands := []struct {
		name string
		help string
	}{
		{name: "local", help: ProfilesLocalCommand().LongHelp},
		{name: "install", help: ProfilesLocalInstallCommand().LongHelp},
		{name: "list", help: ProfilesLocalListCommand().LongHelp},
		{name: "clean", help: ProfilesLocalCleanCommand().LongHelp},
	}

	for _, command := range commands {
		for _, want := range []string{
			"Xcode 16 or newer: ~/Library/Developer/Xcode/UserData/Provisioning Profiles",
			"Xcode 15 or older: ~/Library/MobileDevice/Provisioning Profiles",
			"Use --install-dir",
		} {
			if !strings.Contains(command.help, want) {
				t.Fatalf("%s help missing %q: %s", command.name, want, command.help)
			}
		}
	}
}

func TestIsExpired(t *testing.T) {
	t0 := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		now       time.Time
		want      bool
	}{
		{name: "zero", expiresAt: time.Time{}, now: t0, want: false},
		{name: "before", expiresAt: t0.Add(1 * time.Second), now: t0, want: false},
		{name: "equal", expiresAt: t0, now: t0, want: true},
		{name: "after", expiresAt: t0.Add(-1 * time.Second), now: t0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpired(tt.expiresAt, tt.now); got != tt.want {
				t.Fatalf("isExpired(expiresAt=%s, now=%s)=%t, want %t", tt.expiresAt.Format(time.RFC3339Nano), tt.now.Format(time.RFC3339Nano), got, tt.want)
			}
		})
	}
}
