package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	rootcmd "github.com/rudrankriyam/App-Store-Connect-CLI/cmd"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

func TestSubscriptionsIntroductoryOffersViewFindsOfferAcrossPages(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	const subscriptionID = "1234567890"
	nextURL := "https://api.appstoreconnect.apple.com/v1/subscriptions/" + subscriptionID + "/introductoryOffers?cursor=next"
	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/subscriptions/"+subscriptionID+"/introductoryOffers" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		switch requestCount {
		case 1:
			if got := req.URL.Query().Get("limit"); got != "200" {
				t.Fatalf("expected first-page limit 200, got %q", got)
			}
			body := `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-1"}],"links":{"next":"` + nextURL + `"}}`
			return jsonHTTPResponse(http.StatusOK, body), nil
		case 2:
			if got := req.URL.Query().Get("cursor"); got != "next" {
				t.Fatalf("expected server-provided next URL, got %q", req.URL.String())
			}
			body := `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-2","attributes":{"duration":"ONE_MONTH","offerMode":"FREE_TRIAL","numberOfPeriods":1}}],"links":{"next":""}}`
			return jsonHTTPResponse(http.StatusOK, body), nil
		default:
			t.Fatalf("unexpected request %d: %s", requestCount, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offers", "introductory", "view",
			"--subscription-id", subscriptionID,
			"--id", "offer-2",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if requestCount != 2 {
		t.Fatalf("expected two requests, got %d", requestCount)
	}

	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&response); err != nil {
		t.Fatalf("decode output: %v\nstdout: %s", err, stdout)
	}
	if response.Data.ID != "offer-2" {
		t.Fatalf("expected offer-2, got %q", response.Data.ID)
	}
}

func TestSubscriptionsIntroductoryOffersViewResolvesSubscriptionSelector(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/apps/app-1/subscriptionGroups":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"subscriptionGroups","id":"group-1","attributes":{"referenceName":"Premium"}}],"links":{"next":""}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/subscriptionGroups/group-1/subscriptions":
			if got := req.URL.Query().Get("filter[productId]"); got != "com.example.monthly" {
				t.Fatalf("expected product ID filter, got %q", got)
			}
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"subscriptions","id":"sub-1","attributes":{"name":"Monthly","productId":"com.example.monthly"}}],"links":{"next":""}}`), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/subscriptions/sub-1/introductoryOffers":
			return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-1"}],"links":{"next":""}}`), nil
		default:
			t.Fatalf("unexpected request: %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offers", "introductory", "view",
			"--app", "app-1",
			"--subscription-id", "com.example.monthly",
			"--id", "offer-1",
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
	if requestCount != 3 {
		t.Fatalf("expected three requests, got %d", requestCount)
	}
	if !strings.Contains(stdout, `"id":"offer-1"`) {
		t.Fatalf("expected selected offer output, got %q", stdout)
	}
}

func TestSubscriptionsIntroductoryOffersViewReturnsNotFoundAfterAllPages(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/subscriptions/1234567890/introductoryOffers" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return jsonHTTPResponse(http.StatusOK, `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-other"}],"links":{"next":""}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"subscriptions", "offers", "introductory", "view",
			"--subscription-id", "1234567890",
			"--id", "offer-missing",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected not-found error")
	}
	if !errors.Is(runErr, asc.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", runErr)
	}
	if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitNotFound {
		t.Fatalf("expected exit code %d, got %d", rootcmd.ExitNotFound, got)
	}
	if !strings.Contains(runErr.Error(), `introductory offer "offer-missing" not found for subscription "1234567890"`) {
		t.Fatalf("expected contextual not-found error, got %v", runErr)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("expected no process output before error reporting, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestSubscriptionsIntroductoryOffersViewRejectsRepeatedPaginationURL(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	const subscriptionID = "1234567890"
	nextURL := "https://api.appstoreconnect.apple.com/v1/subscriptions/" + subscriptionID + "/introductoryOffers?cursor=repeated"
	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		if requestCount > 2 {
			return nil, errors.New("unexpected third request")
		}
		body := `{"data":[{"type":"subscriptionIntroductoryOffers","id":"offer-other"}],"links":{"next":"` + nextURL + `"}}`
		return jsonHTTPResponse(http.StatusOK, body), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"subscriptions", "offers", "introductory", "view",
		"--subscription-id", subscriptionID,
		"--id", "offer-missing",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	runErr := root.Run(context.Background())
	if !errors.Is(runErr, asc.ErrRepeatedPaginationURL) {
		t.Fatalf("expected ErrRepeatedPaginationURL, got %v", runErr)
	}
	if requestCount != 2 {
		t.Fatalf("expected two requests before repeated URL detection, got %d", requestCount)
	}
}
