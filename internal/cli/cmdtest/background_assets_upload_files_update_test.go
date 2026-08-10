package cmdtest

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

func TestBackgroundAssetsUploadFilesUpdatePreservesValidRequests(t *testing.T) {
	for _, test := range []struct {
		name         string
		withChecksum bool
	}{
		{name: "uploaded only"},
		{name: "paired file and checksum", withChecksum: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupAuth(t)
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

			args := []string{
				"background-assets", "upload-files", "update",
				"--upload-file-id", "UPLOAD_ID",
				"--uploaded", "true",
				"--output", "json",
			}
			fileContents := []byte("background asset")
			checksum := md5.Sum(fileContents)
			expectedHash := hex.EncodeToString(checksum[:])
			if test.withChecksum {
				filePath := filepath.Join(t.TempDir(), "asset.zip")
				if err := os.WriteFile(filePath, fileContents, 0o600); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
				args = append(args, "--file", filePath, "--checksum")
			}

			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requestCount++
				w.Header().Set("Content-Type", "application/json")

				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v1/backgroundAssetUploadFiles/UPLOAD_ID" && test.withChecksum:
					_, _ = fmt.Fprintf(w, `{"data":{"type":"backgroundAssetUploadFiles","id":"UPLOAD_ID","attributes":{"sourceFileChecksums":{"file":{"hash":%q,"algorithm":"MD5"}}}}}`, expectedHash)
				case req.Method == http.MethodPatch && req.URL.Path == "/v1/backgroundAssetUploadFiles/UPLOAD_ID":
					var payload asc.BackgroundAssetUploadFileUpdateRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Errorf("decode request: %v", err)
						http.Error(w, "invalid request", http.StatusBadRequest)
						return
					}
					assertBackgroundAssetUploadFileUpdatePayload(t, payload, test.withChecksum, expectedHash)
					_, _ = io.WriteString(w, `{"data":{"type":"backgroundAssetUploadFiles","id":"UPLOAD_ID","attributes":{"uploaded":true}}}`)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
					http.Error(w, "unexpected request", http.StatusBadRequest)
				}
			}))
			t.Cleanup(server.Close)
			installDefaultTransportForServer(t, server)

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
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			wantRequests := 1
			if test.withChecksum {
				wantRequests = 2
			}
			if requestCount != wantRequests {
				t.Fatalf("request count = %d, want %d", requestCount, wantRequests)
			}
			assertBackgroundAssetUploadFileID(t, stdout, "UPLOAD_ID")
		})
	}
}

func assertBackgroundAssetUploadFileUpdatePayload(t *testing.T, payload asc.BackgroundAssetUploadFileUpdateRequest, wantChecksum bool, expectedHash string) {
	t.Helper()

	attrs := payload.Data.Attributes
	if attrs == nil || attrs.Uploaded == nil || !*attrs.Uploaded {
		t.Errorf("uploaded attribute = %#v, want true", attrs)
		return
	}
	if !wantChecksum {
		if attrs.SourceFileChecksums != nil {
			t.Errorf("source file checksums = %#v, want omitted", attrs.SourceFileChecksums)
		}
		return
	}
	if attrs.SourceFileChecksums == nil || attrs.SourceFileChecksums.File == nil {
		t.Errorf("source file checksums = %#v, want MD5 file checksum", attrs.SourceFileChecksums)
		return
	}
	if got := attrs.SourceFileChecksums.File; got.Hash != expectedHash || got.Algorithm != asc.ChecksumAlgorithmMD5 {
		t.Errorf("file checksum = %#v, want hash %q and MD5", got, expectedHash)
	}
}

func assertBackgroundAssetUploadFileID(t *testing.T, output, want string) {
	t.Helper()

	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode output: %v\noutput: %s", err, output)
	}
	if response.Data.ID != want {
		t.Fatalf("output ID = %q, want %q", response.Data.ID, want)
	}
}
