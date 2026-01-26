package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/config"
)

func TestResolvePrivateKeyPathPrefersPath(t *testing.T) {
	resetPrivateKeyTemp(t)
	t.Setenv("ASC_PRIVATE_KEY_PATH", "/tmp/AuthKey.p8")
	t.Setenv("ASC_PRIVATE_KEY_B64", base64.StdEncoding.EncodeToString([]byte("ignored")))
	t.Setenv("ASC_PRIVATE_KEY", "ignored")

	path, err := resolvePrivateKeyPath()
	if err != nil {
		t.Fatalf("resolvePrivateKeyPath() error: %v", err)
	}
	if path != "/tmp/AuthKey.p8" {
		t.Fatalf("expected path /tmp/AuthKey.p8, got %q", path)
	}
}

func TestResolvePrivateKeyPathFromBase64(t *testing.T) {
	resetPrivateKeyTemp(t)
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY", "")

	encoded := base64.StdEncoding.EncodeToString([]byte("key-data"))
	t.Setenv("ASC_PRIVATE_KEY_B64", encoded)

	path, err := resolvePrivateKeyPath()
	if err != nil {
		t.Fatalf("resolvePrivateKeyPath() error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "key-data" {
		t.Fatalf("expected key data %q, got %q", "key-data", string(data))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected 0600 permissions, got %v", info.Mode().Perm())
	}
}

func TestResolvePrivateKeyPathFromRawValue(t *testing.T) {
	resetPrivateKeyTemp(t)
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "")

	t.Setenv("ASC_PRIVATE_KEY", "line1\\nline2")
	path, err := resolvePrivateKeyPath()
	if err != nil {
		t.Fatalf("resolvePrivateKeyPath() error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != "line1\nline2" {
		t.Fatalf("expected newline expansion, got %q", string(data))
	}
}

func TestResolvePrivateKeyPathInvalidBase64(t *testing.T) {
	resetPrivateKeyTemp(t)
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "not-base64")

	if _, err := resolvePrivateKeyPath(); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestGetASCClient_ProfileMissingSkipsEnvFallback(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	keyPath := filepath.Join(tempDir, "AuthKey.p8")
	writeECDSAPEM(t, keyPath)

	cfg := &config.Config{
		DefaultKeyName: "personal",
		Keys: []config.Credential{
			{
				Name:           "personal",
				KeyID:          "KEY123",
				IssuerID:       "ISS456",
				PrivateKeyPath: keyPath,
			},
		},
	}
	if err := config.SaveAt(configPath, cfg); err != nil {
		t.Fatalf("SaveAt() error: %v", err)
	}

	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_PROFILE", "missing")
	t.Setenv("ASC_KEY_ID", "ENVKEY")
	t.Setenv("ASC_ISSUER_ID", "ENVISS")
	t.Setenv("ASC_PRIVATE_KEY_PATH", keyPath)

	previousProfile := selectedProfile
	selectedProfile = ""
	t.Cleanup(func() {
		selectedProfile = previousProfile
	})

	_, err := getASCClient()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `credentials not found for profile "missing"`) {
		t.Fatalf("expected profile error, got %v", err)
	}
}

func TestGetASCClient_BypassKeychainPrefersEnvOverConfig(t *testing.T) {
	resetPrivateKeyTemp(t)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	envKeyPath := filepath.Join(tempDir, "AuthKey-Env.p8")
	writeECDSAPEM(t, envKeyPath)

	cfg := &config.Config{
		DefaultKeyName: "config",
		Keys: []config.Credential{
			{
				Name:           "config",
				KeyID:          "CFGKEY",
				IssuerID:       "CFGISS",
				PrivateKeyPath: filepath.Join(tempDir, "missing.p8"),
			},
		},
	}
	if err := config.SaveAt(configPath, cfg); err != nil {
		t.Fatalf("SaveAt() error: %v", err)
	}

	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_PROFILE", "")
	t.Setenv("ASC_KEY_ID", "ENVKEY")
	t.Setenv("ASC_ISSUER_ID", "ENVISS")
	t.Setenv("ASC_PRIVATE_KEY_PATH", envKeyPath)

	previousProfile := selectedProfile
	selectedProfile = ""
	t.Cleanup(func() {
		selectedProfile = previousProfile
	})

	if _, err := getASCClient(); err != nil {
		t.Fatalf("expected env credentials to override config, got %v", err)
	}
}

func resetPrivateKeyTemp(t *testing.T) {
	t.Helper()
	if privateKeyTempPath != "" {
		_ = os.Remove(privateKeyTempPath)
		privateKeyTempPath = ""
	}
	t.Cleanup(func() {
		if privateKeyTempPath != "" {
			_ = os.Remove(privateKeyTempPath)
			privateKeyTempPath = ""
		}
	})
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "")
	t.Setenv("ASC_PRIVATE_KEY", "")
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
}

func writeECDSAPEM(t *testing.T, path string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key error: %v", err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if data == nil {
		t.Fatal("failed to encode PEM")
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write key file error: %v", err)
	}
}
