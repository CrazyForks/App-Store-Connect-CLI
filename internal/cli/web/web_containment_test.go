package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadRootRefusesSymlinkedRepositoryASCDirectory(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	externalDir := t.TempDir()
	if err := os.Symlink(externalDir, filepath.Join(workDir, ".asc")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	// The default --out location is repository-controlled, so a symlinked .asc
	// component must be refused before any directory or file is created.
	outDir := resolveShowOutDir("app", "submission", "")
	root, prefix, err := newDownloadRoot(outDir)
	if err != nil {
		t.Fatalf("newDownloadRoot() error = %v", err)
	}
	if mkdirErr := root.MkdirAll(prefix, 0o755); mkdirErr == nil {
		t.Fatal("MkdirAll() error = nil, want symlink rejection")
	} else if !strings.Contains(mkdirErr.Error(), "symlink") {
		t.Fatalf("MkdirAll() error = %v, want symlink rejection", mkdirErr)
	}

	entries, readErr := os.ReadDir(externalDir)
	if readErr != nil {
		t.Fatalf("ReadDir() error = %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("download directory escaped into the symlink target: %v", entries)
	}
}

func TestWriteAttachmentRefusesSymlinkedDestination(t *testing.T) {
	outDir := t.TempDir()
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.txt")
	writeWebContainmentFile(t, sentinelPath, "original")

	if err := os.Symlink(sentinelPath, filepath.Join(outDir, "screenshot.png")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	root, prefix, err := newDownloadRoot(outDir)
	if err != nil {
		t.Fatalf("newDownloadRoot() error = %v", err)
	}
	_, err = resolveDownloadPath(root, prefix, "screenshot.png", true)
	if err == nil {
		t.Fatal("resolveDownloadPath() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("resolveDownloadPath() error = %v, want symlink rejection", err)
	}
	if got := readWebContainmentFile(t, sentinelPath); got != "original" {
		t.Fatalf("sentinel content = %q, want %q", got, "original")
	}
}

func TestWriteAttachmentRefusesSymlinkedParentDirectory(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "downloads")
	external := t.TempDir()
	if err := os.Symlink(external, outDir); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	// The output directory itself is operator-selected, so a symlinked --out is
	// still honoured; a symlinked directory *inside* it is not.
	root, prefix, err := newDownloadRoot(outDir)
	if err != nil {
		t.Fatalf("newDownloadRoot() error = %v", err)
	}
	nestedExternal := t.TempDir()
	if err := os.Symlink(nestedExternal, filepath.Join(external, "nested")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if err := writeDownloadedAttachment(root, filepath.Join(prefix, "nested", "screenshot.png"), []byte("payload")); err == nil {
		t.Fatal("writeDownloadedAttachment() error = nil, want symlink rejection")
	} else if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("writeDownloadedAttachment() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(nestedExternal, "screenshot.png")); statErr == nil {
		t.Fatal("attachment escaped through a symlinked parent directory")
	}
}

func TestResolveDownloadPathRejectsTraversingFileName(t *testing.T) {
	outDir := t.TempDir()
	root, prefix, err := newDownloadRoot(outDir)
	if err != nil {
		t.Fatalf("newDownloadRoot() error = %v", err)
	}

	if _, err := resolveDownloadPath(root, prefix, filepath.Join("..", "outside.txt"), true); err == nil {
		t.Fatal("resolveDownloadPath() error = nil, want traversal rejection")
	}
}

func TestWriteDownloadedAttachmentWritesOrdinaryFile(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "downloads")
	root, prefix, err := newDownloadRoot(outDir)
	if err != nil {
		t.Fatalf("newDownloadRoot() error = %v", err)
	}

	name, err := resolveDownloadPath(root, prefix, "screenshot.png", false)
	if err != nil {
		t.Fatalf("resolveDownloadPath() error = %v", err)
	}
	if err := writeDownloadedAttachment(root, name, []byte("payload")); err != nil {
		t.Fatalf("writeDownloadedAttachment() error = %v", err)
	}
	if got := readWebContainmentFile(t, filepath.Join(outDir, "screenshot.png")); got != "payload" {
		t.Fatalf("content = %q", got)
	}

	// Without --overwrite a second download must get a unique neighbouring name.
	second, err := resolveDownloadPath(root, prefix, "screenshot.png", false)
	if err != nil {
		t.Fatalf("resolveDownloadPath() error = %v", err)
	}
	if second == "screenshot.png" {
		t.Fatal("resolveDownloadPath() reused the existing name without --overwrite")
	}
	if err := writeDownloadedAttachment(root, second, []byte("payload-2")); err != nil {
		t.Fatalf("writeDownloadedAttachment() error = %v", err)
	}
	if got := readWebContainmentFile(t, filepath.Join(outDir, second)); got != "payload-2" {
		t.Fatalf("content = %q", got)
	}
}

func TestWritePrivacyDeclarationFileRefusesSymlinkedOutput(t *testing.T) {
	outDir := t.TempDir()
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.json")
	writeWebContainmentFile(t, sentinelPath, "{}")

	outPath := filepath.Join(outDir, "privacy.json")
	if err := os.Symlink(sentinelPath, outPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	err := writePrivacyDeclarationFile(outPath, privacyDeclarationFile{})
	if err == nil {
		t.Fatal("writePrivacyDeclarationFile() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("writePrivacyDeclarationFile() error = %v, want symlink rejection", err)
	}
	if got := readWebContainmentFile(t, sentinelPath); got != "{}" {
		t.Fatalf("sentinel content = %q, want %q", got, "{}")
	}
}

func TestWritePrivacyDeclarationFileWritesOrdinaryOutput(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "nested", "privacy.json")

	if err := writePrivacyDeclarationFile(outPath, privacyDeclarationFile{}); err != nil {
		t.Fatalf("writePrivacyDeclarationFile() error = %v", err)
	}
	info, err := os.Lstat(outPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	if got := readWebContainmentFile(t, outPath); !strings.HasSuffix(got, "\n") {
		t.Fatalf("content = %q, want trailing newline", got)
	}
}

func TestParsePrivacyDeclarationFileRefusesSymlinkedInput(t *testing.T) {
	dir := t.TempDir()
	externalPath := filepath.Join(t.TempDir(), "external.json")
	writeWebContainmentFile(t, externalPath, `{"usages":[]}`)

	linkPath := filepath.Join(dir, "privacy.json")
	if err := os.Symlink(externalPath, linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if _, err := parsePrivacyDeclarationFile(linkPath); err == nil {
		t.Fatal("parsePrivacyDeclarationFile() error = nil, want symlink rejection")
	} else if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("parsePrivacyDeclarationFile() error = %v, want symlink rejection", err)
	}
}

func TestDownloadDisplayPathPreservesOperatorPath(t *testing.T) {
	// A relative default output directory keeps its relative display form.
	relOut := filepath.Join(".asc", "web-review", "app", "sub")
	got := downloadDisplayPath(relOut, relOut, filepath.Join(relOut, "shot.png"))
	if want := filepath.Join(relOut, "shot.png"); got != want {
		t.Fatalf("downloadDisplayPath() = %q, want %q", got, want)
	}

	// An operator-selected external directory keeps its absolute display form.
	absOut := filepath.Join(string(filepath.Separator), "tmp", "downloads")
	got = downloadDisplayPath(absOut, ".", "shot-1.png")
	if want := filepath.Join(absOut, "shot-1.png"); got != want {
		t.Fatalf("downloadDisplayPath() = %q, want %q", got, want)
	}
}

func writeWebContainmentFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readWebContainmentFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}
