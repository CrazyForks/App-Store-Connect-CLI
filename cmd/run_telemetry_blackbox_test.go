package cmd

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_BuiltBinaryDoesNotWaitForBlockedTelemetryEndpoint(t *testing.T) {
	binaryPath := buildASCBlackboxBinary(t)
	blockedEndpoint, accepted, release := startBlockedTelemetryEndpoint(t)

	disabledHome := t.TempDir()
	runTimedTelemetryCommand(t, binaryPath, disabledHome, true, "")
	disabledDuration := runTimedTelemetryCommand(t, binaryPath, disabledHome, true, "")

	blockedHome := t.TempDir()
	blockedDuration := runTimedTelemetryCommand(t, binaryPath, blockedHome, false, blockedEndpoint)
	t.Logf("foreground timing: disabled=%s blocked=%s added=%s", disabledDuration, blockedDuration, blockedDuration-disabledDuration)
	if added := blockedDuration - disabledDuration; added >= 175*time.Millisecond {
		t.Fatalf(
			"blocked telemetry added %s to foreground runtime (disabled=%s blocked=%s), want less than 175ms",
			added,
			disabledDuration,
			blockedDuration,
		)
	}

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("detached telemetry worker did not reach blocked endpoint")
	}
	release()
}

func runTimedTelemetryCommand(
	t *testing.T,
	binaryPath string,
	home string,
	disabled bool,
	endpoint string,
) time.Duration {
	t.Helper()
	command := exec.Command(binaryPath, "builds", "--definitely-invalid")
	command.Env = telemetryBlackboxEnv(home, disabled, endpoint)
	start := time.Now()
	output, err := command.CombinedOutput()
	duration := time.Since(start)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != ExitUsage {
		t.Fatalf("built command error = %v, want exit %d; output=%s", err, ExitUsage, output)
	}
	return duration
}

func telemetryBlackboxEnv(home string, disabled bool, endpoint string) []string {
	environment := filterEnvVars(
		os.Environ(),
		"ASC_BYPASS_KEYCHAIN",
		"ASC_CONFIG_PATH",
		"ASC_DEBUG",
		"ASC_TELEMETRY_DISABLED",
		"ASC_TELEMETRY_ENDPOINT",
		"ASC_TELEMETRY_EPHEMERAL",
		"ASC_TIMEOUT",
		"ASC_TIMEOUT_SECONDS",
		"DO_NOT_TRACK",
		"HOME",
	)
	disabledValue := ""
	if disabled {
		disabledValue = "1"
	}
	return append(
		environment,
		"ASC_BYPASS_KEYCHAIN=1",
		"ASC_CONFIG_PATH="+filepath.Join(home, "config.json"),
		"ASC_DEBUG=",
		"ASC_TELEMETRY_DISABLED="+disabledValue,
		"ASC_TELEMETRY_ENDPOINT="+endpoint,
		"ASC_TELEMETRY_EPHEMERAL=",
		"ASC_TIMEOUT=1s",
		"ASC_TIMEOUT_SECONDS=",
		"DO_NOT_TRACK=",
		"HOME="+home,
	)
}

func startBlockedTelemetryEndpoint(t *testing.T) (string, <-chan struct{}, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for blocked telemetry endpoint: %v", err)
	}
	accepted := make(chan struct{}, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		accepted <- struct{}{}
		<-release
		_ = connection.Close()
	}()

	released := false
	cleanup := func() {
		if !released {
			close(release)
			released = true
		}
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("blocked telemetry endpoint did not shut down")
		}
	}
	t.Cleanup(cleanup)
	return "https://" + listener.Addr().String() + "/events", accepted, cleanup
}
