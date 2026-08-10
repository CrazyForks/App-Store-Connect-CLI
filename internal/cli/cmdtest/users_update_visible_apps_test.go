package cmdtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	rootcmd "github.com/rudrankriyam/App-Store-Connect-CLI/cmd"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func TestUsersUpdateSendsRolesAndVisibleAppsInOneRequest(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	var payload asc.UserUpdateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		requests = append(requests, req.Method+" "+req.URL.Path)
		mu.Unlock()

		switch req.URL.Path {
		case "/v1/users/user-1":
			var update asc.UserUpdateRequest
			if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			mu.Lock()
			payload = update
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"type":"users","id":"user-1","attributes":{"roles":["DEVELOPER"],"allAppsVisible":false,"provisioningAllowed":false}},"links":{}}`))
		case "/v1/users/user-1/relationships/visibleApps":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"status":"500","title":"relationship update failed"}]}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	keyPath := filepath.Join(t.TempDir(), "AuthKey.p8")
	writeECDSAPEM(t, keyPath)
	client, err := asc.NewClientWithHTTPClient("TEST_KEY", "TEST_ISSUER", keyPath, &http.Client{Transport: transport})
	if err != nil {
		t.Fatalf("create test client: %v", err)
	}
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	}))

	var exitCode int
	stdout, stderr := captureOutput(t, func() {
		exitCode = rootcmd.Run([]string{
			"users", "update",
			"--id", "user-1",
			"--roles", "developer",
			"--visible-app", "app-2, app-1",
			"--output", "json",
		}, "1.2.3")
	})

	mu.Lock()
	gotRequests := append([]string(nil), requests...)
	gotPayload := payload
	mu.Unlock()

	if exitCode != rootcmd.ExitSuccess {
		t.Fatalf("exit code = %d, want %d; requests=%v payload=%+v stderr=%q", exitCode, rootcmd.ExitSuccess, gotRequests, gotPayload, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !reflect.DeepEqual(gotRequests, []string{"PATCH /v1/users/user-1"}) {
		t.Fatalf("requests = %v, want one user PATCH", gotRequests)
	}
	if gotPayload.Data.Attributes == nil {
		t.Fatal("expected attributes in user PATCH")
	}
	if !reflect.DeepEqual(gotPayload.Data.Attributes.Roles, []string{"DEVELOPER"}) {
		t.Fatalf("roles = %v, want [DEVELOPER]", gotPayload.Data.Attributes.Roles)
	}
	if gotPayload.Data.Attributes.AllAppsVisible == nil || *gotPayload.Data.Attributes.AllAppsVisible {
		t.Fatalf("allAppsVisible = %v, want false", gotPayload.Data.Attributes.AllAppsVisible)
	}
	if gotPayload.Data.Relationships == nil || gotPayload.Data.Relationships.VisibleApps == nil {
		t.Fatal("expected visibleApps in user PATCH")
	}
	visibleApps := gotPayload.Data.Relationships.VisibleApps.Data
	if len(visibleApps) != 2 || visibleApps[0].Type != asc.ResourceTypeApps || visibleApps[0].ID != "app-2" || visibleApps[1].Type != asc.ResourceTypeApps || visibleApps[1].ID != "app-1" {
		t.Fatalf("visibleApps = %+v, want app-2 then app-1", visibleApps)
	}

	var response asc.UserResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode stdout: %v; stdout=%q", err, stdout)
	}
	if response.Data.ID != "user-1" {
		t.Fatalf("response user ID = %q, want user-1", response.Data.ID)
	}
}
