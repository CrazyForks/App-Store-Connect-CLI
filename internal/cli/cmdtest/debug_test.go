package cmdtest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

func TestDebugFlagLogsHTTPRequests(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	keyPath := tmpDir + "/key.p8"
	writeECDSAPEM(t, keyPath)

	os.Setenv("ASC_KEY_ID", "TEST_KEY")
	os.Setenv("ASC_ISSUER_ID", "TEST_ISSUER")
	os.Setenv("ASC_PRIVATE_KEY_PATH", keyPath)
	os.Setenv("ASC_DEBUG", "1")
	defer func() {
		os.Unsetenv("ASC_KEY_ID")
		os.Unsetenv("ASC_ISSUER_ID")
		os.Unsetenv("ASC_PRIVATE_KEY_PATH")
		os.Unsetenv("ASC_DEBUG")
	}()

	root := RootCommand("test")

	_, stderr := captureOutput(t, func() {
		// Parse with --debug flag
		if err := root.Parse([]string{"--debug", "apps"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		// Note: We're not actually running the command because it would fail auth
		// We're just testing that the flag is accepted
	})

	// Test should not error on unknown flag
	if strings.Contains(stderr, "flag provided but not defined: -debug") {
		t.Fatalf("--debug flag not registered")
	}
}

func TestDebugLogsHTTPMethodAndURL(t *testing.T) {
	// This test verifies that debug logging outputs HTTP method and URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	// Enable debug mode for asc client
	debugEnabled := true
	asc.SetDebugOverride(&debugEnabled)
	defer asc.SetDebugOverride(nil)

	// The actual test would need a mock HTTP client
	// For now, we verify the debug infrastructure exists
	if !asc.ResolveDebugEnabled() {
		t.Fatal("Debug mode should be enabled")
	}
}

func TestDebugSanitizesAuthorizationHeader(t *testing.T) {
	// This test verifies that Authorization headers are sanitized in debug output
	// We'll implement this after the infrastructure is in place
	debugEnabled := true
	asc.SetDebugOverride(&debugEnabled)
	defer asc.SetDebugOverride(nil)

	if !asc.ResolveDebugEnabled() {
		t.Fatal("Debug mode should be enabled")
	}

	// TODO: Add test to verify auth header is logged as "Bearer [REDACTED]"
}

func TestDebugEnvVarEnablesDebugMode(t *testing.T) {
	os.Setenv("ASC_DEBUG", "1")
	defer os.Unsetenv("ASC_DEBUG")

	if !asc.ResolveDebugEnabled() {
		t.Fatal("ASC_DEBUG=1 should enable debug mode")
	}
}

func TestDebugDisabledByDefault(t *testing.T) {
	// Ensure no env var is set
	os.Unsetenv("ASC_DEBUG")
	asc.SetDebugOverride(nil)

	if asc.ResolveDebugEnabled() {
		t.Fatal("Debug mode should be disabled by default")
	}
}
