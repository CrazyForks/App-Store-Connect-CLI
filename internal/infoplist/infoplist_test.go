package infoplist

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCheckDeclaredSizeAtLimit(t *testing.T) {
	if err := CheckDeclaredSize(MaxBytes); err != nil {
		t.Fatalf("CheckDeclaredSize(MaxBytes) error: %v", err)
	}
}

func TestCheckDeclaredSizeOneByteOverLimit(t *testing.T) {
	err := CheckDeclaredSize(MaxBytes + 1)
	if err == nil {
		t.Fatal("expected declared-size rejection, got nil")
	}
	want := fmt.Sprintf("declared uncompressed size %d bytes exceeds the %d byte Info.plist limit", MaxBytes+1, MaxBytes)
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestReadBoundedAtLimit(t *testing.T) {
	data, err := ReadBounded(bytes.NewReader(bytes.Repeat([]byte("a"), MaxBytes)))
	if err != nil {
		t.Fatalf("ReadBounded() error: %v", err)
	}
	if len(data) != MaxBytes {
		t.Fatalf("expected %d bytes, got %d", MaxBytes, len(data))
	}
}

func TestReadBoundedOneByteOverLimit(t *testing.T) {
	_, err := ReadBounded(bytes.NewReader(bytes.Repeat([]byte("a"), MaxBytes+1)))
	if err == nil {
		t.Fatal("expected streamed-byte rejection, got nil")
	}
	want := fmt.Sprintf("expanded contents exceed the %d byte Info.plist limit", MaxBytes)
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestReadBoundedStopsShortOfEndlessStream proves the streamed bound, not the
// declared one, is what protects the reader: an endless source is refused after
// MaxBytes+1 bytes instead of being expanded until memory runs out.
func TestReadBoundedStopsShortOfEndlessStream(t *testing.T) {
	source := &countingReader{}

	_, err := ReadBounded(source)
	if err == nil {
		t.Fatal("expected streamed-byte rejection, got nil")
	}
	if !strings.Contains(err.Error(), "Info.plist limit") {
		t.Fatalf("expected Info.plist limit error, got %v", err)
	}
	if source.read > MaxBytes+1 {
		t.Fatalf("expected at most %d bytes read, got %d", MaxBytes+1, source.read)
	}
}

type countingReader struct {
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	r.read += len(p)
	return len(p), nil
}

var _ io.Reader = (*countingReader)(nil)
