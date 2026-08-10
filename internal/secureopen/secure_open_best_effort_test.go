package secureopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenExistingNoFollowBestEffortRejectsSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	_, err := openExistingNoFollowBestEffort(link, os.Open)
	if err == nil {
		t.Fatal("expected error when opening symlink path, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to follow symlink") {
		t.Fatalf("expected symlink refusal error, got %v", err)
	}
}

func TestOpenNewFileNoFollowBestEffortRejectsSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	_, err := openNewFileNoFollowBestEffort(link, 0o600, openNewFileWithExcl)
	if err == nil {
		t.Fatal("expected error when creating through symlink path, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to follow symlink") {
		t.Fatalf("expected symlink refusal error, got %v", err)
	}
}

func TestOpenExistingNoFollowBestEffortDetectsPathSwap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("expected"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(other, []byte("other"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := openExistingNoFollowBestEffort(path, func(string) (*os.File, error) {
		return os.Open(other)
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "file changed during open") {
		t.Fatalf("expected path swap error, got %v", err)
	}
}

func TestOpenExistingNoFollowBestEffortAllowsRegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := openExistingNoFollowBestEffort(path, os.Open)
	if err != nil {
		t.Fatalf("openExistingNoFollowBestEffort() error = %v", err)
	}
	defer file.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("file data = %q, want %q", string(data), "hello")
	}
}

func TestOpenNewFileNoFollowBestEffortCreatesRegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	file, err := openNewFileNoFollowBestEffort(path, 0o640, openNewFileWithExcl)
	if err != nil {
		t.Fatalf("openNewFileNoFollowBestEffort() error = %v", err)
	}
	if _, err := file.Write([]byte("ok")); err != nil {
		file.Close()
		t.Fatalf("Write() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("file data = %q, want %q", string(data), "ok")
	}
}

func TestOpenNewFileNoFollowBestEffortRemovesCreatedFileAfterVerificationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	otherPath := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := openNewFileNoFollowBestEffort(path, 0o600, func(path string, perm os.FileMode) (*os.File, error) {
		created, err := openNewFileWithExcl(path, perm)
		if err != nil {
			return nil, err
		}
		if err := created.Close(); err != nil {
			return nil, err
		}
		return os.Open(otherPath)
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "file changed during open") {
		t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want verification failure", err)
	}
	if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("created file remains after verification failure, Lstat() error = %v", statErr)
	}
	content, readErr := os.ReadFile(otherPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(content) != "other" {
		t.Fatalf("other file content = %q, want %q", content, "other")
	}
}

func TestOpenNewFileNoFollowBestEffortPreservesVerificationAndCleanupErrors(t *testing.T) {
	t.Run("missing destination cleanup", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "output.txt")
		otherPath := filepath.Join(dir, "other.txt")
		if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := openNewFileNoFollowBestEffort(path, 0o600, func(path string, perm os.FileMode) (*os.File, error) {
			created, err := openNewFileWithExcl(path, perm)
			if err != nil {
				return nil, err
			}
			if err := created.Close(); err != nil {
				return nil, err
			}
			if err := os.Remove(path); err != nil {
				return nil, err
			}
			return os.Open(otherPath)
		})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want errors.Is(os.ErrNotExist)", err)
		}
	})

	t.Run("close failure", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "output.txt")
		otherPath := filepath.Join(dir, "other.txt")
		if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := openNewFileNoFollowBestEffort(path, 0o600, func(path string, perm os.FileMode) (*os.File, error) {
			created, err := openNewFileWithExcl(path, perm)
			if err != nil {
				return nil, err
			}
			if err := created.Close(); err != nil {
				return nil, err
			}
			other, err := os.Open(otherPath)
			if err != nil {
				return nil, err
			}
			if err := other.Close(); err != nil {
				return nil, err
			}
			return other, nil
		})
		if !errors.Is(err, os.ErrClosed) {
			t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want errors.Is(os.ErrClosed)", err)
		}
		joined, ok := err.(interface{ Unwrap() []error })
		if !ok || len(joined.Unwrap()) != 2 {
			t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want joined verification and close errors", err)
		}
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("created file remains after close failure, Lstat() error = %v", statErr)
		}
	})

	t.Run("remove failure", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "output.txt")
		otherPath := filepath.Join(dir, "other.txt")
		if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := openNewFileNoFollowBestEffort(path, 0o600, func(path string, perm os.FileMode) (*os.File, error) {
			created, err := openNewFileWithExcl(path, perm)
			if err != nil {
				return nil, err
			}
			if err := created.Close(); err != nil {
				return nil, err
			}
			if err := os.Remove(path); err != nil {
				return nil, err
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(path, "replacement.txt"), []byte("replacement"), 0o600); err != nil {
				return nil, err
			}
			return os.Open(otherPath)
		})
		if err == nil || !strings.Contains(err.Error(), "file changed during open") {
			t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want verification failure", err)
		}
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) || pathErr.Op != "remove" || pathErr.Path != path {
			t.Fatalf("openNewFileNoFollowBestEffort() error = %v, want joined remove error for %q", err, path)
		}
	})
}

func TestOpenAppendFileNoFollowBestEffortPreservesExistingFileAfterVerificationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	otherPath := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := openAppendFileNoFollowBestEffort(path, 0o600, func(string, os.FileMode) (*os.File, error) {
		return os.Open(otherPath)
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "file changed during open") {
		t.Fatalf("openAppendFileNoFollowBestEffort() error = %v, want verification failure", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(content) != "existing" {
		t.Fatalf("existing file content = %q, want %q", content, "existing")
	}
}

func TestOpenAppendFileNoFollowBestEffortCreatesRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	file, err := openAppendFileNoFollowBestEffort(path, 0o640, func(path string, perm os.FileMode) (*os.File, error) {
		return os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm)
	})
	if err != nil {
		t.Fatalf("openAppendFileNoFollowBestEffort() error = %v", err)
	}
	if _, err := file.Write([]byte("ok")); err != nil {
		_ = file.Close()
		t.Fatalf("Write() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "ok" {
		t.Fatalf("file content = %q, want %q", content, "ok")
	}
}

func TestOpenNewFileNoFollowInRootBestEffortRemovesCreatedFileAfterVerificationFailure(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	const stagedName = ".safe-write-stage"
	const otherName = "other.txt"
	otherPath := filepath.Join(rootPath, otherName)
	if err := os.WriteFile(otherPath, []byte("other"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := openNewFileNoFollowInRootBestEffort(root, stagedName, func() (*os.File, error) {
		created, err := root.OpenFile(stagedName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, err
		}
		if err := created.Close(); err != nil {
			return nil, err
		}
		return root.Open(otherName)
	})
	if file != nil {
		_ = file.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "file changed during open") {
		t.Fatalf("openNewFileNoFollowInRootBestEffort() error = %v, want verification failure", err)
	}
	if _, statErr := root.Lstat(stagedName); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("staged file remains after verification failure, Lstat() error = %v", statErr)
	}
	content, readErr := os.ReadFile(otherPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(content) != "other" {
		t.Fatalf("other file content = %q, want %q", content, "other")
	}
}

func openNewFileWithExcl(path string, perm os.FileMode) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	return os.OpenFile(path, flags, perm)
}
