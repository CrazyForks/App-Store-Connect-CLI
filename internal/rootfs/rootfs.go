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
//
// Roots created with AllowingInternalSymlinks relax only the parent-component
// rule, accepting a symlinked directory whose target stays inside the root; a
// symlinked final component is always refused.
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
	// afterValidationForTest makes path-swap regressions deterministic. It is
	// intentionally unexported and unset outside package tests.
	afterValidationForTest func()
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
	parent, base, err := r.openParentRooted(resolved)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	if err := r.checkParentComponents(resolved); err != nil {
		return nil, err
	}
	info, err := parent.Lstat(base)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, symlinkError(resolved)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular file", resolved)
	}
	if r.afterValidationForTest != nil {
		r.afterValidationForTest()
	}
	file, err := secureopen.OpenExistingNoFollowInRoot(parent, base)
	if err != nil {
		return nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("%q is not a regular file", resolved)
	}
	return file, nil
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
	rooted, relative, err := r.openRooted(resolved, true)
	if err != nil {
		return err
	}
	defer rooted.Close()
	if err := r.validateDirectoryComponents(resolved); err != nil {
		return err
	}
	if err := rooted.MkdirAll(relative, perm); err != nil {
		return err
	}
	return r.validateDirectoryComponents(resolved)
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
	parent, base, err := r.openParentRooted(resolved)
	if err != nil {
		return 0, err
	}
	defer parent.Close()

	hadExisting, err := checkReplaceableFileInRoot(parent, base, resolved)
	if err != nil {
		return 0, err
	}
	if r.afterValidationForTest != nil {
		r.afterValidationForTest()
	}

	temporary, temporaryName, err := secureopen.CreateTempNoFollowInRoot(parent, ".", temporaryFilePattern, perm)
	if err != nil {
		return 0, err
	}
	success := false
	defer func() {
		_ = temporary.Close()
		if !success {
			_ = parent.Remove(temporaryName)
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

	if err := replaceFileInRoot(parent, temporaryName, base, hadExisting); err != nil {
		return 0, err
	}
	success = true
	return written, nil
}

// WriteFilePreservingMode atomically creates or replaces a file beneath the
// root, reusing an existing regular destination's permissions and falling back
// to perm for a new file. Use it where the pre-rooted in-place write preserved
// an operator's chosen mode across rewrites.
func (r Root) WriteFilePreservingMode(name string, data []byte, perm os.FileMode) error {
	resolved, err := r.prepareWrite(name)
	if err != nil {
		return err
	}
	parent, base, err := r.openParentRooted(resolved)
	if err != nil {
		return err
	}
	if info, err := parent.Lstat(base); err == nil {
		if info.Mode().IsRegular() {
			perm = info.Mode().Perm()
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = parent.Close()
		return err
	}
	if err := parent.Close(); err != nil {
		return err
	}
	return r.WriteFile(name, data, perm)
}

// CreateNewFile writes data to a new file beneath the root and fails when the
// destination already exists.
func (r Root) CreateNewFile(name string, data []byte, perm os.FileMode) error {
	resolved, err := r.prepareWrite(name)
	if err != nil {
		return err
	}
	parent, base, err := r.openParentRooted(resolved)
	if err != nil {
		return err
	}
	defer parent.Close()

	info, err := parent.Lstat(base)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return symlinkError(resolved)
		}
		return fmt.Errorf("%q already exists: %w", resolved, os.ErrExist)
	case !errors.Is(err, os.ErrNotExist):
		return err
	}
	if r.afterValidationForTest != nil {
		r.afterValidationForTest()
	}

	file, err := secureopen.OpenNewFileNoFollowInRoot(parent, base, perm)
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
	parent, base, err := r.openParentRooted(resolved)
	if err != nil {
		return err
	}
	defer parent.Close()

	if info, err := parent.Lstat(base); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return symlinkError(resolved)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%q is not a regular file", resolved)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if r.afterValidationForTest != nil {
		r.afterValidationForTest()
	}

	file, err := secureopen.OpenAppendNoFollowInRoot(parent, base, perm)
	if err != nil {
		return err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	if !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return fmt.Errorf("%q is not a regular file", resolved)
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

func (r Root) openRooted(absolute string, resolveFinal bool) (*os.Root, string, error) {
	rooted, err := os.OpenRoot(r.path)
	if err != nil {
		return nil, "", err
	}
	relative, err := r.rootedRelative(absolute, resolveFinal)
	if err != nil {
		_ = rooted.Close()
		return nil, "", err
	}
	return rooted, relative, nil
}

func (r Root) openParentRooted(absolute string) (*os.Root, string, error) {
	rooted, relative, err := r.openRooted(absolute, false)
	if err != nil {
		return nil, "", err
	}
	parent, err := rooted.OpenRoot(filepath.Dir(relative))
	_ = rooted.Close()
	if err != nil {
		return nil, "", err
	}
	return parent, filepath.Base(relative), nil
}

// rootedRelative converts an already-contained absolute path into a name for
// os.Root. Existing internal directory symlinks are resolved to their physical
// path so AllowingInternalSymlinks remains compatible with absolute in-root
// links, while the final file component remains unresolved for no-follow open.
func (r Root) rootedRelative(absolute string, resolveFinal bool) (string, error) {
	components, err := r.componentsBelowRoot(absolute)
	if err != nil {
		return "", err
	}
	physicalRoot, err := filepath.EvalSymlinks(r.path)
	if err != nil {
		return "", err
	}
	current := physicalRoot
	resolveCount := len(components)
	if !resolveFinal && resolveCount > 0 {
		resolveCount--
	}

	for index := 0; index < resolveCount; index++ {
		candidate := filepath.Join(current, components[index])
		info, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			for _, remaining := range components[index:] {
				current = filepath.Join(current, remaining)
			}
			return relativeWithinRoot(physicalRoot, current)
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if !r.internalSymlinks {
				return "", symlinkError(candidate)
			}
			resolved, err := filepath.EvalSymlinks(candidate)
			if err != nil {
				return "", err
			}
			if _, err := relativeWithinRoot(physicalRoot, resolved); err != nil {
				return "", symlinkError(candidate)
			}
			resolvedInfo, err := os.Stat(candidate)
			if err != nil {
				return "", err
			}
			if !resolvedInfo.IsDir() {
				return "", fmt.Errorf("%q is not a directory", candidate)
			}
			current = resolved
			continue
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%q is not a directory", candidate)
		}
		current = candidate
	}
	for _, remaining := range components[resolveCount:] {
		current = filepath.Join(current, remaining)
	}
	return relativeWithinRoot(physicalRoot, current)
}

