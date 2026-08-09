package routingcoverage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// PreparedRoutingCoverageFile is a validated routing coverage upload source.
type PreparedRoutingCoverageFile struct {
	Path     string
	FileName string
	FileSize int64
	Checksum string
}

// PrepareRoutingCoverageFile validates and fingerprints a routing coverage file.
func PrepareRoutingCoverageFile(path string) (PreparedRoutingCoverageFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return PreparedRoutingCoverageFile{}, fmt.Errorf("file is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return PreparedRoutingCoverageFile{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return PreparedRoutingCoverageFile{}, fmt.Errorf("refusing to read symlink %q", path)
	}
	if !info.Mode().IsRegular() {
		return PreparedRoutingCoverageFile{}, fmt.Errorf("expected regular file: %q", path)
	}
	if info.Size() <= 0 {
		return PreparedRoutingCoverageFile{}, fmt.Errorf("file size must be greater than 0")
	}

	checksum, err := asc.ComputeFileChecksum(path, asc.ChecksumAlgorithmMD5)
	if err != nil {
		return PreparedRoutingCoverageFile{}, fmt.Errorf("checksum failed: %w", err)
	}
	return PreparedRoutingCoverageFile{
		Path:     filepath.Clean(path),
		FileName: filepath.Base(path),
		FileSize: info.Size(),
		Checksum: checksum.Hash,
	}, nil
}

// UploadPreparedRoutingCoverageFile creates, uploads, and commits routing coverage.
func UploadPreparedRoutingCoverageFile(ctx context.Context, client *asc.Client, versionID string, file PreparedRoutingCoverageFile) (*asc.RoutingAppCoverageResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("client is required")
	}

	requestCtx, cancel := shared.ContextWithTimeout(ctx)
	response, err := client.CreateRoutingAppCoverage(requestCtx, versionID, file.FileName, file.FileSize)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to create: %w", err)
	}
	if response == nil || len(response.Data.Attributes.UploadOperations) == 0 {
		return nil, fmt.Errorf("no upload operations returned")
	}
	if strings.TrimSpace(response.Data.ID) == "" {
		return nil, fmt.Errorf("created routing coverage response is missing an ID")
	}

	uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx)
	err = asc.ExecuteUploadOperations(uploadCtx, file.Path, response.Data.Attributes.UploadOperations)
	uploadCancel()
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	uploadedChecksum, err := asc.ComputeFileChecksum(file.Path, asc.ChecksumAlgorithmMD5)
	if err != nil {
		return nil, fmt.Errorf("checksum failed: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(uploadedChecksum.Hash), strings.TrimSpace(file.Checksum)) {
		return nil, fmt.Errorf("file changed during upload: %q", file.Path)
	}

	uploaded := true
	attributes := asc.RoutingAppCoverageUpdateAttributes{
		SourceFileChecksum: &file.Checksum,
		Uploaded:           &uploaded,
	}
	commitCtx, commitCancel := shared.ContextWithUploadTimeout(ctx)
	committed, err := client.UpdateRoutingAppCoverage(commitCtx, response.Data.ID, attributes)
	commitCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to commit upload: %w", err)
	}
	if committed == nil || strings.TrimSpace(committed.Data.ID) == "" {
		return nil, fmt.Errorf("committed routing coverage response is missing an ID")
	}
	return committed, nil
}
