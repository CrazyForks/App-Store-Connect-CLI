package builds

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

type buildExpireAllRequestDeadline struct {
	method    string
	path      string
	remaining time.Duration
}

type buildExpireAllDeadlineTransport struct {
	base http.RoundTripper

	mu       sync.Mutex
	requests []buildExpireAllRequestDeadline
}

func (t *buildExpireAllDeadlineTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	remaining := time.Duration(0)
	if ok {
		remaining = time.Until(deadline)
	}

	t.mu.Lock()
	t.requests = append(t.requests, buildExpireAllRequestDeadline{
		method:    req.Method,
		path:      req.URL.Path,
		remaining: remaining,
	})
	t.mu.Unlock()

	return t.base.RoundTrip(req)
}

func (t *buildExpireAllDeadlineTransport) snapshot() []buildExpireAllRequestDeadline {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]buildExpireAllRequestDeadline(nil), t.requests...)
}

func TestBuildsExpireAllUsesFreshRequestDeadlines(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "500ms")
	t.Setenv("ASC_MAX_RETRIES", "0")

	var requestCount atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		index := int(requestCount.Add(1) - 1)

		if req.Header.Get("Authorization") == "" {
			t.Errorf("request %d missing authorization header", index+1)
		}

		w.Header().Set("Content-Type", "application/json")
		switch index {
		case 0:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/builds" {
				t.Errorf("first request = %s %s, want GET /v1/builds", req.Method, req.URL.Path)
			}
			query := req.URL.Query()
			if query.Get("filter[app]") != "app-1" || query.Get("limit") != "200" || query.Get("sort") != "-uploadedDate" {
				t.Errorf("first request query = %q, want app filter, limit 200, and descending upload date", req.URL.RawQuery)
			}
			time.Sleep(100 * time.Millisecond)
			_, _ = io.WriteString(w, `{"data":[{"type":"builds","id":"build-new","attributes":{"version":"2.0","uploadedDate":"2026-01-02T00:00:00Z","expired":false}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/builds?cursor=next"}}`)
		case 1:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/builds" || req.URL.Query().Get("cursor") != "next" {
				t.Errorf("second request = %s %s?%s, want paginated GET", req.Method, req.URL.Path, req.URL.RawQuery)
			}
			time.Sleep(100 * time.Millisecond)
			_, _ = io.WriteString(w, `{"data":[{"type":"builds","id":"build-old","attributes":{"version":"1.0","uploadedDate":"2026-01-01T00:00:00Z","expired":false}}],"links":{}}`)
		case 2, 3:
			wantID := "build-new"
			if index == 3 {
				wantID = "build-old"
			}
			if req.Method != http.MethodPatch || req.URL.Path != "/v1/builds/"+wantID {
				t.Errorf("request %d = %s %s, want PATCH /v1/builds/%s", index+1, req.Method, req.URL.Path, wantID)
			}
			var payload struct {
				Data struct {
					Type       string `json:"type"`
					ID         string `json:"id"`
					Attributes struct {
						Expired bool `json:"expired"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Errorf("decode request %d body: %v", index+1, err)
			}
			if payload.Data.Type != "builds" || payload.Data.ID != wantID || !payload.Data.Attributes.Expired {
				t.Errorf("request %d payload = %+v, want expired build %s", index+1, payload.Data, wantID)
			}
			if index == 2 {
				time.Sleep(100 * time.Millisecond)
			}
			_, _ = io.WriteString(w, `{"data":{"type":"builds","id":"`+wantID+`","attributes":{"version":"1.0","uploadedDate":"2026-01-01T00:00:00Z","expired":true}},"links":{}}`)
		default:
			t.Errorf("unexpected request %d: %s %s", index+1, req.Method, req.URL.String())
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	client, recorder := newBuildExpireAllDeadlineClient(t, server)
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	}))

	cmd := BuildsExpireAllCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.Parse([]string{
		"--app", "app-1",
		"--older-than", "2026-02-01",
		"--confirm",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse command: %v", err)
	}

	stdout, stderr, runErr := captureBuildExpireAllOutput(t, func() error {
		return cmd.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run command: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var result asc.BuildExpireAllResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode output %q: %v", stdout, err)
	}
	if result.SelectedCount != 2 || result.ExpiredCount != 2 || len(result.Failures) != 0 {
		t.Fatalf("unexpected summary: %+v", result)
	}
	if len(result.Builds) != 2 || result.Builds[0].ID != "build-new" || result.Builds[1].ID != "build-old" {
		t.Fatalf("unexpected build order: %+v", result.Builds)
	}

	requests := recorder.snapshot()
	wantRequests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/v1/builds"},
		{method: http.MethodGet, path: "/v1/builds"},
		{method: http.MethodPatch, path: "/v1/builds/build-new"},
		{method: http.MethodPatch, path: "/v1/builds/build-old"},
	}
	if len(requests) != len(wantRequests) {
		t.Fatalf("requests = %+v, want %d requests", requests, len(wantRequests))
	}
	for i, want := range wantRequests {
		got := requests[i]
		if got.method != want.method || got.path != want.path {
			t.Errorf("request %d = %s %s, want %s %s", i+1, got.method, got.path, want.method, want.path)
		}
		if got.remaining < 350*time.Millisecond || got.remaining > 500*time.Millisecond {
			t.Errorf("request %d %s %s deadline remaining = %s, want a fresh timeout near 500ms", i+1, got.method, got.path, got.remaining)
		}
	}
}

func newBuildExpireAllDeadlineClient(t *testing.T, server *httptest.Server) (*asc.Client, *buildExpireAllDeadlineTransport) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error: %v", err)
	}
	keyPath := filepath.Join(t.TempDir(), "AuthKey.p8")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	transport, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server transport type = %T, want *http.Transport", server.Client().Transport)
	}
	transport = transport.Clone()
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	recorder := &buildExpireAllDeadlineTransport{base: transport}

	client, err := asc.NewClientWithHTTPClient("KEY123", "ISS456", keyPath, &http.Client{Transport: recorder})
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient() error: %v", err)
	}
	return client, recorder
}

func captureBuildExpireAllOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	runErr := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return <-outC, <-errC, runErr
}

func TestParseOlderThanDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "days", input: "90d", want: 90 * 24 * time.Hour},
		{name: "weeks", input: "2w", want: 14 * 24 * time.Hour},
		{name: "months", input: "3m", want: 90 * 24 * time.Hour},
		{name: "uppercase unit", input: "10D", want: 10 * 24 * time.Hour},
		{name: "empty", input: "", wantErr: true},
		{name: "missing unit", input: "10", wantErr: true},
		{name: "zero", input: "0d", wantErr: true},
		{name: "bad unit", input: "10y", wantErr: true},
		{name: "bad number", input: "xd", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOlderThanDuration(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestParseOlderThanThreshold(t *testing.T) {
	now := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "date only",
			input: "2026-01-01",
			want:  time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "rfc3339",
			input: "2026-01-01T08:30:00Z",
			want:  time.Date(2026, time.January, 1, 8, 30, 0, 0, time.UTC),
		},
		{
			name:  "duration",
			input: "7d",
			want:  now.Add(-(7 * 24 * time.Hour)),
		},
		{
			name:    "invalid",
			input:   "not-a-threshold",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOlderThanThreshold(test.input, now)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(test.want) {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

func TestBuildExpireAllItem(t *testing.T) {
	item := buildExpireAllItem(buildExpireCandidate{
		resource: asc.Resource[asc.BuildAttributes]{
			ID: "build-1",
			Attributes: asc.BuildAttributes{
				Version:      "1.2.3",
				UploadedDate: "2026-01-01T00:00:00Z",
			},
		},
		ageDays: 40,
	})

	if item.ID != "build-1" || item.Version != "1.2.3" || item.AgeDays != 40 {
		t.Fatalf("unexpected expire-all item: %+v", item)
	}
}
