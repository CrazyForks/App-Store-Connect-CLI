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

func TestSubscriptionsOfferCodesCreateNormalizesValuesAndBuildsPayload(t *testing.T) {
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
		if req.URL.Path != "/v1/subscriptionOfferCodes" {
			t.Fatalf("expected path /v1/subscriptionOfferCodes, got %s", req.URL.Path)
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
		if attrs["name"] != "Spring Promo" {
			t.Fatalf("expected name Spring Promo, got %#v", attrs["name"])
		}
		if attrs["offerEligibility"] != "REPLACE_INTRO_OFFERS" {
			t.Fatalf("expected normalized offerEligibility REPLACE_INTRO_OFFERS, got %#v", attrs["offerEligibility"])
		}
		if attrs["duration"] != "ONE_MONTH" {
			t.Fatalf("expected normalized duration ONE_MONTH, got %#v", attrs["duration"])
		}
		if attrs["offerMode"] != "FREE_TRIAL" {
			t.Fatalf("expected normalized offerMode FREE_TRIAL, got %#v", attrs["offerMode"])
		}
		if attrs["numberOfPeriods"] != float64(2) {
			t.Fatalf("expected numberOfPeriods 2, got %#v", attrs["numberOfPeriods"])
		}
		if attrs["autoRenewEnabled"] != true {
			t.Fatalf("expected autoRenewEnabled true, got %#v", attrs["autoRenewEnabled"])
		}

		eligibilityItems := attrs["customerEligibilities"].([]any)
		gotEligibilities := make([]string, 0, len(eligibilityItems))
		for _, item := range eligibilityItems {
			gotEligibilities = append(gotEligibilities, item.(string))
		}
		wantEligibilities := []string{"NEW", "EXISTING"}
		if !slices.Equal(gotEligibilities, wantEligibilities) {
			t.Fatalf("expected customer eligibilities %v, got %v", wantEligibilities, gotEligibilities)
		}

		subscriptionRelationship := data["relationships"].(map[string]any)["subscription"].(map[string]any)["data"].(map[string]any)
		if subscriptionRelationship["id"] != "sub-1" {
			t.Fatalf("expected subscription id sub-1, got %#v", subscriptionRelationship["id"])
		}

		included := payload["included"].([]any)
		if len(included) != 1 {
			t.Fatalf("expected 1 included price object, got %d", len(included))
		}
		territory := included[0].(map[string]any)["relationships"].(map[string]any)["territory"].(map[string]any)["data"].(map[string]any)
		if territory["id"] != "USA" {
			t.Fatalf("expected normalized territory id USA, got %#v", territory["id"])
		}

		body := `{"data":{"type":"subscriptionOfferCodes","id":"sub-offer-1","attributes":{"name":"Spring Promo","active":true}}}`
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
			"subscriptions", "offer-codes", "create",
			"--subscription-id", "sub-1",
			"--name", "Spring Promo",
			"--offer-eligibility", "replace_intro_offers",
			"--customer-eligibilities", "new,existing",
			"--offer-duration", "one_month",
			"--offer-mode", "free_trial",
			"--number-of-periods", "2",
			"--prices", "usa:pp-us",
			"--auto-renew-enabled", "true",
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
	if out.Data.ID != "sub-offer-1" {
		t.Fatalf("expected created offer code id sub-offer-1, got %q", out.Data.ID)
	}
}

func TestSubscriptionsOfferCodesCreateReturnsCreateFailure(t *testing.T) {
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
		if req.URL.Path != "/v1/subscriptionOfferCodes" {
			t.Fatalf("expected path /v1/subscriptionOfferCodes, got %s", req.URL.Path)
		}
		body := `{"errors":[{"status":"422","title":"Unprocessable Entity","detail":"invalid offer settings"}]}`
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offer-codes", "create",
			"--subscription-id", "sub-1",
			"--name", "Spring Promo",
			"--offer-eligibility", "replace_intro_offers",
			"--customer-eligibilities", "new",
			"--offer-duration", "one_month",
			"--offer-mode", "free_trial",
			"--number-of-periods", "1",
			"--prices", "usa:pp-us",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(runErr.Error(), "subscriptions offer-codes create: failed to create") {
		t.Fatalf("expected create failure, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
}
