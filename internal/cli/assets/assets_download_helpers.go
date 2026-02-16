package assets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func sanitizeBaseFileName(value string) string {
	base := strings.TrimSpace(value)
	if base == "" {
		return ""
	}

	// Defensive: ensure we never write outside the target directory.
	base = filepath.Base(base)
	base = strings.TrimSpace(base)

	if base == "" || base == "." || base == ".." {
		return ""
	}

	// Extra defense: normalize separators across platforms.
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.TrimSpace(base)

	if base == "" || base == "." || base == ".." {
		return ""
	}
	return base
}

func resolveImageAssetDownloadURL(asset *asc.ImageAsset, fileName string) (string, error) {
	if asset == nil {
		return "", fmt.Errorf("image asset is missing")
	}

	template := strings.TrimSpace(asset.TemplateURL)
	if template == "" {
		return "", fmt.Errorf("image asset template URL is missing")
	}
	if asset.Width <= 0 || asset.Height <= 0 {
		return "", fmt.Errorf("image asset dimensions are missing")
	}

	resolved := template
	resolved = strings.ReplaceAll(resolved, "{w}", fmt.Sprintf("%d", asset.Width))
	resolved = strings.ReplaceAll(resolved, "{h}", fmt.Sprintf("%d", asset.Height))
	if strings.Contains(resolved, "{f}") {
		// ASC imageAsset.templateUrl often includes "{f}" for file format.
		// Prefer the extension from the asset filename when available; fall back to png.
		format := ""
		ext := strings.TrimSpace(filepath.Ext(strings.TrimSpace(fileName)))
		if ext != "" {
			format = strings.TrimPrefix(ext, ".")
		}
		if strings.TrimSpace(format) == "" {
			format = "png"
		}
		resolved = strings.ReplaceAll(resolved, "{f}", format)
	}

	// If the URL still contains template braces, it is likely not usable as-is.
	if strings.Contains(resolved, "{") || strings.Contains(resolved, "}") {
		return "", fmt.Errorf("unresolved template URL: %q", template)
	}

	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("parse resolved URL: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		// ok
	default:
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}

	return resolved, nil
}

func downloadURLToFile(ctx context.Context, rawURL string, outputPath string, overwrite bool) (int64, string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return 0, "", fmt.Errorf("download URL is required")
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return 0, "", fmt.Errorf("output path is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			msg = strings.Join(strings.Fields(msg), " ")
		}
		if msg == "" {
			msg = strings.TrimSpace(resp.Status)
		}
		return 0, contentType, fmt.Errorf("unexpected status %d (%s)", resp.StatusCode, msg)
	}

	n, err := writeDownloadedFile(outputPath, resp.Body, overwrite)
	return n, contentType, err
}

func writeDownloadedFile(path string, reader io.Reader, overwrite bool) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}

	if !overwrite {
		file, err := shared.OpenNewFileNoFollow(path, 0o600)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return 0, fmt.Errorf("output file already exists: %w", err)
			}
			return 0, err
		}
		defer file.Close()

		written, err := io.Copy(file, reader)
		if err != nil {
			return 0, err
		}
		return written, file.Sync()
	}

	// Best-effort protection: refuse overwriting symlinks; use temp+rename.
	// Important: do not remove the destination until the new file is fully written.
	return shared.WriteFileNoSymlinkOverwrite(path, reader, 0o600, ".asc-download-*", ".asc-download-backup-*")
}
