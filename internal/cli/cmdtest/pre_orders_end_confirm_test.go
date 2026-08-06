package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestPreOrdersEndWithConfirmPostsExpectedPayload(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/endAppAvailabilityPreOrders" {
			t.Fatalf("expected /v1/endAppAvailabilityPreOrders, got %s", req.URL.Path)
		}

		var payload struct {
			Data struct {
				Type          string `json:"type"`
				Relationships struct {
					TerritoryAvailabilities struct {
						Data []struct {
							Type string `json:"type"`
							ID   string `json:"id"`
						} `json:"data"`
					} `json:"territoryAvailabilities"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Data.Type != "endAppAvailabilityPreOrders" {
			t.Fatalf("unexpected data type %q", payload.Data.Type)
		}
		linkages := payload.Data.Relationships.TerritoryAvailabilities.Data
		if len(linkages) != 2 || linkages[0].Type != "territoryAvailabilities" || linkages[0].ID != "ta-1" || linkages[1].ID != "ta-2" {
			t.Fatalf("unexpected territory availability linkages: %+v", linkages)
		}

		return jsonHTTPResponse(http.StatusCreated, `{"data":{"type":"endAppAvailabilityPreOrders","id":"end-1"},"links":{"self":"https://api.appstoreconnect.apple.com/v1/endAppAvailabilityPreOrders/end-1"}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{
		"pre-orders", "end",
		"--territory-availability", "ta-1,ta-2",
		"--confirm",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if requestCount != 1 {
		t.Fatalf("expected one request, got %d", requestCount)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var output struct {
		Data struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode stdout: %v; stdout=%q", err, stdout)
	}
	if output.Data.Type != "endAppAvailabilityPreOrders" || output.Data.ID != "end-1" {
		t.Fatalf("unexpected output: %+v", output.Data)
	}
}
