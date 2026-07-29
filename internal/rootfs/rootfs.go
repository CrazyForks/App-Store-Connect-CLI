// Package rootfs provides rooted filesystem operations for paths that are not
// fully trusted, such as filenames, directory components, or manifest entries
// that come from a repository checkout or a remote API response.
//
// Every operation is anchored to a trusted root chosen by the operator (for
// example a --out-dir flag, a manifest directory, or the resolved .asc
// directory). Paths are validated lexically so absolute paths, volume or
// UNC-style changes, and parent traversal are rejected, and filesystem access
// refuses to follow symlinks for any component below the root. Writes stage
// through unpredictable, exclusive, no-follow temporary files so a
// pre-created symlink cannot redirect them.
package rootfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/secureopen"
)

var (
	// ErrEscapesRoot reports a path that does not stay beneath the trusted root.
	ErrEscapesRoot = errors.New("path escapes trusted root")
	// ErrSymlink reports a path component that is a symlink below the trusted root.
	ErrSymlink = errors.New("refusing to follow symlink")
)

const (
	temporaryFilePattern = ".asc-tmp-*"
	backupFilePattern    = ".asc-tmp-backup-*"
)

// Root is a trusted directory anchor for rooted filesystem operations.
type Root struct {
	path string
	// internalSymlinks tolerates symlinked components below the root when they
	// resolve back inside the root.
	internalSymlinks bool
}

// New returns a Root anchored at path. The root itself is operator-selected and
// may live outside the current repository; only paths below it are constrained.
func New(path string) (Root, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return Root{}, fmt.Errorf("%w: trusted root path is empty", ErrEscapesRoot)
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return Root{}, fmt.Errorf("resolve trusted root %q: %w", path, err)
	}
	return Root{path: filepath.Clean(absolute)}, nil
}

// Path returns the absolute trusted root path.
func (r Root) Path() string {
	return r.path
}

// AllowingInternalSymlinks returns a copy of the root that accepts a symlinked
// directory component below the root when that component resolves back inside
// the root, and still rejects one that escapes.
//
// Use it only where symlinked directories inside the root are an established,
// supported layout. A symlinked final component is still refused.
func (r Root) AllowingInternalSymlinks() Root {
	r.internalSymlinks = true
	return r
}

// containsResolvedComponent reports whether a symlinked component below the root
// resolves back inside the root.
func (r Root) containsResolvedComponent(path string) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	root := r.path
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// checkSymlinkComponent decides whether a symlinked component below the root is
// acceptable for this root's policy.
func (r Root) checkSymlinkComponent(path string) error {
	if r.internalSymlinks && r.containsResolvedComponent(path) {
		return nil
	}
	return symlinkError(path)
}

// ValidateRelative reports whether name is safe to join onto a trusted root.
// Both Unix and Windows separator conventions are considered so a repository
// can not smuggle a drive-relative, UNC-style, or backslash-traversing path
// past validation on a different host platform.
func ValidateRelative(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: path is empty", ErrEscapesRoot)
	}
	if strings.ContainsRune(trimmed, 0) {
		return fmt.Errorf("%w: %q contains a NUL byte", ErrEscapesRoot, name)
	}
	if isAbsoluteLike(trimmed) {
		return fmt.Errorf("%w: %q must be relative to the trusted root", ErrEscapesRoot, name)
	}
	for _, component := range strings.FieldsFunc(trimmed, isPathSeparator) {
		if component == ".." {
			return fmt.Errorf("%w: %q traverses above the trusted root", ErrEscapesRoot, name)
		}
	}
	return nil
}

// ValidateRelativeAllowingTraversal rejects absolute, drive-relative and
// UNC-style paths but permits ".." segments, for callers that resolve a path
// against a base directory below the root and then confirm containment of the
// joined result with Resolve.
func ValidateRelativeAllowingTraversal(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: path is empty", ErrEscapesRoot)
	}
	if strings.ContainsRune(trimmed, 0) {
		return fmt.Errorf("%w: %q contains a NUL byte", ErrEscapesRoot, name)
	}
	if isAbsoluteLike(trimmed) {
		return fmt.Errorf("%w: %q must be relative to the trusted root", ErrEscapesRoot, name)
	}
	return nil
}

// Resolve validates name and returns its absolute path beneath the root. name
// may be relative to the root or an absolute path that is already inside it.
func (r Root) Resolve(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%w: path is empty", ErrEscapesRoot)
	}
	if strings.ContainsRune(trimmed, 0) {
		return "", fmt.Errorf("%w: %q contains a NUL byte", ErrEscapesRoot, name)
	}

	if isAbsoluteLike(trimmed) {
		if !filepath.IsAbs(trimmed) {
			return "", fmt.Errorf("%w: %q is not an absolute path below %q", ErrEscapesRoot, name, r.path)
		}
		cleaned := filepath.Clean(trimmed)
		if err := r.checkWithin(cleaned, name); err != nil {
			return "", err
		}
		return cleaned, nil
	}

	if err := ValidateRelative(trimmed); err != nil {
		return "", err
	}
	joined := filepath.Join(r.path, trimmed)
	if err := r.checkWithin(joined, name); err != nil {
		return "", err
	}
	return joined, nil
}

