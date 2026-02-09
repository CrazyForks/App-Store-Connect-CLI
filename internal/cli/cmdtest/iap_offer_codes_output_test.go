package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIAPOfferCodesCreateUsesDefaultEligibilitiesAndParsedPrices(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/inAppPurchaseOfferCodes" {
			t.Fatalf("expected path /v1/inAppPurchaseOfferCodes, got %s", req.URL.Path)
		}

		rawBody, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body error: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(rawBody, &payload); err != nil {
			t.Fatalf("decode request body: %v\nbody=%s", err, string(rawBody))
		}

		data := payload["data"].(map[string]any)
		attrs := data["attributes"].(map[string]any)
		if attrs["name"] != "SPRING" {
			t.Fatalf("expected name SPRING, got %#v", attrs["name"])
		}

		eligibilityItems := attrs["customerEligibilities"].([]any)
		gotEligibilities := make([]string, 0, len(eligibilityItems))
		for _, item := range eligibilityItems {
			gotEligibilities = append(gotEligibilities, item.(string))
		}
		wantEligibilities := []string{"NON_SPENDER", "ACTIVE_SPENDER", "CHURNED_SPENDER"}
		if !slices.Equal(gotEligibilities, wantEligibilities) {
			t.Fatalf("expected default eligibilities %v, got %v", wantEligibilities, gotEligibilities)
		}

		relationships := data["relationships"].(map[string]any)
		iapRelationship := relationships["inAppPurchase"].(map[string]any)["data"].(map[string]any)
		if iapRelationship["id"] != "iap-1" {
			t.Fatalf("expected inAppPurchase id iap-1, got %#v", iapRelationship["id"])
		}

		included := payload["included"].([]any)
		if len(included) != 2 {
			t.Fatalf("expected 2 included price objects, got %d", len(included))
		}

		territoryIDs := make([]string, 0, 2)
		for _, resource := range included {
			relationships := resource.(map[string]any)["relationships"].(map[string]any)
			territory := relationships["territory"].(map[string]any)["data"].(map[string]any)
			territoryIDs = append(territoryIDs, territory["id"].(string))
		}
		if !slices.Equal(territoryIDs, []string{"USA", "JPN"}) {
			t.Fatalf("expected normalized territory ids [USA JPN], got %v", territoryIDs)
		}

		body := `{"data":{"type":"inAppPurchaseOfferCodes","id":"offer-1","attributes":{"name":"SPRING","active":true}}}`
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"iap", "offer-codes", "create",
			"--iap-id", "iap-1",
			"--name", "SPRING",
			"--prices", "usa:pp-us,jpn:pp-jp",
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
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("unmarshal output: %v\nstdout: %s", err, stdout)
	}
	if out.Data.ID != "offer-1" {
		t.Fatalf("expected created offer code id offer-1, got %q", out.Data.ID)
	}
}

func TestIAPOfferCodesCreateReturnsCreateFailure(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/inAppPurchaseOfferCodes" {
			t.Fatalf("expected path /v1/inAppPurchaseOfferCodes, got %s", req.URL.Path)
		}
		body := `{"errors":[{"status":"409","title":"Conflict","detail":"duplicate code"}]}`
		return &http.Response{
			StatusCode: http.StatusConflict,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{
			"iap", "offer-codes", "create",
			"--iap-id", "iap-1",
			"--name", "SPRING",
			"--prices", "usa:pp-us",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(runErr.Error(), "iap offer-codes create: failed to create") {
		t.Fatalf("expected create failure, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
}
