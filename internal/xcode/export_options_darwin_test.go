//go:build darwin

package xcode

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestCaptureBitriseStdout(t *testing.T) {
	wantErr := errors.New("generator sentinel")
	captured, err := captureBitriseStdout(func() error {
		fmt.Fprint(os.Stdout, "Checking if project uses CloudKit")
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("captureBitriseStdout() error = %v, want %v", err, wantErr)
	}
	if captured != "Checking if project uses CloudKit" {
		t.Fatalf("captureBitriseStdout() output = %q", captured)
	}
}
