package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCIUsageSummaryParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/team-uuid/usage/summary" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"plan": {
				"name": "Plan",
				"available": 1467,
				"used": 33,
				"total": 1500,
				"reset_date": "2026-03-16",
				"reset_date_time": "2026-03-16T09:43:54Z"
			},
			"links": {
				"manage": "https://developer.apple.com/xcode-cloud/"
			}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	result, err := client.GetCIUsageSummary(context.Background(), "team-uuid")
	if err != nil {
		t.Fatalf("GetCIUsageSummary() error = %v", err)
	}
	if result.Plan.Name != "Plan" {
		t.Fatalf("expected plan name %q, got %q", "Plan", result.Plan.Name)
	}
	if result.Plan.Available != 1467 {
		t.Fatalf("expected available 1467, got %d", result.Plan.Available)
	}
	if result.Plan.Used != 33 {
		t.Fatalf("expected used 33, got %d", result.Plan.Used)
	}
	if result.Plan.Total != 1500 {
		t.Fatalf("expected total 1500, got %d", result.Plan.Total)
	}
	if result.Plan.ResetDate != "2026-03-16" {
		t.Fatalf("expected reset_date %q, got %q", "2026-03-16", result.Plan.ResetDate)
	}
	if result.Plan.ResetDateTime != "2026-03-16T09:43:54Z" {
		t.Fatalf("expected reset_date_time %q, got %q", "2026-03-16T09:43:54Z", result.Plan.ResetDateTime)
	}
	if result.Links["manage"] != "https://developer.apple.com/xcode-cloud/" {
		t.Fatalf("expected manage link, got %v", result.Links)
	}
}

func TestGetCIUsageSummaryRejectsEmptyTeamID(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	_, err := client.GetCIUsageSummary(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty team ID")
	}
	if !strings.Contains(err.Error(), "team id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCIUsageMonthsQueryParams(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"usage":[],"product_usage":[],"info":{}}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	_, err := client.GetCIUsageMonths(context.Background(), "team-uuid", 1, 2025, 12, 2025)
	if err != nil {
		t.Fatalf("GetCIUsageMonths() error = %v", err)
	}
	for _, param := range []string{"start_month=1", "start_year=2025", "end_month=12", "end_year=2025"} {
		if !strings.Contains(gotQuery, param) {
			t.Fatalf("expected query to contain %q, got %q", param, gotQuery)
		}
	}
}

func TestGetCIUsageMonthsParsesProductUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"usage": [{"month":1,"year":2026,"duration":120}],
			"product_usage": [
				{
					"product_id": "prod-1",
					"product_name": "My App",
					"bundle_id": "com.example.app",
					"usage": [{"month":1,"year":2026,"duration":120}]
				}
			],
			"info": {"start_month":1,"start_year":2026,"end_month":1,"end_year":2026}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	result, err := client.GetCIUsageMonths(context.Background(), "team-uuid", 1, 2026, 1, 2026)
	if err != nil {
		t.Fatalf("GetCIUsageMonths() error = %v", err)
	}
	if len(result.Usage) != 1 || result.Usage[0].Duration != 120 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
	if len(result.ProductUsage) != 1 {
		t.Fatalf("expected 1 product usage, got %d", len(result.ProductUsage))
	}
	pu := result.ProductUsage[0]
	if pu.ProductID != "prod-1" || pu.ProductName != "My App" || pu.BundleID != "com.example.app" {
		t.Fatalf("unexpected product usage: %+v", pu)
	}
}

func TestGetCIUsageMonthsRejectsEmptyTeamID(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	_, err := client.GetCIUsageMonths(context.Background(), "  ", 1, 2026, 1, 2026)
	if err == nil {
		t.Fatal("expected error for empty team ID")
	}
	if !strings.Contains(err.Error(), "team id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCIUsageDaysParsesWorkflowUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/products/prod-1/usage/days") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("start") != "2026-01-01" || r.URL.Query().Get("end") != "2026-01-31" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"usage": [{"date":"2026-01-01","duration":60}],
			"workflow_usage": [
				{
					"workflow_id": "wf-1",
					"workflow_name": "CI Workflow",
					"usage": [{"date":"2026-01-01","duration":60}]
				}
			],
			"info": {}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	result, err := client.GetCIUsageDays(context.Background(), "team-uuid", "prod-1", "2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatalf("GetCIUsageDays() error = %v", err)
	}
	if len(result.Usage) != 1 || result.Usage[0].Duration != 60 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
	if len(result.WorkflowUsage) != 1 {
		t.Fatalf("expected 1 workflow usage, got %d", len(result.WorkflowUsage))
	}
	wf := result.WorkflowUsage[0]
	if wf.WorkflowID != "wf-1" || wf.WorkflowName != "CI Workflow" {
		t.Fatalf("unexpected workflow usage: %+v", wf)
	}
}

