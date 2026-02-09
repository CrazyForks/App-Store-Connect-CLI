package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppTagsListOutputAndQueryOptions(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1/appTags" {
			t.Fatalf("expected path /v1/apps/app-1/appTags, got %s", req.URL.Path)
		}

		query := req.URL.Query()
		if query.Get("filter[visibleInAppStore]") != "true,false" {
			t.Fatalf("expected visibility filter true,false, got %q", query.Get("filter[visibleInAppStore]"))
		}
		if query.Get("sort") != "-name" {
			t.Fatalf("expected sort -name, got %q", query.Get("sort"))
		}
		if query.Get("fields[appTags]") != "name,visibleInAppStore" {
			t.Fatalf("expected fields[appTags], got %q", query.Get("fields[appTags]"))
		}
		if query.Get("include") != "territories" {
			t.Fatalf("expected include territories, got %q", query.Get("include"))
		}
		if query.Get("fields[territories]") != "currency" {
			t.Fatalf("expected territory fields currency, got %q", query.Get("fields[territories]"))
		}
		if query.Get("limit[territories]") != "2" {
			t.Fatalf("expected territory limit 2, got %q", query.Get("limit[territories]"))
		}
		if query.Get("limit") != "5" {
			t.Fatalf("expected limit 5, got %q", query.Get("limit"))
		}

		body := `{
			"data":[
				{
					"type":"appTags",
					"id":"tag-1",
					"attributes":{"name":"Featured","visibleInAppStore":true}
				}
			],
			"included":[
				{
					"type":"territories",
					"id":"USA",
					"attributes":{"currency":"USD"}
				}
			],
			"links":{"next":""}
		}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-tags", "list",
			"--app", "app-1",
			"--visible-in-app-store", "true,false",
			"--sort", "-name",
			"--fields", "name,visibleInAppStore",
			"--include", "territories",
			"--territory-fields", "currency",
			"--territory-limit", "2",
			"--limit", "5",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var out struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name              string `json:"name"`
				VisibleInAppStore bool   `json:"visibleInAppStore"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("unmarshal output: %v\nstdout: %s", err, stdout)
	}
	if len(out.Data) != 1 {
		t.Fatalf("expected 1 app tag, got %d", len(out.Data))
	}
	if out.Data[0].ID != "tag-1" || out.Data[0].Attributes.Name != "Featured" || !out.Data[0].Attributes.VisibleInAppStore {
		t.Fatalf("unexpected app tag output: %+v", out.Data[0])
	}
}

func TestAppTagsListReturnsFetchFailure(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1/appTags" {
			t.Fatalf("expected path /v1/apps/app-1/appTags, got %s", req.URL.Path)
		}
		body := `{"errors":[{"status":"403","title":"Forbidden","detail":"not allowed"}]}`
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"app-tags", "list", "--app", "app-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(runErr.Error(), "app-tags list: failed to fetch") {
		t.Fatalf("expected fetch failure, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
}
