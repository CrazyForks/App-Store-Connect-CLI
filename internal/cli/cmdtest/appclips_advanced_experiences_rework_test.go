package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	appclipscli "github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/appclips"
)

func TestAppClipsAdvancedExperiencesCreateSupportsInlineLocalization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/appClipAdvancedExperiences" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var payload asc.AppClipAdvancedExperienceCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Data.Relationships.AppClip.Data.ID != "clip-1" {
			t.Fatalf("expected app clip id clip-1, got %s", payload.Data.Relationships.AppClip.Data.ID)
		}
		if payload.Data.Relationships.HeaderImage.Data.ID != "img-1" {
			t.Fatalf("expected header image id img-1, got %s", payload.Data.Relationships.HeaderImage.Data.ID)
		}
		if len(payload.Data.Relationships.Localizations.Data) != 1 || payload.Data.Relationships.Localizations.Data[0].ID != "${localization-1}" {
			t.Fatalf("unexpected localization linkage: %#v", payload.Data.Relationships.Localizations.Data)
		}
		if len(payload.Included) != 1 || payload.Included[0].ID != payload.Data.Relationships.Localizations.Data[0].ID {
			t.Fatalf("included localization must use the relationship local ID: %#v", payload.Included)
		}
		included := payload.Included[0].Attributes
		if included.Language != asc.AppClipAdvancedExperienceLanguageEN || included.Title != "Order ahead" || included.Subtitle != "Ready when you arrive" {
			t.Fatalf("unexpected inline localization: %#v", included)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"type":"appClipAdvancedExperiences","id":"adv-1","attributes":{"link":"https://example.com"}},"links":{}}`))
	}))
	defer server.Close()
	newDefaultExperiencesCreateClient(t, server)

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-clips", "advanced-experiences", "create",
			"--app-clip-id", "clip-1",
			"--link", "https://example.com",
			"--default-language", "EN",
			"--is-powered-by",
			"--header-image-id", "img-1",
			"--language", "EN",
			"--title", "Order ahead",
			"--subtitle", "Ready when you arrive",
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
	var response asc.AppClipAdvancedExperienceResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout=%q", err, stdout)
	}
	if response.Data.ID != "adv-1" {
		t.Fatalf("expected advanced experience id adv-1, got %s", response.Data.ID)
	}
}

func TestAppClipsAdvancedExperienceImagesCreateSupportsUnattachedAndAttachedUploads(t *testing.T) {
	for _, test := range []struct {
		name             string
		experienceID     string
		wantExperienceID string
		wantAttach       bool
	}{
		{name: "unattached"},
		{name: "attached", experienceID: "adv-1", wantExperienceID: "adv-1", wantAttach: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			imageData := []byte("advanced-image")
			filePath := filepath.Join(t.TempDir(), "advanced.png")
			if err := os.WriteFile(filePath, imageData, 0o600); err != nil {
				t.Fatalf("write image: %v", err)
			}

			var mu sync.Mutex
			var requests []string
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				mu.Unlock()

				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/appClipAdvancedExperienceImages":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"data":{"type":"appClipAdvancedExperienceImages","id":"img-1","attributes":{"uploadOperations":[{"method":"PUT","url":%q,"length":%d,"offset":0}]}},"links":{}}`, server.URL+"/upload", len(imageData))
				case r.Method == http.MethodPut && r.URL.Path == "/upload":
					_, _ = io.Copy(io.Discard, r.Body)
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodPatch && r.URL.Path == "/v1/appClipAdvancedExperienceImages/img-1":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"data":{"type":"appClipAdvancedExperienceImages","id":"img-1","attributes":{"fileName":"advanced.png","fileSize":14,"assetDeliveryState":{"state":"COMPLETE"}}},"links":{}}`))
				case r.Method == http.MethodPatch && r.URL.Path == "/v1/appClipAdvancedExperiences/adv-1":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"data":{"type":"appClipAdvancedExperiences","id":"adv-1"},"links":{}}`))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()
			newDefaultExperiencesCreateClient(t, server)

			args := []string{"app-clips", "advanced-experiences", "images", "create", "--file", filePath}
			if test.experienceID != "" {
				args = append(args, "--experience-id", test.experienceID)
			}
			root := RootCommand("1.2.3")
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var result asc.AppClipAdvancedExperienceImageUploadResult
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("unmarshal stdout: %v\nstdout=%q", err, stdout)
			}
			if result.ID != "img-1" || result.ExperienceID != test.wantExperienceID || !result.Uploaded {
				t.Fatalf("unexpected upload result: %#v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			attached := false
			for _, request := range requests {
				attached = attached || request == "PATCH /v1/appClipAdvancedExperiences/adv-1"
			}
			if attached != test.wantAttach {
				t.Fatalf("attach request = %t, want %t; requests=%v", attached, test.wantAttach, requests)
			}
		})
	}
}

func TestAppClipsAdvancedExperienceImagesDeleteIsDeprecatedUnsupportedShim(t *testing.T) {
	clientFactoryCalls := 0
	restore := appclipscli.SetClientFactory(func() (*asc.Client, error) {
		clientFactoryCalls++
		return nil, errors.New("must not authenticate")
	})
	t.Cleanup(restore)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"app-clips", "advanced-experiences", "images", "delete", "--id", "img-1", "--confirm"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected usage failure, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected no success output, got %q", stdout)
	}
	for _, want := range []string{"DEPRECATED", "does not support deleting", "images create --file", "advanced-experiences update --experience-id", "--header-image-id"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want %q", stderr, want)
		}
	}
	if clientFactoryCalls != 0 {
		t.Fatalf("expected no authentication or HTTP setup, got %d client factory calls", clientFactoryCalls)
	}

	cmd := appclipscli.AppClipAdvancedExperienceImagesDeleteCommand()
	if !strings.HasPrefix(cmd.ShortHelp, "DEPRECATED:") {
		t.Fatalf("ShortHelp = %q, want deprecation marker", cmd.ShortHelp)
	}
	if !strings.Contains(cmd.LongHelp, "images create --file") || !strings.Contains(cmd.LongHelp, "--header-image-id") {
		t.Fatalf("LongHelp lacks migration guidance: %q", cmd.LongHelp)
	}
}