// CheckContained verifies that name stays beneath the root and that neither its
// parent components nor its final component is a symlink below the root.
func (r Root) CheckContained(name string) error {
	resolved, err := r.Resolve(name)
	if err != nil {
		return err
	}
	if err := r.checkParentComponents(resolved); err != nil {
		return err
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return symlinkError(resolved)
	}
	return nil
}

// CheckParents verifies that name stays beneath the root and that every
// component below the root leading to it is acceptable under the root's symlink
// policy. The final component is not inspected.
func (r Root) CheckParents(name string) error {
	resolved, err := r.Resolve(name)
	if err != nil {
		return err
	}
	return r.checkParentComponents(resolved)
}

// OpenFile opens an existing regular file beneath the root without following
// symlinks in the final component or in any component below the root.
func (r Root) OpenFile(name string) (*os.File, error) {
	resolved, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	if err := r.checkParentComponents(resolved); err != nil {
		return nil, err
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, symlinkError(resolved)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", resolved)
	}
	return secureopen.OpenExistingNoFollow(resolved)
}

// ReadFile reads a regular file beneath the root without following symlinks.
func (r Root) ReadFile(name string) ([]byte, error) {
	file, err := r.OpenFile(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// ReadFileOptional reads a regular file beneath the root and reports whether it
// exists. A missing file is not an error; a symlinked path still is.
func (r Root) ReadFileOptional(name string) ([]byte, bool, error) {
	data, err := r.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// MkdirAll creates name and any missing parents beneath the root, rejecting any
// existing component that is a symlink or not a directory.
func (r Root) MkdirAll(name string, perm os.FileMode) error {
	resolved, err := r.Resolve(name)
	if err != nil {
		return err
	}
	if err := r.ensureRootDir(perm); err != nil {
		return err
	}
	components, err := r.componentsBelowRoot(resolved)
	if err != nil {
		return err
	}
	current := r.path
	for _, component := range components {
		current = filepath.Join(current, component)
		if err := r.mkdirNoFollow(current, perm); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile atomically creates or replaces a file beneath the root.
func (r Root) WriteFile(name string, data []byte, perm os.FileMode) error {
	_, err := r.WriteFrom(name, bytes.NewReader(data), perm)
	return err
}

// WriteFrom atomically creates or replaces a file beneath the root with the
// contents of reader and returns the number of bytes written.
func (r Root) WriteFrom(name string, reader io.Reader, perm os.FileMode) (int64, error) {
	resolved, err := r.prepareWrite(name)
	if err != nil {
		return 0, err
	}

	hadExisting, err := checkReplaceableFile(resolved)
	if err != nil {
		return 0, err
	}

	directory := filepath.Dir(resolved)
	temporary, err := secureopen.CreateTempNoFollow(directory, temporaryFilePattern, perm)
	if err != nil {
		return 0, err
	}
	temporaryPath := temporary.Name()
	success := false
	defer func() {
		_ = temporary.Close()
		if !success {
			_ = os.Remove(temporaryPath)
		}
	}()

	// Set the final mode explicitly so the process umask cannot widen it.
	if err := temporary.Chmod(perm); err != nil {
		return 0, err
	}
	written, err := io.Copy(temporary, reader)
	if err != nil {
		return 0, err
	}
	if err := temporary.Sync(); err != nil {
		return 0, err
	}
	if err := temporary.Close(); err != nil {
		return 0, err
	}

	if err := replaceFile(temporaryPath, resolved, hadExisting); err != nil {
		return 0, err
	}
	success = true
	return written, nil
}

// CreateNewFile writes data to a new file beneath the root and fails when the
// destination already exists.
func (r Root) CreateNewFile(name string, data []byte, perm os.FileMode) error {
	resolved, err := r.prepareWrite(name)
	if err != nil {
		return err
	}

	info, err := os.Lstat(resolved)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return symlinkError(resolved)
		}
		return fmt.Errorf("%q already exists: %w", resolved, os.ErrExist)
	case !errors.Is(err, os.ErrNotExist):
		return err
	}

	file, err := secureopen.OpenNewFileNoFollow(resolved, perm)
	if err != nil {
		return err
	}
	if _, writeErr := file.Write(data); writeErr != nil {
		_ = file.Close()
		return writeErr
	}
	return file.Close()
}

// AppendFile appends data to a file beneath the root, creating it when missing,
// without following a final or parent symlink.
func (r Root) AppendFile(name string, data []byte, perm os.FileMode) error {
	resolved, err := r.prepareWrite(name)
	if err != nil {
		return err
	}

	if info, err := os.Lstat(resolved); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return symlinkError(resolved)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%q is not a regular file", resolved)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	file, err := secureopen.OpenAppendNoFollow(resolved, perm)
	if err != nil {
		return err
	}
	// The descriptor is known not to be a symlink, so tightening permissions
	// here cannot affect an external file.
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (r Root) prepareWrite(name string) (string, error) {
	resolved, err := r.Resolve(name)
	if err != nil {
		return "", err
	}
	if resolved == r.path {
		return "", fmt.Errorf("%w: %q is the trusted root itself", ErrEscapesRoot, name)
	}
	parent, err := r.relativeToRoot(filepath.Dir(resolved))
	if err != nil {
		return "", err
	}
	if err := r.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	return resolved, nil
}

func (r Root) ensureRootDir(perm os.FileMode) error {
	info, err := os.Stat(r.path)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("trusted root %q is not a directory", r.path)
		}
		return nil
	case !errors.Is(err, os.ErrNotExist):
		return err
	}
	if err := os.MkdirAll(r.path, perm); err != nil {
		return err
	}
	return nil
}

func (r Root) componentsBelowRoot(absolute string) ([]string, error) {
	relative, err := r.relativeToRoot(absolute)
	if err != nil {
		return nil, err
	}
	if relative == "." {
		return nil, nil
	}
	parts := strings.Split(relative, string(filepath.Separator))
	components := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		components = append(components, part)
	}
	return components, nil
}

func (r Root) relativeToRoot(absolute string) (string, error) {
	relative, err := filepath.Rel(r.path, absolute)
	if err != nil {
		return "", fmt.Errorf("%w: %q is not below %q", ErrEscapesRoot, absolute, r.path)
	}
	return relative, nil
}

func (r Root) checkWithin(absolute string, original string) error {
	if !strings.EqualFold(filepath.VolumeName(absolute), filepath.VolumeName(r.path)) {
		return fmt.Errorf("%w: %q changes volume from %q", ErrEscapesRoot, original, r.path)
	}
	relative, err := filepath.Rel(r.path, absolute)
	if err != nil {
		return fmt.Errorf("%w: %q is not below %q", ErrEscapesRoot, original, r.path)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q is not below %q", ErrEscapesRoot, original, r.path)
	}
	return nil
}

func (r Root) checkParentComponents(absolute string) error {
	components, err := r.componentsBelowRoot(absolute)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		return nil
	}
	current := r.path
	for _, component := range components[:len(components)-1] {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if err := r.checkSymlinkComponent(current); err != nil {
				return err
			}
			if resolved, err := os.Stat(current); err != nil {
				return err
			} else if !resolved.IsDir() {
				return fmt.Errorf("%q is not a directory", current)
			}
			continue
		}
		if !info.IsDir() {
			return fmt.Errorf("%q is not a directory", current)
		}
	}
	return nil
}

