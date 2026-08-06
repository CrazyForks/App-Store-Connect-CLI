package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevicesListUsesBundleIDPlatformFilter(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/devices" {
			t.Fatalf("expected GET /v1/devices, got %s %s", req.Method, req.URL.Path)
		}
		if got := req.URL.Query().Get("filter[platform]"); got != "UNIVERSAL" {
			t.Fatalf("filter[platform] = %q, want UNIVERSAL", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[],"links":{"self":"https://api.appstoreconnect.apple.com/v1/devices"}}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"devices", "list", "--platform", "universal"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestBundleIDsCreateUsesBundleIDPlatformPayload(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/bundleIds" {
			t.Fatalf("expected POST /v1/bundleIds, got %s %s", req.Method, req.URL.Path)
		}
		var payload struct {
			Data struct {
				Type       string `json:"type"`
				Attributes struct {
					Identifier string `json:"identifier"`
					Name       string `json:"name"`
					Platform   string `json:"platform"`
				} `json:"attributes"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload.Data.Type != "bundleIds" || payload.Data.Attributes.Identifier != "com.example.universal" || payload.Data.Attributes.Name != "Universal" || payload.Data.Attributes.Platform != "UNIVERSAL" {
			t.Fatalf("unexpected payload: %+v", payload.Data)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"type":"bundleIds","id":"bundle-1","attributes":{"identifier":"com.example.universal","name":"Universal","platform":"UNIVERSAL"}}}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "create", "--identifier", "com.example.universal", "--name", "Universal", "--platform", "universal"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestBundleIDPlatformCommandsRejectGeneralPlatforms(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "devices list tvOS", args: []string{"devices", "list", "--platform", "TV_OS"}},
		{name: "devices register visionOS", args: []string{"devices", "register", "--name", "Invalid", "--udid", "INVALID", "--platform", "VISION_OS"}},
		{name: "bundle ID create tvOS", args: []string{"bundle-ids", "create", "--identifier", "com.example.invalid", "--name", "Invalid", "--platform", "TV_OS"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			if err := root.Parse(tt.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			_, stderr := captureOutput(t, func() {
				err := root.Run(context.Background())
				if err == nil || !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("error = %v, want usage error", err)
				}
			})
			const diagnostic = "Error: --platform must be one of: IOS, MAC_OS, UNIVERSAL\n"
			if count := strings.Count(stderr, diagnostic); count != 1 {
				t.Fatalf("diagnostic count = %d in stderr %q", count, stderr)
			}
			if !strings.Contains(stderr, "USAGE\n") {
				t.Fatalf("expected usage on stderr, got %q", stderr)
			}
		})
	}
}
