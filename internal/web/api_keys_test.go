package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientCreateAPIKeySendsTeamKeyPayload(t *testing.T) {
	var requestBody map[string]any
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/iris/v1/apiKeys" {
			t.Fatalf("expected /iris/v1/apiKeys, got %s", r.URL.Path)
		}
		if r.Header.Get("X-CSRF-ITC") != "[asc-ui]" {
			t.Fatalf("expected integrations CSRF header, got %q", r.Header.Get("X-CSRF-ITC"))
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
					"data": {
						"type": "apiKeys",
						"id": "ABC123XYZ",
						"attributes": {
							"nickname": "Release automation",
							"roles": ["APP_MANAGER"],
							"allAppsVisible": true,
							"canDownload": true,
							"isActive": true,
							"keyType": "PUBLIC_API"
						}
					}
				}`))
	}))

	key, err := client.CreateAPIKey(context.Background(), APIKeyCreateAttributes{
		Nickname: "Release automation",
		Role:     "APP_MANAGER",
	})
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}
	if key.KeyID != "ABC123XYZ" || key.Name != "Release automation" {
		t.Fatalf("unexpected key: %#v", key)
	}

	data, ok := requestBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON:API data object, got %#v", requestBody)
	}
	if data["type"] != "apiKeys" {
		t.Fatalf("expected apiKeys type, got %#v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes object, got %#v", data)
	}
	if attrs["nickname"] != "Release automation" || attrs["keyType"] != "PUBLIC_API" || attrs["allAppsVisible"] != true {
		t.Fatalf("unexpected attributes: %#v", attrs)
	}
	roles, ok := attrs["roles"].([]any)
	if !ok || len(roles) != 1 || roles[0] != "APP_MANAGER" {
		t.Fatalf("unexpected roles: %#v", attrs["roles"])
	}
}

func TestClientDownloadAPIKeyDecodesP8(t *testing.T) {
	p8 := "-----BEGIN PRIVATE KEY-----\nfixture\n-----END PRIVATE KEY-----\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(p8))
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/iris/v1/apiKeys/ABC123XYZ" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("fields[apiKeys]") != "privateKey" {
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"apiKeys","id":"ABC123XYZ","attributes":{"privateKey":"` + encoded + `"}}}`))
	}))

	got, err := client.DownloadAPIKey(context.Background(), "ABC123XYZ")
	if err != nil {
		t.Fatalf("DownloadAPIKey() error: %v", err)
	}
	if string(got) != p8 {
		t.Fatalf("unexpected decoded P8: %q", string(got))
	}
}

func TestClientDownloadAPIKeyRejectsNonPEMPayload(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("not a key"))
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"attributes":{"privateKey":"` + encoded + `"}}}`))
	}))

	if _, err := client.DownloadAPIKey(context.Background(), "ABC123XYZ"); err == nil {
		t.Fatal("expected invalid P8 error")
	}
}

func TestClientGetAPIKeyParsesIssuerID(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("include") != "provider" {
			t.Fatalf("expected provider include, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
					"data": {
						"type": "apiKeys",
						"id": "ABC123XYZ",
						"attributes": {"nickname":"Release automation","roles":["ADMIN"],"allAppsVisible":true,"isActive":true,"keyType":"PUBLIC_API"},
						"relationships": {"provider":{"data":{"type":"contentProviders","id":"69a6de00-aaaa-bbbb-cccc-123456789abc"}}}
					}
				}`))
	}))

	key, err := client.GetAPIKey(context.Background(), "ABC123XYZ")
	if err != nil {
		t.Fatalf("GetAPIKey() error: %v", err)
	}
	if key.IssuerID != "69a6de00-aaaa-bbbb-cccc-123456789abc" {
		t.Fatalf("unexpected issuer ID %q", key.IssuerID)
	}
}

func TestIsAPIKeyDownloadRetryable(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusConflict, http.StatusTooManyRequests, http.StatusBadGateway} {
		if !IsAPIKeyDownloadRetryable(&APIError{Status: status}) {
			t.Fatalf("expected status %d to be retryable", status)
		}
	}
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		if IsAPIKeyDownloadRetryable(&APIError{Status: status}) {
			t.Fatalf("expected status %d not to be retryable", status)
		}
	}
}

type apiKeyRewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t apiKeyRewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	requestURL := *request.URL
	requestURL.Scheme = t.target.Scheme
	requestURL.Host = t.target.Host
	clone.URL = &requestURL
	return t.base.RoundTrip(clone)
}

func newAPIKeyHTTPTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	return &Client{
		httpClient: &http.Client{Transport: apiKeyRewriteTransport{
			target: target,
			base:   server.Client().Transport,
		}},
	}
}
