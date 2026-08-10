package shared

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/secureopen"
)

// SafeWriteFileNoSymlink writes a file to path without following symlinks and with an optional
// overwrite mode that preserves the original destination until the new file is fully written.
//
// When overwrite is false, the destination must not already exist.
// When overwrite is true, we refuse to overwrite symlinks and we use temp+rename; if rename fails
// because the destination exists (notably on Windows), we fall back to a safe replace that uses a
// backup file to preserve the original if the final move fails.
func SafeWriteFileNoSymlink(path string, perm os.FileMode, overwrite bool, tempPattern string, backupPattern string, write func(*os.File) (int64, error)) (int64, error) {
	if len(path) > 0 && os.IsPathSeparator(path[len(path)-1]) {
		return 0, fmt.Errorf("output path %q must be a file", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}

	if !overwrite {
		// Callers pass a complete, already-resolved output path. Pin its immediate
		// parent so staging, publication, and cleanup cannot be redirected by a
		// concurrent parent rename.
		parent, err := os.OpenRoot(filepath.Dir(path))
		if err != nil {
			return 0, err
		}
		defer parent.Close()

		base := filepath.Base(path)
		if _, err := parent.Lstat(base); err == nil {
			return 0, fmt.Errorf("output file already exists: %w", os.ErrExist)
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}

		file, temporaryName, err := secureopen.CreateTempNoFollowInRoot(parent, ".", tempPattern, perm)
		if err != nil {
			return 0, err
		}
		temporaryPath := file.Name()
		displayDestinationError := func(err error) error {
			return replaceErrorPaths(err, path, temporaryPath, temporaryName)
		}
		displayTemporaryError := func(err error) error {
			return replaceErrorPaths(err, "temporary output", temporaryPath, temporaryName)
		}
		displayPublishError := func(err error) error {
			err = displayDestinationError(err)
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("output file already exists: %w", err)
			}
			return err
		}
		written, err := writeNewFileNoSymlink(temporaryName, file, func(file *os.File) (int64, error) {
			written, err := write(file)
			return written, displayDestinationError(err)
		}, newFileWriteOps{
			syncFile: func() error {
				return displayDestinationError(file.Sync())
			},
			closeFile: func() error {
				return displayDestinationError(file.Close())
			},
			removeFile: func(name string) error {
				return displayTemporaryError(removeRootedFile(parent, name))
			},
		})
		if err != nil {
			return written, err
		}

		if err := publishStagedFileNoReplace(temporaryName, base, stagedFilePublishOps{
			renameFile: func(oldName, newName string) error {
				return displayPublishError(secureopen.RenameNoReplaceInRoot(parent, oldName, newName))
			},
			linkFile: func(oldName, newName string) error {
				return displayPublishError(parent.Link(oldName, newName))
			},
			removeFile: func(name string) error {
				return displayTemporaryError(removeRootedFile(parent, name))
			},
		}); err != nil {
			return written, err
		}
		return written, nil
	}

	return writeFileNoSymlinkOverwrite(path, perm, tempPattern, backupPattern, write)
}

type newFileWriteOps struct {
	syncFile   func() error
	closeFile  func() error
	removeFile func(string) error
}

type stagedFilePublishOps struct {
	renameFile func(string, string) error
	linkFile   func(string, string) error
	removeFile func(string) error
}

func publishStagedFileNoReplace(temporaryName, destinationName string, ops stagedFilePublishOps) error {
	removed := false
	removeStaged := func() error {
		err := ops.removeFile(temporaryName)
		if err == nil {
			removed = true
		}
		return err
	}
	defer func() {
		if !removed {
			_ = removeStaged()
		}
	}()

	renameErr := ops.renameFile(temporaryName, destinationName)
	if renameErr == nil {
		// Rename consumes the staged name, so no cleanup remains.
		removed = true
		return nil
	}
	if !errors.Is(renameErr, secureopen.ErrRenameNoReplaceUnsupported) {
		return cleanupIncompleteFile(temporaryName, renameErr, func() error { return nil }, func(string) error {
			return removeStaged()
		})
	}

	// Filesystems without a native no-replace rename may still support hard
	// links. Link is also atomic and cannot replace an existing destination.
	if err := ops.linkFile(temporaryName, destinationName); err != nil {
		return cleanupIncompleteFile(temporaryName, err, func() error { return nil }, func(string) error {
			return removeStaged()
		})
	}
	// Once publication succeeds, the complete destination is committed. Cleanup
	// remains best effort so callers do not retry a write that already succeeded.
	_ = removeStaged()
	return nil
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
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := ops.syncFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	if err := closeFile(); err != nil {
		return written, cleanupIncompleteFile(path, err, closeFile, ops.removeFile)
	}
	return written, nil
}

type displayPathError struct {
	err     error
	message string
}

func (e *displayPathError) Error() string {
	return e.message
}

func (e *displayPathError) Unwrap() error {
	return e.err
}

func replaceErrorPaths(err error, replacement string, paths ...string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, path := range paths {
		if path != "" && path != replacement {
			message = strings.ReplaceAll(message, path, replacement)
		}
	}
	if message == err.Error() {
		return err
	}
	return &displayPathError{err: err, message: message}
}

func removeRootedFile(root *os.Root, name string) error {
	err := root.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func cleanupIncompleteFile(path string, primaryErr error, closeFile func() error, removeFile func(string) error) error {
	closeErr := closeFile()
	removeErr := removeFile(path)
	if closeErr == nil && removeErr == nil {
		return primaryErr
	}
	return errors.Join(primaryErr, closeErr, removeErr)
}
