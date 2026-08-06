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

func TestSubscriptionsPromotionalOffersCreateBuildsInlinePrices(t *testing.T) {
	tests := []struct {
		name                 string
		mode                 string
		prices               string
		wantTerritory        string
		wantPricePoint       string
		wantPricePointLinked bool
	}{
		{name: "paid", mode: "pay_as_you_go", prices: "United States:pp-us", wantTerritory: "USA", wantPricePoint: "pp-us", wantPricePointLinked: true},
		{name: "free trial", mode: "free_trial", prices: "Germany", wantTerritory: "DEU"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			originalTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = originalTransport })
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.Path != "/v1/subscriptionPromotionalOffers" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				data := payload["data"].(map[string]any)
				priceRefs := data["relationships"].(map[string]any)["prices"].(map[string]any)["data"].([]any)
				included := payload["included"].([]any)
				if len(priceRefs) != 1 || len(included) != 1 {
					t.Fatalf("expected one price linkage and included resource, got refs=%#v included=%#v", priceRefs, included)
				}
				priceRef := priceRefs[0].(map[string]any)
				includedPrice := included[0].(map[string]any)
				if priceRef["id"] != includedPrice["id"] || priceRef["type"] != "subscriptionPromotionalOfferPrices" {
					t.Fatalf("price linkage does not match included resource: ref=%#v included=%#v", priceRef, includedPrice)
				}
				relationships := includedPrice["relationships"].(map[string]any)
				territory := relationships["territory"].(map[string]any)["data"].(map[string]any)
				if territory["id"] != test.wantTerritory {
					t.Fatalf("expected territory %s, got %#v", test.wantTerritory, territory)
				}
				pricePoint, hasPricePoint := relationships["subscriptionPricePoint"]
				if hasPricePoint != test.wantPricePointLinked {
					t.Fatalf("subscriptionPricePoint presence = %t, want %t", hasPricePoint, test.wantPricePointLinked)
				}
				if hasPricePoint {
					pricePointID := pricePoint.(map[string]any)["data"].(map[string]any)["id"]
					if pricePointID != test.wantPricePoint {
						t.Fatalf("expected price point %s, got %#v", test.wantPricePoint, pricePointID)
					}
				}
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"type":"subscriptionPromotionalOffers","id":"promo-1"}}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{
					"subscriptions", "offers", "promotional", "create",
					"--subscription-id", "8000000001",
					"--offer-code", "SPRING",
					"--name", "Spring",
					"--offer-duration", "one_month",
					"--offer-mode", test.mode,
					"--number-of-periods", "1",
					"--prices", test.prices,
				}); err != nil {
					t.Fatalf("parse: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run: %v", err)
				}
			})
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			var output struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(stdout), &output); err != nil || output.Data.ID != "promo-1" {
				t.Fatalf("unexpected stdout %q: %v", stdout, err)
			}
		})
	}
}

func TestSubscriptionsPromotionalOffersCreateRejectsInvalidPriceShapeBeforeAuth(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		prices  string
		wantErr string
	}{
		{name: "paid legacy price ID", mode: "PAY_UP_FRONT", prices: "price-1", wantErr: "TERRITORY:PRICE_POINT_ID"},
		{name: "free trial has price point", mode: "FREE_TRIAL", prices: "US:price-1", wantErr: "FREE_TRIAL"},
		{name: "unknown territory", mode: "FREE_TRIAL", prices: "NOT_A_TERRITORY", wantErr: "could not be mapped"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{
					"subscriptions", "offers", "promotional", "create",
					"--subscription-id", "8000000001", "--offer-code", "SPRING", "--name", "Spring",
					"--offer-duration", "ONE_MONTH", "--offer-mode", test.mode,
					"--number-of-periods", "1", "--prices", test.prices,
				}); err != nil {
					t.Fatalf("parse: %v", err)
				}
				runErr = root.Run(context.Background())
			})
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("expected usage error, got %v", runErr)
			}
			if !strings.Contains(stderr, test.wantErr) || stdout != "" {
				t.Fatalf("stdout=%q stderr=%q, want stderr containing %q", stdout, stderr, test.wantErr)
			}
		})
	}
}
