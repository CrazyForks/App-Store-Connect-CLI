package shared

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SafeWriteFileNoSymlink writes a file to path without following symlinks and with an optional
// overwrite mode that preserves the original destination until the new file is fully written.
//
// When overwrite is false, the destination must not already exist.
// When overwrite is true, we refuse to overwrite symlinks and we use temp+rename; if rename fails
// because the destination exists (notably on Windows), we fall back to a safe replace that uses a
// backup file to preserve the original if the final move fails.
func SafeWriteFileNoSymlink(path string, perm os.FileMode, overwrite bool, tempPattern string, backupPattern string, write func(*os.File) (int64, error)) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}

	if !overwrite {
		file, err := OpenNewFileNoFollow(path, perm)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return 0, fmt.Errorf("output file already exists: %w", err)
			}
			return 0, err
		}
		return writeNewFileNoSymlink(path, file, write, newFileWriteOps{
			syncFile:   file.Sync,
			closeFile:  file.Close,
			removeFile: os.Remove,
		})
	}

	return writeFileNoSymlinkOverwrite(path, perm, tempPattern, backupPattern, write)
}

type newFileWriteOps struct {
	syncFile   func() error
	closeFile  func() error
	removeFile func(string) error
}

func writeNewFileNoSymlink(path string, file *os.File, write func(*os.File) (int64, error), ops newFileWriteOps) (int64, error) {
	closed := false
	closeFile := func() error {
		if closed {
			return nil
		}
		closed = true
		return ops.closeFile()
	}
	defer func() {
		_ = closeFile()
	}()

	written, err := write(file)
	if err != nil {
		return 0, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := ops.syncFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := closeFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	return written, nil
}

func cleanupIncompleteFile(path string, primaryErr error, closeFile func() error, removeFile func(string) error) error {
	closeErr := closeFile()
	removeErr := removeFile(path)
	if closeErr == nil && removeErr == nil {
		return primaryErr
	}
	return errors.Join(primaryErr, closeErr, removeErr)
}
