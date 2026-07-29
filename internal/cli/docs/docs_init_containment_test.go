package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitReference_RefusesSymlinkedASCReference(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, ascReferenceFile)); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: false})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_RefusesSymlinkedAgentsFile(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_RefusesSymlinkedClaudeFile(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, "CLAUDE.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_RefusesLowercaseSymlinkedAgentsFile(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, "Agents.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_WritesNothingWhenAgentsFileIsSymlinked(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")
	writeDocsContainmentFile(t, filepath.Join(repo, "CLAUDE.md"), "# Claude\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ascReferenceFile)); !os.IsNotExist(statErr) {
		t.Fatalf("Lstat(ASC.md) error = %v, want IsNotExist: a failed init must not leave ASC.md behind", statErr)
	}
	if got := readDocsContainmentFile(t, filepath.Join(repo, "CLAUDE.md")); got != "# Claude\n" {
		t.Fatalf("CLAUDE.md content = %q, want unchanged", got)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_WritesNothingWhenClaudeFileIsSymlinked(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.md")
	writeDocsContainmentFile(t, sentinelPath, "# Sentinel\n")
	writeDocsContainmentFile(t, filepath.Join(repo, "AGENTS.md"), "# Agents\n")

	if err := os.Symlink(sentinelPath, filepath.Join(repo, "CLAUDE.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err == nil {
		t.Fatal("InitReference() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("InitReference() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ascReferenceFile)); !os.IsNotExist(statErr) {
		t.Fatalf("Lstat(ASC.md) error = %v, want IsNotExist: a failed init must not leave ASC.md behind", statErr)
	}
	if got := readDocsContainmentFile(t, filepath.Join(repo, "AGENTS.md")); got != "# Agents\n" {
		t.Fatalf("AGENTS.md content = %q, want unchanged", got)
	}
	if got := readDocsContainmentFile(t, sentinelPath); got != "# Sentinel\n" {
		t.Fatalf("sentinel content = %q, want unchanged", got)
	}
}

func TestInitReference_WritesOrdinaryRepositoryFiles(t *testing.T) {
	repo := newDocsContainmentRepo(t)
	writeDocsContainmentFile(t, filepath.Join(repo, "AGENTS.md"), "# Agents\n")
	writeDocsContainmentFile(t, filepath.Join(repo, "CLAUDE.md"), "# Claude\n")

	result, err := InitReference(InitOptions{Path: repo, Force: false, Link: true})
	if err != nil {
		t.Fatalf("InitReference() error = %v", err)
	}
	if !result.Created {
		t.Fatalf("InitReference() result = %#v, want created", result)
	}
	if len(result.Linked) != 2 {
		t.Fatalf("InitReference() linked = %v, want AGENTS.md and CLAUDE.md", result.Linked)
	}
	if got := readDocsContainmentFile(t, filepath.Join(repo, ascReferenceFile)); !strings.Contains(got, "asc") {
		t.Fatalf("ASC.md content = %q", got)
	}
	if got := readDocsContainmentFile(t, filepath.Join(repo, "AGENTS.md")); !strings.Contains(got, "ASC.md") {
		t.Fatalf("AGENTS.md content = %q, want ASC.md reference", got)
	}
	if got := readDocsContainmentFile(t, filepath.Join(repo, "CLAUDE.md")); !strings.Contains(got, "@ASC.md") {
		t.Fatalf("CLAUDE.md content = %q, want @ASC.md directive", got)
	}

	// A rerun with --force must still overwrite the ordinary file in place.
	rerun, err := InitReference(InitOptions{Path: repo, Force: true, Link: true})
	if err != nil {
		t.Fatalf("InitReference() rerun error = %v", err)
	}
	if !rerun.Overwritten {
		t.Fatalf("InitReference() rerun result = %#v, want overwritten", rerun)
	}
}

func newDocsContainmentRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}
	return repo
}

func writeDocsContainmentFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readDocsContainmentFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}
