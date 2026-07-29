package xcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXcodeInjectRejectsParentTraversingOutputPath(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "../escaped.txt", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want traversal rejection")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escaped.txt")); statErr == nil {
		t.Fatal("output escaped the manifest root")
	}
}

func TestXcodeInjectRejectsAbsoluteOutputPath(t *testing.T) {
	dir := t.TempDir()
	externalPath := filepath.Join(t.TempDir(), "escaped.txt")
	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "`+filepath.ToSlash(externalPath)+`", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want absolute path rejection")
	}
	if _, statErr := os.Stat(externalPath); statErr == nil {
		t.Fatal("output escaped the manifest root")
	}
}

func TestXcodeInjectRejectsParentTraversingCopySource(t *testing.T) {
	dir := t.TempDir()
	secretDir := t.TempDir()
	secretPath := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	// A sibling directory of the manifest root reached with "..".
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("root level"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(nested, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "../outside.txt", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want copy source traversal rejection")
	}
	if _, statErr := os.Stat(filepath.Join(nested, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from a source outside the manifest root")
	}
}

func TestXcodeInjectRejectsAbsoluteCopySource(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "`+filepath.ToSlash(secretPath)+`", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want copy source rejection")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from an absolute source")
	}
}

func TestXcodeInjectRejectsEscapingSymlinkedOutputParent(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(dir, "Generated")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "Generated/Info.plist", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(external, "Info.plist")); statErr == nil {
		t.Fatal("output escaped through a symlinked parent")
	}
}

func TestXcodeInjectRejectsEscapingSymlinkedCopySource(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "secret.txt"), []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "Assets")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "Assets/secret.txt", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from a symlinked external source")
	}
}

func TestXcodeInjectRejectsSymlinkedCopySourceFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Symlink(secretPath, filepath.Join(dir, "Contents.json")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "Contents.json", "path": "Generated/Copied.json"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.json")); statErr == nil {
		t.Fatal("copy produced an artifact from a symlinked source file")
	}
}