func TestGetCIUsageDaysRejectsEmptyInputs(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	tests := []struct {
		name      string
		teamID    string
		productID string
		start     string
		end       string
		wantErr   string
	}{
		{"empty team", "", "prod", "2026-01-01", "2026-01-31", "team id is required"},
		{"empty product", "team", "", "2026-01-01", "2026-01-31", "product id is required"},
		{"empty start", "team", "prod", "", "2026-01-31", "start date is required"},
		{"empty end", "team", "prod", "2026-01-01", "", "end date is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetCIUsageDays(context.Background(), tt.teamID, tt.productID, tt.start, tt.end)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestListCIProductsParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/products-v4") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Fatalf("expected limit=100, got %q", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"id":"prod-1","name":"My App","bundle_id":"com.example.app","type":"solo"},
				{"id":"prod-2","name":"Other App","bundle_id":"com.other.app","type":"solo","icon_url":"https://example.com/icon.png"}
			]
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	result, err := client.ListCIProducts(context.Background(), "team-uuid")
	if err != nil {
		t.Fatalf("ListCIProducts() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 products, got %d", len(result.Items))
	}
	if result.Items[0].ID != "prod-1" || result.Items[0].BundleID != "com.example.app" {
		t.Fatalf("unexpected first product: %+v", result.Items[0])
	}
	if result.Items[1].IconURL != "https://example.com/icon.png" {
		t.Fatalf("expected icon_url, got %q", result.Items[1].IconURL)
	}
}

func TestListCIProductsRejectsEmptyTeamID(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	_, err := client.ListCIProducts(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty team ID")
	}
	if !strings.Contains(err.Error(), "team id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCIUsageSummaryHandles4xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	_, err := client.GetCIUsageSummary(context.Background(), "team-uuid")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	var apiErr *APIError
	if ok := errorAs(err, &apiErr); !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.Status != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", apiErr.Status)
	}
}

func TestCIUsagePlanJSONRoundTrip(t *testing.T) {
	raw := `{"name":"Plan","reset_date":"2026-03-16","reset_date_time":"2026-03-16T09:43:54Z","available":1467,"used":33,"total":1500}`
	var plan CIUsagePlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if plan.ResetDate != "2026-03-16" {
		t.Fatalf("expected reset_date %q, got %q", "2026-03-16", plan.ResetDate)
	}
	if plan.ResetDateTime != "2026-03-16T09:43:54Z" {
		t.Fatalf("expected reset_date_time %q, got %q", "2026-03-16T09:43:54Z", plan.ResetDateTime)
	}

	out, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if !strings.Contains(string(out), `"reset_date":"2026-03-16"`) {
		t.Fatalf("expected reset_date in output, got %s", out)
	}
}

func TestNewCIClientSetsBaseURL(t *testing.T) {
	session := &AuthSession{Client: http.DefaultClient}
	client := NewCIClient(session)
	if !strings.HasSuffix(client.baseURL, "/ci/api") {
		t.Fatalf("expected base URL ending in /ci/api, got %q", client.baseURL)
	}
}

// errorAs is a test helper that wraps errors.As without importing errors
// (already available via the APIError assertion pattern in this package).
func errorAs(err error, target any) bool {
	type unwrapper interface{ Unwrap() error }
	switch v := target.(type) {
	case **APIError:
		if e, ok := err.(*APIError); ok {
			*v = e
			return true
		}
		if u, ok := err.(unwrapper); ok {
			return errorAs(u.Unwrap(), target)
		}
	}
	return false
}
