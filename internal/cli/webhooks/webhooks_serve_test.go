package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebhooksServeReceivesAndWritesEvent(t *testing.T) {
	eventsDir := filepath.Join(t.TempDir(), "events")
	port := freeLocalPort(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	cmd := WebhooksServeCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.Parse([]string{
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
		"--dir", eventsDir,
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	go func() {
		errCh <- cmd.Run(ctx)
	}()
	defer shutdownServeCommand(t, cancel, errCh)

	statusCode := postJSONWithRetry(t, fmt.Sprintf("http://127.0.0.1:%d", port), `{
		"id":"evt-123",
		"eventType":"BUILD_UPLOAD_STATE_UPDATED",
		"data":{"type":"webhookEvents"}
	}`)
	if statusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, statusCode)
	}

	payload := waitForJSONPayloadFile(t, eventsDir)
	if payload["id"] != "evt-123" {
		t.Fatalf("expected id evt-123, got %v", payload["id"])
	}
	if payload["eventType"] != "BUILD_UPLOAD_STATE_UPDATED" {
		t.Fatalf("expected eventType BUILD_UPLOAD_STATE_UPDATED, got %v", payload["eventType"])
	}
}

func TestWebhooksServeExecReceivesPayload(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "payload.json")
	port := freeLocalPort(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	cmd := WebhooksServeCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.Parse([]string{
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
		"--exec", fmt.Sprintf("cat > %q", outPath),
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	go func() {
		errCh <- cmd.Run(ctx)
	}()
	defer shutdownServeCommand(t, cancel, errCh)

	statusCode := postJSONWithRetry(t, fmt.Sprintf("http://127.0.0.1:%d", port), `{"id":"evt-exec-1","eventType":"TEST_EVENT"}`)
	if statusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, statusCode)
	}

	waitForFile(t, outPath)
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exec output file: %v", err)
	}
	if !strings.Contains(string(content), `"id":"evt-exec-1"`) {
		t.Fatalf("expected payload file to contain event id, got %q", string(content))
	}
}

func TestReadWebhookServeJSONPayloadAllowsMaxInt64Limit(t *testing.T) {
	payload, err := readWebhookServeJSONPayload(
		io.NopCloser(strings.NewReader(`{"id":"evt-max-int","eventType":"TEST_EVENT"}`)),
		math.MaxInt64,
	)
	if err != nil {
		t.Fatalf("expected payload to be accepted with max int64 limit, got error: %v", err)
	}
	if got, want := string(payload), `{"id":"evt-max-int","eventType":"TEST_EVENT"}`; got != want {
		t.Fatalf("expected compact payload %q, got %q", want, got)
	}
}

func freeLocalPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP addr, got %T", listener.Addr())
	}
	return tcpAddr.Port
}

func postJSONWithRetry(t *testing.T, baseURL string, payload string) int {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Post(baseURL, "application/json", strings.NewReader(payload))
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}

	t.Fatalf("timed out posting webhook payload to %s", baseURL)
	return 0
}

func waitForJSONPayloadFile(t *testing.T, dir string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if strings.HasSuffix(entry.Name(), ".json") {
					data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
					if readErr != nil {
						continue
					}

					var payload map[string]any
					if err := json.Unmarshal(data, &payload); err == nil {
						return payload
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for a valid JSON payload file in %q", dir)
	return nil
}

func waitForFile(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for file %q", path)
}

func shutdownServeCommand(t *testing.T, cancel context.CancelFunc, errCh <-chan error) {
	t.Helper()

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve command returned error on shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for serve command shutdown")
	}
}
