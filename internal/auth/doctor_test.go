package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/config"
)

func TestDoctorConfigPermissionsWarning(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write config error: %v", err)
	}
	t.Setenv("ASC_CONFIG_PATH", configPath)

	report := Doctor(DoctorOptions{})
	section := findDoctorSection(t, report, "Storage")
	if !sectionHasStatus(section, DoctorWarn, "Config file permissions") {
		t.Fatalf("expected config permissions warning, got %#v", section.Checks)
	}

	report = Doctor(DoctorOptions{Fix: true})
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config error: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected config permissions fixed to 0600, got %#o", info.Mode().Perm())
	}
}

func TestDoctorTempFilesWarns(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	tempFile, err := os.CreateTemp(os.TempDir(), "asc-key-*.p8")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	tempFile.Close()
	t.Cleanup(func() {
		_ = os.Remove(tempFile.Name())
	})

	report := Doctor(DoctorOptions{})
	section := findDoctorSection(t, report, "Temp Files")
	if !sectionHasStatus(section, DoctorWarn, "orphaned temp key file") {
		t.Fatalf("expected temp file warning, got %#v", section.Checks)
	}
}

func TestDoctorPrivateKeyPermissionsFix(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "AuthKey.p8")
	writeECDSAPEM(t, keyPath, 0o600, true)
	if err := os.Chmod(keyPath, 0o644); err != nil {
		t.Fatalf("chmod key error: %v", err)
	}

	cfg := &config.Config{
		DefaultKeyName: "test",
		Keys: []config.Credential{
			{
				Name:           "test",
				KeyID:          "KEY123",
				IssuerID:       "ISS456",
				PrivateKeyPath: keyPath,
			},
		},
	}
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.SaveAt(configPath, cfg); err != nil {
		t.Fatalf("save config error: %v", err)
	}
	t.Setenv("ASC_CONFIG_PATH", configPath)

	report := Doctor(DoctorOptions{Fix: true})
	section := findDoctorSection(t, report, "Private Keys")
	if !sectionHasStatus(section, DoctorOK, "permissions fixed to 0600") {
		t.Fatalf("expected private key permissions fix, got %#v", section.Checks)
	}
}

func findDoctorSection(t *testing.T, report DoctorReport, title string) DoctorSection {
	t.Helper()
	for _, section := range report.Sections {
		if section.Title == title {
			return section
		}
	}
	t.Fatalf("expected section %q, got %#v", title, report.Sections)
	return DoctorSection{}
}

func sectionHasStatus(section DoctorSection, status DoctorStatus, contains string) bool {
	for _, check := range section.Checks {
		if check.Status == status && strings.Contains(check.Message, contains) {
			return true
		}
	}
	return false
}