func (r Root) mkdirNoFollow(path string, perm os.FileMode) error {
	if err := r.validateExistingDir(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Mkdir(path, perm); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return r.validateExistingDir(path)
}

func (r Root) validateExistingDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if err := r.checkSymlinkComponent(path); err != nil {
			return err
		}
		resolved, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !resolved.IsDir() {
			return fmt.Errorf("%q is not a directory", path)
		}
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}

func checkReplaceableFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, symlinkError(path)
	}
	if info.IsDir() {
		return false, fmt.Errorf("%q is a directory", path)
	}
	return true, nil
}

// replaceFile moves the staged temporary file onto path. Unix renames replace
// the destination atomically; Windows renames fail when the destination exists,
// so the original is moved aside first and restored if the final move fails.
func replaceFile(temporaryPath, path string, hadExisting bool) error {
	if err := os.Rename(temporaryPath, path); err == nil {
		return nil
	} else if !hadExisting || runtime.GOOS != "windows" {
		return err
	}

	backup, err := secureopen.CreateTempNoFollow(filepath.Dir(path), backupFilePattern, 0o600)
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Rename(backupPath, path)
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func symlinkError(path string) error {
	return fmt.Errorf("%w: %q", ErrSymlink, path)
}

func isAbsoluteLike(path string) bool {
	if path == "" {
		return false
	}
	if isPathSeparator(rune(path[0])) {
		return true
	}
	if filepath.IsAbs(path) {
		return true
	}
	return len(path) >= 2 && path[1] == ':' && isASCIILetter(path[0])
}

func isPathSeparator(r rune) bool {
	return r == '/' || r == '\\'
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
