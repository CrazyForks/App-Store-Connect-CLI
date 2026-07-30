//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package secureopen

import (
	"fmt"
	"os"
)

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

// OpenNewFileNoFollowInRoot creates a new file relative to root using
// best-effort final-component checks. Root itself prevents parent traversal.
func OpenNewFileNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return openNewFileNoFollowInRootBestEffort(root, name, perm, func() (*os.File, error) {
		return root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	})
}

// OpenAppendNoFollowInRoot opens a file for appending relative to root using
// best-effort final-component checks. Root itself prevents parent traversal.
func OpenAppendNoFollowInRoot(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return openNewFileNoFollowInRootBestEffort(root, name, perm, func() (*os.File, error) {
		return root.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm)
	})
}

// OpenExistingNoFollowInRoot opens an existing file relative to root using
// best-effort final-component checks. Root itself prevents parent traversal.
func OpenExistingNoFollowInRoot(root *os.Root, name string) (*os.File, error) {
	before, err := rootLstatNoSymlink(root, name)
	if err != nil {
		return nil, err
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	if err := verifyRootOpenedPath(root, name, file, before); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func openNewFileNoFollowInRootBestEffort(root *os.Root, name string, perm os.FileMode, opener func() (*os.File, error)) (*os.File, error) {
	if _, err := rootLstatNoSymlink(root, name); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	file, err := opener()
	if err != nil {
		return nil, err
	}
	if err := verifyRootOpenedPath(root, name, file, nil); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func rootLstatNoSymlink(root *os.Root, name string) (os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to follow symlink %q", name)
	}
	return info, nil
}

func verifyRootOpenedPath(root *os.Root, name string, file *os.File, before os.FileInfo) error {
	openedInfo, err := file.Stat()
	if err != nil {
		return err
	}
	after, err := rootLstatNoSymlink(root, name)
	if err != nil {
		return err
	}
	if before != nil && !os.SameFile(before, after) {
		return fmt.Errorf("file changed during open %q", name)
	}
	if !os.SameFile(after, openedInfo) {
		return fmt.Errorf("file changed during open %q", name)
	}
	return nil
}
