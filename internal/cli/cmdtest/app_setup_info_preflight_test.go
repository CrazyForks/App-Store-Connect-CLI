package cmdtest

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAppSetupInfoSetPlansAmbiguousLocalizationTargetBeforeMutation(t *testing.T) {
	setupAppSetupInfoPreflightAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	var requests []string
	mutationCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		if appSetupInfoMutation(req.Method) {
			mutationCount++
		}

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/appInfos":
			return appSetupInfoResponse(http.StatusOK, `{
				"data":[
					{"type":"appInfos","id":"info-1","attributes":{"state":"READY_FOR_SALE"}},
					{"type":"appInfos","id":"info-2","attributes":{"state":"READY_FOR_SALE"}}
				]
			}`), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/apps/app-1":
			return appSetupInfoResponse(http.StatusOK, `{"data":{"type":"apps","id":"app-1","attributes":{"bundleId":"com.example.changed"}}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	_, _, runErr := runAppSetupInfoSet(
		t,
		"--app", "app-1",
		"--bundle-id", "com.example.changed",
		"--locale", "en-US",
		"--name", "Example App",
	)
	if runErr == nil || !strings.Contains(runErr.Error(), "multiple app infos found") {
		t.Fatalf("expected ambiguous app info error, got %v", runErr)
	}
	if mutationCount != 0 {
		t.Errorf("mutation count = %d, want 0", mutationCount)
	}
	wantRequests := []string{"GET /v1/apps/app-1/appInfos"}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Errorf("requests = %v, want %v", requests, wantRequests)
	}
}

func TestAppSetupInfoSetRejectsCreateWithoutNameBeforeMutation(t *testing.T) {
	setupAppSetupInfoPreflightAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	var requests []string
	mutationCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		if appSetupInfoMutation(req.Method) {
			mutationCount++
		}

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appInfos/info-1/appInfoLocalizations":
			assertAppSetupInfoLocalizationQuery(t, req)
			return appSetupInfoResponse(http.StatusOK, `{"data":[]}`), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/apps/app-1":
			return appSetupInfoResponse(http.StatusOK, `{"data":{"type":"apps","id":"app-1","attributes":{"bundleId":"com.example.changed"}}}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appInfoLocalizations":
			return appSetupInfoResponse(http.StatusCreated, `{"data":{"type":"appInfoLocalizations","id":"loc-created","attributes":{"locale":"en-US","subtitle":"Subtitle"}}}`), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	stdout, stderr, runErr := runAppSetupInfoSet(
		t,
		"--app", "app-1",
		"--app-info", "info-1",
		"--bundle-id", "com.example.changed",
		"--locale", "en-US",
		"--subtitle", "Subtitle",
	)
	if runErr == nil || !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(runErr.Error(), "--name is required when creating an app info localization") {
		t.Errorf("expected create name usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Error: --name is required when creating an app info localization") {
		t.Errorf("stderr = %q, want missing name diagnostic", stderr)
	}
	if mutationCount != 0 {
		t.Errorf("mutation count = %d, want 0", mutationCount)
	}
	wantRequests := []string{"GET /v1/appInfos/info-1/appInfoLocalizations"}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Errorf("requests = %v, want %v", requests, wantRequests)
	}
}

func TestAppSetupInfoSetPlansExplicitLocalizationTargetBeforeWrites(t *testing.T) {
	tests := []struct {
		name                 string
		localizations        string
		args                 []string
		localizationMethod   string
		localizationPath     string
		localizationRequest  string
		localizationResponse string
	}{
		{
			name:                "create",
			localizations:       `{"data":[]}`,
			args:                []string{"--name", "Example App", "--subtitle", "Subtitle"},
			localizationMethod:  http.MethodPost,
			localizationPath:    "/v1/appInfoLocalizations",
			localizationRequest: `{"data":{"type":"appInfoLocalizations","attributes":{"locale":"en-US","name":"Example App","subtitle":"Subtitle"},"relationships":{"appInfo":{"data":{"type":"appInfos","id":"info-1"}}}}}`,
			localizationResponse: `{
				"data":{"type":"appInfoLocalizations","id":"loc-created","attributes":{"locale":"en-US","name":"Example App","subtitle":"Subtitle"}}
			}`,
		},
		{
			name:                "update without name",
			localizations:       `{"data":[{"type":"appInfoLocalizations","id":"loc-1","attributes":{"locale":"en-US","name":"Existing App"}}]}`,
			args:                []string{"--subtitle", "Updated Subtitle", "--privacy-policy-url", "https://example.com/privacy"},
			localizationMethod:  http.MethodPatch,
			localizationPath:    "/v1/appInfoLocalizations/loc-1",
			localizationRequest: `{"data":{"type":"appInfoLocalizations","id":"loc-1","attributes":{"subtitle":"Updated Subtitle","privacyPolicyUrl":"https://example.com/privacy"}}}`,
			localizationResponse: `{
				"data":{"type":"appInfoLocalizations","id":"loc-1","attributes":{"locale":"en-US","name":"Existing App","subtitle":"Updated Subtitle","privacyPolicyUrl":"https://example.com/privacy"}}
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAppSetupInfoPreflightAuth(t)

			originalTransport := http.DefaultTransport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
			})

			var requests []string
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests = append(requests, req.Method+" "+req.URL.Path)

				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v1/appInfos/info-1/appInfoLocalizations":
					assertAppSetupInfoLocalizationQuery(t, req)
					return appSetupInfoResponse(http.StatusOK, test.localizations), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/v1/apps/app-1":
					assertJSONDocument(t, req.Body, `{"data":{"type":"apps","id":"app-1","attributes":{"bundleId":"com.example.changed"}}}`)
					return appSetupInfoResponse(http.StatusOK, `{"data":{"type":"apps","id":"app-1","attributes":{"bundleId":"com.example.changed"}}}`), nil
				case req.Method == test.localizationMethod && req.URL.Path == test.localizationPath:
					assertJSONDocument(t, req.Body, test.localizationRequest)
					return appSetupInfoResponse(http.StatusOK, test.localizationResponse), nil
				default:
					return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
				}
			})

			args := []string{
				"--app", "app-1",
				"--app-info", "info-1",
				"--bundle-id", "com.example.changed",
				"--locale", "en-US",
			}
			args = append(args, test.args...)
			stdout, stderr, runErr := runAppSetupInfoSet(t, args...)
			if runErr != nil {
				t.Fatalf("run error: %v (stderr=%q)", runErr, stderr)
			}
			if !strings.Contains(stdout, `"appId":"app-1"`) {
				t.Fatalf("unexpected stdout: %q", stdout)
			}

			wantRequests := []string{
				"GET /v1/appInfos/info-1/appInfoLocalizations",
				"PATCH /v1/apps/app-1",
				test.localizationMethod + " " + test.localizationPath,
			}
			if !reflect.DeepEqual(requests, wantRequests) {
				t.Errorf("requests = %v, want %v", requests, wantRequests)
			}
		})
	}
}

func setupAppSetupInfoPreflightAuth(t *testing.T) {
	t.Helper()
	setupAuth(t)
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
}

func runAppSetupInfoSet(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		parseArgs := append([]string{"app-setup", "info", "set"}, args...)
		if err := root.Parse(parseArgs); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func assertAppSetupInfoLocalizationQuery(t *testing.T, req *http.Request) {
	t.Helper()
	if got := req.URL.Query().Get("filter[locale]"); got != "en-US" {
		t.Fatalf("filter[locale] = %q, want en-US", got)
	}
	if got := req.URL.Query().Get("limit"); got != "200" {
		t.Fatalf("limit = %q, want 200", got)
	}
}

func appSetupInfoMutation(method string) bool {
	switch method {
	case http.MethodPatch, http.MethodPost, http.MethodDelete:
		return true
	default:
		return false
	}
}

func appSetupInfoResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
