package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// Regression coverage for https://github.com/rorkai/App-Store-Connect-CLI/issues/1948:
// FREE_TRIAL win-back offers must be creatable without --price, because the
// API rejects subscriptionPricePoint relationships on free offers.

func winBackFreeTrialCreateArgs(extra ...string) []string {
	args := []string{
		"subscriptions", "offers", "win-back", "create",
		"--subscription-id", "1234567890",
		"--reference-name", "Yearly winback",
		"--offer-id", "yearly_winback_1mo",
		"--duration", "ONE_MONTH",
		"--offer-mode", "FREE_TRIAL",
		"--period-count", "1",
		"--eligibility-paid-months", "1",
		"--eligibility-last-subscribed-min", "1",
		"--eligibility-last-subscribed-max", "12",
		"--start-date", "2026-08-11",
		"--priority", "HIGH",
	}
	return append(args, extra...)
}

func TestWinBackOffersCreateFreeTrialSendsTerritoryOnlyPrices(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", req.Method)
		}
		if req.URL.Path != "/v1/winBackOffers" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var payload struct {
			Data struct {
				Attributes struct {
					OfferMode string `json:"offerMode"`
				} `json:"attributes"`
				Relationships struct {
					Prices struct {
						Data []struct {
							Type string `json:"type"`
							ID   string `json:"id"`
						} `json:"data"`
					} `json:"prices"`
				} `json:"relationships"`
			} `json:"data"`
			Included []struct {
				Type          string                     `json:"type"`
				ID            string                     `json:"id"`
				Relationships map[string]json.RawMessage `json:"relationships"`
			} `json:"included"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload.Data.Attributes.OfferMode != "FREE_TRIAL" {
			t.Fatalf("expected offerMode FREE_TRIAL, got %q", payload.Data.Attributes.OfferMode)
		}
		if got := len(payload.Data.Relationships.Prices.Data); got != 2 {
			t.Fatalf("expected 2 price linkages, got %d", got)
		}
		if got := len(payload.Included); got != 2 {
			t.Fatalf("expected 2 included prices, got %d", got)
		}
		wantTerritories := []string{"USA", "FRA"}
		for i, included := range payload.Included {
			if included.Type != "winBackOfferPrices" {
				t.Fatalf("included[%d] type = %q, want winBackOfferPrices", i, included.Type)
			}
			if _, ok := included.Relationships["subscriptionPricePoint"]; ok {
				t.Fatalf("included[%d] must not carry subscriptionPricePoint for FREE_TRIAL: %s", i, body)
			}
			var territory struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal(included.Relationships["territory"], &territory); err != nil {
				t.Fatalf("included[%d] territory decode error: %v", i, err)
			}
			if territory.Data.Type != "territories" || territory.Data.ID != wantTerritories[i] {
				t.Fatalf("included[%d] territory = %+v, want %s", i, territory.Data, wantTerritories[i])
			}
		}
		return jsonResponse(http.StatusCreated, `{"data":{"type":"winBackOffers","id":"offer-1","attributes":{"referenceName":"Yearly winback","offerId":"yearly_winback_1mo"}}}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(winBackFreeTrialCreateArgs("--territory", "US,France")); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Fatal("expected JSON output")
	}
}

func TestWinBackOffersCreateFreeTrialRejectsPrice(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	assertUsageExit(t, winBackFreeTrialCreateArgs(
		"--territory", "USA",
		"--price", "eyJzIjoiMTIzNCIsInQiOiJVU0EiLCJwIjoiMTAwOTkifQ",
	), "--price is not supported when --offer-mode is FREE_TRIAL")
}

func TestWinBackOffersCreateFreeTrialRequiresTerritory(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	assertUsageExit(t, winBackFreeTrialCreateArgs(),
		"--territory is required when --offer-mode is FREE_TRIAL")
}

func TestWinBackOffersCreatePaidRejectsTerritory(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	args := winBackFreeTrialCreateArgs("--territory", "USA")
	for i, arg := range args {
		if arg == "FREE_TRIAL" {
			args[i] = "PAY_AS_YOU_GO"
		}
	}
	assertUsageExit(t, args,
		"--territory is only supported when --offer-mode is FREE_TRIAL")
}
