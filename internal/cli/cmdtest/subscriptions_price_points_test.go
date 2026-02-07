package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionsPricePointsListPaginateUsesPerPageTimeout(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_TIMEOUT", "120ms")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requests := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		time.Sleep(70 * time.Millisecond)

		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}

		switch req.URL.RawQuery {
		case "limit=200":
			if req.URL.Path != "/v1/subscriptions/sub-1/pricePoints" {
				t.Fatalf("unexpected first page path: %s", req.URL.Path)
			}
			body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-1"}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/subscriptions/sub-1/pricePoints?cursor=AQ&limit=200"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case "cursor=AQ&limit=200":
			if req.URL.Path != "/v1/subscriptions/sub-1/pricePoints" {
				t.Fatalf("unexpected second page path: %s", req.URL.Path)
			}
			body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-2"}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/subscriptions/sub-1/pricePoints?cursor=BQ&limit=200"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case "cursor=BQ&limit=200":
			if req.URL.Path != "/v1/subscriptions/sub-1/pricePoints" {
				t.Fatalf("unexpected third page path: %s", req.URL.Path)
			}
			body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-3"}],"links":{}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request path/query: %s?%s", req.URL.Path, req.URL.RawQuery)
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "price-points", "list",
			"--subscription-id", "sub-1",
			"--paginate",
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
	if requests != 3 {
		t.Fatalf("expected 3 paginated requests, got %d", requests)
	}
	if !strings.Contains(stdout, `"id":"pp-1"`) || !strings.Contains(stdout, `"id":"pp-2"`) || !strings.Contains(stdout, `"id":"pp-3"`) {
		t.Fatalf("expected aggregated paginated output, got %q", stdout)
	}
}

func TestSubscriptionsPricePointsListStreamRequiresPaginate(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "price-points", "list",
			"--subscription-id", "sub-1",
			"--stream",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatalf("expected error for --stream without --paginate")
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--stream requires --paginate") {
		t.Fatalf("expected --stream requires --paginate error, got %q", stderr)
	}
}

func TestSubscriptionsPricePointsListStreamOutput(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/subscriptions/sub-1/pricePoints" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		query := req.URL.RawQuery
		switch {
		case strings.Contains(query, "limit=200") && !strings.Contains(query, "cursor="):
			body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-1","attributes":{"customerPrice":"1.99"}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/subscriptions/sub-1/pricePoints?cursor=AQ&limit=200"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case strings.Contains(query, "cursor=AQ"):
			body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-2","attributes":{"customerPrice":"2.99"}}],"links":{}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected query: %s", query)
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "price-points", "list",
			"--subscription-id", "sub-1",
			"--paginate",
			"--stream",
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

	// Streaming should produce multiple JSON lines (NDJSON), not one aggregated blob
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines (one per page), got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], `"id":"pp-1"`) {
		t.Fatalf("expected first page to contain pp-1, got %q", lines[0])
	}
	if !strings.Contains(lines[1], `"id":"pp-2"`) {
		t.Fatalf("expected second page to contain pp-2, got %q", lines[1])
	}
}

func TestSubscriptionsPricePointsListTerritoryFilter(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/subscriptions/sub-1/pricePoints" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		query := req.URL.Query()
		if query.Get("filter[territory]") != "USA" {
			t.Fatalf("expected filter[territory]=USA, got %q", query.Get("filter[territory]"))
		}

		body := `{"data":[{"type":"subscriptionPricePoints","id":"pp-usa","attributes":{"customerPrice":"9.99","proceeds":"8.49"}}],"links":{}}`
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
			"subscriptions", "price-points", "list",
			"--subscription-id", "sub-1",
			"--territory", "USA",
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
	if !strings.Contains(stdout, `"id":"pp-usa"`) {
		t.Fatalf("expected filtered output, got %q", stdout)
	}
}
