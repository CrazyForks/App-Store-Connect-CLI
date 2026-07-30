//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package secureopen

import "os"

// OpenNewFileNoFollow creates a new file without following symlinks.
// Uses a best-effort pre/post open validation sequence because a portable
// O_NOFOLLOW equivalent is not available on this platform.
func OpenNewFileNoFollow(path string, perm os.FileMode) (*os.File, error) {
	return openNewFileNoFollowBestEffort(path, perm, func(path string, perm os.FileMode) (*os.File, error) {
		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		return os.OpenFile(path, flags, perm)
	})
}

// OpenAppendNoFollow opens a file for appending using best-effort symlink
// checks because a portable O_NOFOLLOW equivalent is not available on this
// platform.
func OpenAppendNoFollow(path string, perm os.FileMode) (*os.File, error) {
	return openNewFileNoFollowBestEffort(path, perm, func(path string, perm os.FileMode) (*os.File, error) {
		flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE
		return os.OpenFile(path, flags, perm)
	})
}

// OpenExistingNoFollow opens an existing file with best-effort symlink checks.
// The validation/open sequence reduces, but does not eliminate, TOCTOU risk.
func OpenExistingNoFollow(path string) (*os.File, error) {
	return openExistingNoFollowBestEffort(path, os.Open)
}
