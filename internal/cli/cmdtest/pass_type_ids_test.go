package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/rudrankriyam/App-Store-Connect-CLI/cmd"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func TestPassTypeIDsCertificateQueryFlagsRequireIncludeBeforeClient(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{name: "list", args: []string{"pass-type-ids", "list"}},
		{name: "view", args: []string{"pass-type-ids", "view", "--pass-type-id", "PASS_ID"}},
	}
	flags := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "certificate fields",
			args:    []string{"--certificate-fields", "name"},
			wantErr: "Error: --certificate-fields requires --include certificates",
		},
		{
			name:    "certificate limit",
			args:    []string{"--limit-certificates", "1"},
			wantErr: "Error: --limit-certificates requires --include certificates",
		},
		{
			name:    "deterministic precedence",
			args:    []string{"--limit-certificates", "1", "--certificate-fields", "name"},
			wantErr: "Error: --certificate-fields requires --include certificates",
		},
	}

	for _, command := range commands {
		for _, flagCase := range flags {
			t.Run(command.name+"/"+flagCase.name, func(t *testing.T) {
				clientFactoryCalls := 0
				restore := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
					clientFactoryCalls++
					return nil, errors.New("client factory must not run during validation")
				})
				t.Cleanup(restore)

				var code int
				stdout, stderr := captureOutput(t, func() {
					args := append(append([]string(nil), command.args...), flagCase.args...)
					code = rootcmd.Run(args, "1.2.3")
				})

				if code != rootcmd.ExitUsage {
					t.Errorf("exit code = %d, want %d", code, rootcmd.ExitUsage)
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				if count := strings.Count(stderr, flagCase.wantErr); count != 1 {
					t.Errorf("stderr contains %q %d times, want once: %q", flagCase.wantErr, count, stderr)
				}
				if clientFactoryCalls != 0 {
					t.Errorf("client factory calls = %d, want 0", clientFactoryCalls)
				}
			})
		}
	}
}

func TestPassTypeIDsCertificateQueryFlagsPreserveValidRequests(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		path         string
		response     string
		wantID       string
		collection   bool
		wantIncluded bool
	}{
		{
			name: "list",
			args: []string{
				"pass-type-ids", "list",
				"--fields", "certificates",
				"--include", "certificates",
				"--certificate-fields", "name,expirationDate",
				"--limit-certificates", "1",
				"--output", "json",
			},
			path:         "/v1/passTypeIds",
			response:     `{"data":[{"type":"passTypeIds","id":"PASS_LIST","attributes":{"name":"List pass"}}],"included":[{"type":"certificates","id":"CERT_LIST","attributes":{"name":"List certificate"}}]}`,
			wantID:       "PASS_LIST",
			collection:   true,
			wantIncluded: true,
		},
		{
			name: "view",
			args: []string{
				"pass-type-ids", "view", "--pass-type-id", "PASS_VIEW",
				"--fields", "certificates",
				"--include", "certificates",
				"--certificate-fields", "name,expirationDate",
				"--limit-certificates", "1",
				"--output", "json",
			},
			path:         "/v1/passTypeIds/PASS_VIEW",
			response:     `{"data":{"type":"passTypeIds","id":"PASS_VIEW","attributes":{"name":"Viewed pass"}},"included":[{"type":"certificates","id":"CERT_VIEW","attributes":{"name":"Viewed certificate"}}]}`,
			wantID:       "PASS_VIEW",
			collection:   false,
			wantIncluded: true,
		},
		{
			name: "relationship field without include",
			args: []string{
				"pass-type-ids", "list",
				"--fields", "certificates",
				"--output", "json",
			},
			path:       "/v1/passTypeIds",
			response:   `{"data":[{"type":"passTypeIds","id":"PASS_FIELDS","relationships":{"certificates":{"links":{"related":"https://api.appstoreconnect.apple.com/v1/passTypeIds/PASS_FIELDS/certificates"}}}}]}`,
			wantID:     "PASS_FIELDS",
			collection: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requestCount++
				if req.Method != http.MethodGet || req.URL.Path != test.path {
					t.Errorf("request = %s %s, want GET %s", req.Method, req.URL.Path, test.path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				wantQuery := url.Values{"fields[passTypeIds]": {"certificates"}}
				if test.wantIncluded {
					wantQuery.Set("fields[certificates]", "name,expirationDate")
					wantQuery.Set("include", "certificates")
					wantQuery.Set("limit[certificates]", "1")
				}
				if got := req.URL.Query().Encode(); got != wantQuery.Encode() {
					t.Errorf("query = %q, want %q", got, wantQuery.Encode())
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, test.response)
			}))
			t.Cleanup(server.Close)

			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("parse server URL: %v", err)
			}
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Scheme != "https" || req.URL.Host != "api.appstoreconnect.apple.com" {
					t.Errorf("request URL = %s, want official App Store Connect host", req.URL.String())
				}
				if authorization := req.Header.Get("Authorization"); !strings.HasPrefix(authorization, "Bearer ") || authorization == "Bearer " {
					t.Errorf("Authorization = %q, want non-empty bearer token", authorization)
				}
				cloned := req.Clone(req.Context())
				cloned.URL.Scheme = serverURL.Scheme
				cloned.URL.Host = serverURL.Host
				return server.Client().Transport.RoundTrip(cloned)
			})
			client, err := asc.NewClientWithHTTPClient(
				"TEST_KEY",
				"TEST_ISSUER",
				os.Getenv("ASC_PRIVATE_KEY_PATH"),
				&http.Client{Transport: transport},
			)
			if err != nil {
				t.Fatalf("new client: %v", err)
			}
			t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				return client, nil
			}))

			root := RootCommand("1.2.3")
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if requestCount != 1 {
				t.Fatalf("request count = %d, want 1", requestCount)
			}
			assertPassTypeIDResponseShape(t, stdout, test.wantID, test.collection, test.wantIncluded)
		})
	}
}

