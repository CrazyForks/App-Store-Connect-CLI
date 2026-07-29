package rootfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateRelativeRejectsEscapes(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"unix absolute", "/etc/passwd"},
		{"windows absolute backslash", `\Windows\System32`},
		{"windows drive absolute", `C:\Windows\System32`},
		{"windows drive relative", "C:evil.txt"},
		{"unc share", `\\attacker\share\evil.txt`},
		{"parent traversal", "../evil.txt"},
		{"nested parent traversal", "a/../../evil.txt"},
		{"bare parent", ".."},
		{"backslash parent traversal", `..\evil.txt`},
		{"mixed separator traversal", `a\..\..\evil.txt`},
		{"nul byte", "a\x00b"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := ValidateRelative(testCase.path); err == nil {
				t.Fatalf("ValidateRelative(%q) = nil, want error", testCase.path)
			}
		})
	}
}

func TestValidateRelativeAcceptsOrdinaryPaths(t *testing.T) {
	cases := []string{
		"file.txt",
		"metadata/en-US/description.txt",
		"./file.txt",
		"a..b/c.txt",
		"...hidden",
	}

	for _, path := range cases {
		if err := ValidateRelative(path); err != nil {
			t.Fatalf("ValidateRelative(%q) error = %v, want nil", path, err)
		}
	}
}

func TestResolveRejectsEscapingAbsolutePath(t *testing.T) {
	root := mustRoot(t, t.TempDir())
	outside := filepath.Join(t.TempDir(), "sentinel.txt")

	if _, err := root.Resolve(outside); !errors.Is(err, ErrEscapesRoot) {
		t.Fatalf("Resolve(%q) error = %v, want ErrEscapesRoot", outside, err)
	}
}

func TestResolveAcceptsAbsolutePathInsideRoot(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)
	inside := filepath.Join(dir, "nested", "file.txt")

	resolved, err := root.Resolve(inside)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", inside, err)
	}
	if resolved != filepath.Clean(inside) {
		t.Fatalf("Resolve(%q) = %q, want %q", inside, resolved, filepath.Clean(inside))
	}
}

func TestReadFileRefusesSymlinkedFinalComponent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	secretDir := t.TempDir()
	secretPath := filepath.Join(secretDir, "secret.txt")
	mustWrite(t, secretPath, "top-secret")

	linkPath := filepath.Join(dir, "description.txt")
	if err := os.Symlink(secretPath, linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	if _, err := root.ReadFile("description.txt"); !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadFile() error = %v, want ErrSymlink", err)
	}
}

func TestReadFileRefusesSymlinkedParentComponent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	secretDir := t.TempDir()
	mustWrite(t, filepath.Join(secretDir, "secret.txt"), "top-secret")

	if err := os.Symlink(secretDir, filepath.Join(dir, "en-US")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	if _, err := root.ReadFile(filepath.Join("en-US", "secret.txt")); !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadFile() error = %v, want ErrSymlink", err)
	}
}

func TestReadFileReadsOrdinaryFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "keywords.txt"), "one,two")

	root := mustRoot(t, dir)
	data, err := root.ReadFile("keywords.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "one,two" {
		t.Fatalf("ReadFile() = %q, want %q", data, "one,two")
	}
}

func TestReadFileOptionalReportsMissingWithoutError(t *testing.T) {
	root := mustRoot(t, t.TempDir())

	data, found, err := root.ReadFileOptional("missing.txt")
	if err != nil {
		t.Fatalf("ReadFileOptional() error = %v", err)
	}
	if found {
		t.Fatal("ReadFileOptional() found = true, want false")
	}
	if len(data) != 0 {
		t.Fatalf("ReadFileOptional() data = %q, want empty", data)
	}
}

func TestWriteFileRefusesFinalSymlink(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "sentinel.txt")
	mustWrite(t, sentinelPath, "original")

	if err := os.Symlink(sentinelPath, filepath.Join(dir, "out.json")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	if err := root.WriteFile("out.json", []byte("attacker"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("WriteFile() error = %v, want ErrSymlink", err)
	}
	if got := mustRead(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
}

func TestWriteFileRefusesSymlinkedParent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "sentinel.txt")
	mustWrite(t, sentinelPath, "original")

	if err := os.Symlink(sentinelDir, filepath.Join(dir, "nested")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	err := root.WriteFile(filepath.Join("nested", "sentinel.txt"), []byte("attacker"), 0o600)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("WriteFile() error = %v, want ErrSymlink", err)
	}
	if got := mustRead(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
}

func TestWriteFileCreatesAndReplacesInRoot(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)

	target := filepath.Join("nested", "deep", "out.json")
	if err := root.WriteFile(target, []byte("first"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, target)); got != "first" {
		t.Fatalf("content = %q, want %q", got, "first")
	}
	if err := root.WriteFile(target, []byte("second"), 0o600); err != nil {
		t.Fatalf("WriteFile() replace error = %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, target)); got != "second" {
		t.Fatalf("content = %q, want %q", got, "second")
	}

	info, err := os.Lstat(filepath.Join(dir, target))
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	if leftovers := temporaryLeftovers(t, filepath.Join(dir, "nested", "deep")); len(leftovers) > 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
}

func TestWriteFileDoesNotUsePredictableTemporaryPath(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "sentinel.txt")
	mustWrite(t, sentinelPath, "original")

	// A predictable "<target>.tmp" staging path lets a lower-trust checkout
	// pre-create a symlink that redirects the staged write.
	if err := os.Symlink(sentinelPath, filepath.Join(dir, "checkpoint.json.tmp")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	if err := root.WriteFile("checkpoint.json", []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if got := mustRead(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
	if got := mustRead(t, filepath.Join(dir, "checkpoint.json")); got != `{"ok":true}` {
		t.Fatalf("checkpoint content = %q", got)
	}
}

func TestWriteFromWritesReaderContents(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)

	written, err := root.WriteFrom("payload.bin", bytes.NewReader([]byte("payload")), 0o600)
	if err != nil {
		t.Fatalf("WriteFrom() error = %v", err)
	}
	if written != int64(len("payload")) {
		t.Fatalf("WriteFrom() = %d, want %d", written, len("payload"))
	}
	if got := mustRead(t, filepath.Join(dir, "payload.bin")); got != "payload" {
		t.Fatalf("content = %q", got)
	}
}

func TestCreateNewFileRefusesExistingFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "AuthKey.p8"), "existing")

	root := mustRoot(t, dir)
	if err := root.CreateNewFile("AuthKey.p8", []byte("new"), 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("CreateNewFile() error = %v, want os.ErrExist", err)
	}
	if got := mustRead(t, filepath.Join(dir, "AuthKey.p8")); got != "existing" {
		t.Fatalf("content = %q, want %q", got, "existing")
	}
}