func relativeWithinRoot(root string, path string) (string, error) {
	if !strings.EqualFold(filepath.VolumeName(path), filepath.VolumeName(root)) {
		return "", fmt.Errorf("%w: %q changes volume from %q", ErrEscapesRoot, path, root)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("%w: %q is not below %q", ErrEscapesRoot, path, root)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q is not below %q", ErrEscapesRoot, path, root)
	}
	return relative, nil
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

func (r Root) validateDirectoryComponents(absolute string) error {
	components, err := r.componentsBelowRoot(absolute)
	if err != nil {
		return err
	}
	current := r.path
	for _, component := range components {
		current = filepath.Join(current, component)
		if err := r.validateExistingDir(current); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
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

func checkReplaceableFileInRoot(rooted *os.Root, name string, displayPath string) (bool, error) {
	info, err := rooted.Lstat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, symlinkError(displayPath)
	}
	if info.IsDir() {
		return false, fmt.Errorf("%q is a directory", displayPath)
	}
	return true, nil
}

// replaceFileInRoot moves the staged temporary file onto path. Unix renames replace
// the destination atomically; Windows renames fail when the destination exists,
// so the original is moved aside first and restored if the final move fails.
func replaceFileInRoot(parent *os.Root, temporaryName string, name string, hadExisting bool) error {
	if err := parent.Rename(temporaryName, name); err == nil {
		return nil
	} else if !hadExisting || runtime.GOOS != "windows" {
		return err
	}

	backup, backupName, err := secureopen.CreateTempNoFollowInRoot(parent, ".", backupFilePattern, 0o600)
	if err != nil {
		return err
	}
	if err := backup.Close(); err != nil {
		return err
	}
	if err := parent.Remove(backupName); err != nil {
		return err
	}
	if err := parent.Rename(name, backupName); err != nil {
		return err
	}
	if err := parent.Rename(temporaryName, name); err != nil {
		_ = parent.Rename(backupName, name)
		return err
	}
	_ = parent.Remove(backupName)
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
