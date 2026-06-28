package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSubscriptionsListAppAggregatesGroups(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/subscriptionGroups":
			body := `{"data":[{"type":"subscriptionGroups","id":"group-1"},{"type":"subscriptionGroups","id":"group-2"}],"links":{}}`
			return jsonHTTPResponse(http.StatusOK, body), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/subscriptionGroups/group-1/subscriptions":
			body := `{"data":[{"type":"subscriptions","id":"sub-1","attributes":{"name":"Monthly","productId":"com.example.monthly"}}],"links":{}}`
			return jsonHTTPResponse(http.StatusOK, body), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/subscriptionGroups/group-2/subscriptions":
			body := `{"data":[{"type":"subscriptions","id":"sub-2","attributes":{"name":"Yearly","productId":"com.example.yearly"}}],"links":{}}`
			return jsonHTTPResponse(http.StatusOK, body), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "list",
			"--app", "app-1",
			"--output", "json",
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
	if !strings.Contains(stdout, `"id":"sub-1"`) || !strings.Contains(stdout, `"id":"sub-2"`) {
		t.Fatalf("expected subscriptions from both groups, got %q", stdout)
	}
}