func TestMkdirAllRefusesSymlinkedComponent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "metadata")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	err := root.MkdirAll(filepath.Join("metadata", "en-US"), 0o755)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("MkdirAll() error = %v, want ErrSymlink", err)
	}
	if _, statErr := os.Stat(filepath.Join(outside, "en-US")); statErr == nil {
		t.Fatal("MkdirAll() created a directory outside the root")
	}
}

func TestMkdirAllCreatesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)

	if err := root.MkdirAll(filepath.Join("metadata", "en-US"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "metadata", "en-US"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Fatal("MkdirAll() did not create a directory")
	}
}

func TestMkdirAllCreatesMissingRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "created-root")
	root := mustRoot(t, dir)

	if err := root.MkdirAll(".", 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("Stat(root) = %v, %v", info, err)
	}
}

func TestAppendFileRefusesSymlinkAndLeavesModeIntact(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "sentinel.txt")
	mustWrite(t, sentinelPath, "original")
	if err := os.Chmod(sentinelPath, 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	if err := os.Symlink(sentinelPath, filepath.Join(dir, "snitch.log")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir)
	if err := root.AppendFile("snitch.log", []byte("entry\n"), 0o600); !errors.Is(err, ErrSymlink) {
		t.Fatalf("AppendFile() error = %v, want ErrSymlink", err)
	}
	if got := mustRead(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
	info, err := os.Lstat(sentinelPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("sentinel mode = %v, want 0644", info.Mode().Perm())
	}
}

func TestAppendFileAppendsInRoot(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)

	if err := root.AppendFile("snitch.log", []byte("one\n"), 0o600); err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}
	if err := root.AppendFile("snitch.log", []byte("two\n"), 0o600); err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}
	if got := mustRead(t, filepath.Join(dir, "snitch.log")); got != "one\ntwo\n" {
		t.Fatalf("content = %q", got)
	}
}

func TestAllowingInternalSymlinksAcceptsContainedSymlinkedParent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	realDir := filepath.Join(dir, "SharedGenerated")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Symlink(realDir, filepath.Join(dir, "Generated")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir).AllowingInternalSymlinks()
	if err := root.WriteFile(filepath.Join("Generated", "Info.plist"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if got := mustRead(t, filepath.Join(realDir, "Info.plist")); got != "payload" {
		t.Fatalf("content = %q", got)
	}
}

func TestAllowingInternalSymlinksRejectsEscapingSymlinkedParent(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(dir, "Generated")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir).AllowingInternalSymlinks()
	err := root.WriteFile(filepath.Join("Generated", "Info.plist"), []byte("payload"), 0o644)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("WriteFile() error = %v, want ErrSymlink", err)
	}
	if _, statErr := os.Stat(filepath.Join(external, "Info.plist")); statErr == nil {
		t.Fatal("WriteFile() escaped through a symlinked parent")
	}
}

func TestAllowingInternalSymlinksStillRejectsFinalSymlink(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.txt")
	mustWrite(t, sentinelPath, "original")
	if err := os.Symlink(sentinelPath, filepath.Join(dir, "Info.plist")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root := mustRoot(t, dir).AllowingInternalSymlinks()
	if err := root.WriteFile("Info.plist", []byte("payload"), 0o644); !errors.Is(err, ErrSymlink) {
		t.Fatalf("WriteFile() error = %v, want ErrSymlink", err)
	}
	if got := mustRead(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
}

func TestCheckContainedRejectsSymlinkedParentForExternalCandidate(t *testing.T) {
	requireSymlinks(t)

	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "Configs")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	mustWrite(t, filepath.Join(outside, "Shared.xcconfig"), "MARKETING_VERSION = 1.0.0\n")

	root := mustRoot(t, dir)
	err := root.CheckContained(filepath.Join(dir, "Configs", "Shared.xcconfig"))
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("CheckContained() error = %v, want ErrSymlink", err)
	}
}

func TestErrorMessagesIdentifyRejectedPath(t *testing.T) {
	dir := t.TempDir()
	root := mustRoot(t, dir)

	_, err := root.Resolve("../escape.txt")
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "../escape.txt") {
		t.Fatalf("error %q does not identify the rejected path", err)
	}
}

func mustRoot(t *testing.T, path string) Root {
	t.Helper()
	root, err := New(path)
	if err != nil {
		t.Fatalf("New(%q) error = %v", path, err)
	}
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}

func temporaryLeftovers(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	var leftovers []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".asc-tmp-") {
			leftovers = append(leftovers, entry.Name())
		}
	}
	return leftovers
}

func requireSymlinks(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, "link")); err != nil {
		t.Skip("symlink creation is not permitted on this host")
	}
}
