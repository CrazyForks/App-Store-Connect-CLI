package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeCreateAttrsDefaults(t *testing.T) {
	attrs, err := normalizeCreateAttrs(AppCreateAttributes{
		Name:     "My App",
		BundleID: "com.example.app",
		SKU:      "SKU123",
	})
	if err != nil {
		t.Fatalf("normalizeCreateAttrs error: %v", err)
	}
	if attrs.PrimaryLocale != defaultPrimaryLocale {
		t.Fatalf("expected default locale %q, got %q", defaultPrimaryLocale, attrs.PrimaryLocale)
	}
	if attrs.Platform != defaultPlatform {
		t.Fatalf("expected default platform %q, got %q", defaultPlatform, attrs.Platform)
	}
	if attrs.VersionString != defaultVersion {
		t.Fatalf("expected default version %q, got %q", defaultVersion, attrs.VersionString)
	}
}

func TestNormalizeCreateAttrsRejectsInvalidPlatform(t *testing.T) {
	_, err := normalizeCreateAttrs(AppCreateAttributes{
		Name:     "My App",
		BundleID: "com.example.app",
		SKU:      "SKU123",
		Platform: "WATCH_OS",
	})
	if err == nil {
		t.Fatal("expected invalid platform error")
	}
}

func TestBuildAppCreateRequestUsesLocalizationForName(t *testing.T) {
	req := buildAppCreateRequest(AppCreateAttributes{
		Name:          "My App",
		BundleID:      "com.example.app",
		SKU:           "SKU123",
		PrimaryLocale: "en-US",
		Platform:      "IOS",
		VersionString: "1.0",
	})

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	payload := string(raw)

	if strings.Contains(payload, `"attributes":{"name":"My App","sku"`) {
		t.Fatalf("expected name not to be part of top-level app attributes, payload=%s", payload)
	}
	if !strings.Contains(payload, `"appInfoLocalizations"`) {
		t.Fatalf("expected appInfoLocalization relationship, payload=%s", payload)
	}
	if !strings.Contains(payload, `"name":"My App"`) {
		t.Fatalf("expected localized app name in payload, payload=%s", payload)
	}
}

func TestFindAppEscapesBundleIDQuery(t *testing.T) {
	var gotRawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}
	_, err := client.FindApp(context.Background(), "com.example/app?x=1")
	if err != nil {
		t.Fatalf("FindApp error: %v", err)
	}
	if strings.Contains(gotRawQuery, "com.example/app?x=1") {
		t.Fatalf("expected escaped query value, got raw query %q", gotRawQuery)
	}
	if !strings.Contains(gotRawQuery, "filter%5BbundleId%5D=") && !strings.Contains(gotRawQuery, "filter[bundleId]=") {
		t.Fatalf("expected bundleId filter query, got %q", gotRawQuery)
	}
}

func TestDeleteAppSendsRemovedPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"apps","id":"1234567890","attributes":{"name":"Throwaway","bundleId":"com.example.throwaway","removed":true}}}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}
	app, err := client.DeleteApp(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("DeleteApp error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/apps/1234567890" {
		t.Fatalf("expected /apps/1234567890, got %s", gotPath)
	}
	for _, want := range []string{
		`"type":"apps"`,
		`"id":"1234567890"`,
		`"removed":true`,
	} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("expected request body to contain %s, got %s", want, gotBody)
		}
	}
	if app == nil || app.Data.ID != "1234567890" {
		t.Fatalf("expected app response id, got %+v", app)
	}
}

func TestDeleteAppRequiresID(t *testing.T) {
	client := &Client{}
	if _, err := client.DeleteApp(context.Background(), "  "); err == nil {
		t.Fatal("expected missing app id error")
	}
}
