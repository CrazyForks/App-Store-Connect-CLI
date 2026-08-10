package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestAssetListCommandsRenewRequestTimeout(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		setsPath    string
		assetsPath  string
		setsBody    string
		assetsBody  string
		assetID     string
		assetsField string
	}{
		{
			name:        "screenshots",
			command:     "screenshots",
			setsPath:    "/v1/appStoreVersionLocalizations/loc-1/appScreenshotSets",
			assetsPath:  "/v1/appScreenshotSets/set-1/appScreenshots",
			setsBody:    `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}]}`,
			assetsBody:  `{"data":[{"type":"appScreenshots","id":"shot-1","attributes":{"fileName":"home.png","fileSize":123}}]}`,
			assetID:     "shot-1",
			assetsField: "screenshots",
		},
		{
			name:        "video previews",
			command:     "video-previews",
			setsPath:    "/v1/appStoreVersionLocalizations/loc-1/appPreviewSets",
			assetsPath:  "/v1/appPreviewSets/set-1/appPreviews",
			setsBody:    `{"data":[{"type":"appPreviewSets","id":"set-1","attributes":{"previewType":"IPHONE_65"}}]}`,
			assetsBody:  `{"data":[{"type":"appPreviews","id":"preview-1","attributes":{"fileName":"preview.mov","fileSize":456}}]}`,
			assetID:     "preview-1",
			assetsField: "previews",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_TIMEOUT", "500ms")
			t.Setenv("ASC_TIMEOUT_SECONDS", "")
			t.Setenv("ASC_MAX_RETRIES", "0")

			originalTransport := http.DefaultTransport
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
			})

			const requestDelay = 300 * time.Millisecond
			callCount := 0
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				callCount++
				var body string
				switch callCount {
				case 1:
					if req.Method != http.MethodGet || req.URL.Path != tt.setsPath {
						t.Fatalf("unexpected sets request: %s %s", req.Method, req.URL.Path)
					}
					body = tt.setsBody
				case 2:
					if req.Method != http.MethodGet || req.URL.Path != tt.assetsPath {
						t.Fatalf("unexpected assets request: %s %s", req.Method, req.URL.Path)
					}
					body = tt.assetsBody
				default:
					t.Fatalf("unexpected request count %d", callCount)
				}

				select {
				case <-time.After(requestDelay):
					return jsonHTTPResponse(http.StatusOK, body), nil
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
			})

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse([]string{tt.command, "list", "--version-localization", "loc-1"}); err != nil {
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
			if callCount != 2 {
				t.Fatalf("request count = %d, want 2", callCount)
			}

			var result struct {
				VersionLocalizationID string `json:"versionLocalizationId"`
				Sets                  []struct {
					Set struct {
						ID string `json:"id"`
					} `json:"set"`
					Screenshots []struct {
						ID string `json:"id"`
					} `json:"screenshots"`
					Previews []struct {
						ID string `json:"id"`
					} `json:"previews"`
				} `json:"sets"`
			}
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("decode output: %v (stdout=%q)", err, stdout)
			}
			if result.VersionLocalizationID != "loc-1" || len(result.Sets) != 1 || result.Sets[0].Set.ID != "set-1" {
				t.Fatalf("unexpected aggregate output: %+v", result)
			}

			var gotAssetID string
			switch tt.assetsField {
			case "screenshots":
				if len(result.Sets[0].Screenshots) == 1 {
					gotAssetID = result.Sets[0].Screenshots[0].ID
				}
			case "previews":
				if len(result.Sets[0].Previews) == 1 {
					gotAssetID = result.Sets[0].Previews[0].ID
				}
			}
			if gotAssetID != tt.assetID {
				t.Fatalf("asset ID = %q, want %q", gotAssetID, tt.assetID)
			}
		})
	}
}