func assertPassTypeIDResponseShape(t *testing.T, output, wantID string, collection, wantIncluded bool) {
	t.Helper()

	var envelope struct {
		Data     json.RawMessage `json:"data"`
		Included []struct {
			ID string `json:"id"`
		} `json:"included"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("parse JSON output: %v\noutput: %s", err, output)
	}

	if collection {
		var resources []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(envelope.Data, &resources); err != nil {
			t.Fatalf("parse collection data: %v", err)
		}
		if len(resources) != 1 || resources[0].ID != wantID {
			t.Fatalf("collection data = %#v, want one resource with ID %q", resources, wantID)
		}
	} else {
		var resource struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(envelope.Data, &resource); err != nil {
			t.Fatalf("parse instance data: %v", err)
		}
		if resource.ID != wantID {
			t.Fatalf("instance ID = %q, want %q", resource.ID, wantID)
		}
	}

	if wantIncluded {
		if len(envelope.Included) != 1 || !strings.HasPrefix(envelope.Included[0].ID, "CERT_") {
			t.Fatalf("included = %#v, want one certificate", envelope.Included)
		}
	} else if len(envelope.Included) != 0 {
		t.Fatalf("included = %#v, want none", envelope.Included)
	}
}

func TestPassTypeIDsValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "pass-type-ids list missing pass type id for certificates list",
			args:    []string{"pass-type-ids", "certificates", "list"},
			wantErr: "--pass-type-id is required",
		},
		{
			name:    "pass-type-ids certificates view missing pass type id",
			args:    []string{"pass-type-ids", "certificates", "view"},
			wantErr: "--pass-type-id is required",
		},
		{
			name:    "pass-type-ids view missing pass type id",
			args:    []string{"pass-type-ids", "view"},
			wantErr: "--pass-type-id is required",
		},
		{
			name:    "pass-type-ids create missing identifier",
			args:    []string{"pass-type-ids", "create", "--name", "Example"},
			wantErr: "--identifier is required",
		},
		{
			name:    "pass-type-ids create missing name",
			args:    []string{"pass-type-ids", "create", "--identifier", "pass.com.example"},
			wantErr: "--name is required",
		},
		{
			name:    "pass-type-ids update missing pass type id",
			args:    []string{"pass-type-ids", "update", "--name", "Updated"},
			wantErr: "--pass-type-id is required",
		},
		{
			name:    "pass-type-ids update missing name",
			args:    []string{"pass-type-ids", "update", "--pass-type-id", "PASS_ID"},
			wantErr: "--name is required",
		},
		{
			name:    "pass-type-ids delete missing pass type id",
			args:    []string{"pass-type-ids", "delete", "--confirm"},
			wantErr: "--pass-type-id is required",
		},
		{
			name:    "pass-type-ids delete missing confirm",
			args:    []string{"pass-type-ids", "delete", "--pass-type-id", "PASS_ID"},
			wantErr: "--confirm is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}
