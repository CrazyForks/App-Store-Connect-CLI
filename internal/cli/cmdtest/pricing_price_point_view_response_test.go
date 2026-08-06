package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPricingPricePointViewPreservesSingleResponse(t *testing.T) {
	setupAuth(t)
	installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v3/appPricePoints/price-point-1" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"appPricePoints","id":"price-point-1","attributes":{"customerPrice":"0.99","proceeds":"0.70"}},"included":[{"type":"territories","id":"USA","attributes":{"currency":"USD"}}],"links":{"self":"https://api.appstoreconnect.apple.com/v3/appPricePoints/price-point-1"}}`), nil
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"pricing", "price-points", "view",
			"--price-point", "price-point-1",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("decode stdout: %v (stdout=%q)", err, stdout)
	}
	var data map[string]any
	if err := json.Unmarshal(envelope["data"], &data); err != nil {
		t.Fatalf("expected data object, got %s: %v", envelope["data"], err)
	}
	if data["id"] != "price-point-1" {
		t.Fatalf("expected price-point-1, got %v", data["id"])
	}
	var included []map[string]any
	if err := json.Unmarshal(envelope["included"], &included); err != nil {
		t.Fatalf("decode included: %v", err)
	}
	if len(included) != 1 || included[0]["id"] != "USA" {
		t.Fatalf("unexpected included resources: %#v", included)
	}
}
