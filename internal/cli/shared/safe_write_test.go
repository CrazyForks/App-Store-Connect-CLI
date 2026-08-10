package shared

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeWriteFileNoSymlinkNoOverwriteRemovesPartialFileAfterCallbackFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	writeErr := errors.New("simulated write failure")

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("partial"))
			if err != nil {
				return int64(written), err
			}
			return int64(written), writeErr
		},
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want %v", err, writeErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected partial destination to be removed, stat error = %v", err)
	}
}

func TestCleanupIncompleteFilePreservesPrimaryAndCleanupErrors(t *testing.T) {
	primaryErr := errors.New("sync failure")
	closeErr := errors.New("close failure")
	removeErr := errors.New("remove failure")
	closeCalled := false
	removeCalled := false

	err := cleanupIncompleteFile(
		"artifact.bin",
		primaryErr,
		func() error {
			closeCalled = true
			return closeErr
		},
		func(path string) error {
			if !closeCalled {
				t.Fatal("remove called before close")
			}
			if path != "artifact.bin" {
				t.Fatalf("remove path = %q, want artifact.bin", path)
			}
			removeCalled = true
			return removeErr
		},
	)
	if !closeCalled {
		t.Fatal("close was not called")
	}
	if !removeCalled {
		t.Fatal("remove was not called")
	}
	for _, wantErr := range []error{primaryErr, closeErr, removeErr} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("cleanupIncompleteFile() error = %v, want errors.Is(_, %v)", err, wantErr)
		}
	}
}

func TestWriteNewFileNoSymlinkRemovesFileAfterSyncFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	file, err := OpenNewFileNoFollow(destination, 0o600)
	if err != nil {
		t.Fatalf("OpenNewFileNoFollow() error = %v", err)
	}
	syncErr := errors.New("simulated sync failure")

	_, err = writeNewFileNoSymlink(
		destination,
		file,
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			return int64(written), err
		},
		newFileWriteOps{
			syncFile:   func() error { return syncErr },
			closeFile:  file.Close,
			removeFile: os.Remove,
		},
	)
	if !errors.Is(err, syncErr) {
		t.Fatalf("writeNewFileNoSymlink() error = %v, want %v", err, syncErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete destination to be removed, stat error = %v", err)
	}
}

func TestWriteNewFileNoSymlinkRemovesFileAfterCloseFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	file, err := OpenNewFileNoFollow(destination, 0o600)
	if err != nil {
		t.Fatalf("OpenNewFileNoFollow() error = %v", err)
	}
	closeErr := errors.New("simulated close failure")

	_, err = writeNewFileNoSymlink(
		destination,
		file,
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			return int64(written), err
		},
		newFileWriteOps{
			syncFile: file.Sync,
			closeFile: func() error {
				if err := file.Close(); err != nil {
					return err
				}
				return closeErr
			},
			removeFile: os.Remove,
		},
	)
	if !errors.Is(err, closeErr) {
		t.Fatalf("writeNewFileNoSymlink() error = %v, want %v", err, closeErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete destination to be removed, stat error = %v", err)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteCreatesFileOnSuccess(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	content := []byte("complete")

	written, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write(content)
			return int64(written), err
		},
	)
	if err != nil {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("SafeWriteFileNoSymlink() written = %d, want %d", written, len(content))
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("destination content = %q, want %q", got, content)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwritePreservesExistingFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	original := []byte("original")
	if err := os.WriteFile(destination, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	callbackCalled := false
	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			callbackCalled = true
			written, err := file.Write([]byte("replacement"))
			return int64(written), err
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrExist", err)
	}
	if callbackCalled {
		t.Fatal("write callback was called for an existing destination")
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("destination content = %q, want %q", got, original)
	}
}
