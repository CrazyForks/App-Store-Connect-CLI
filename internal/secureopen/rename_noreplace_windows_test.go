//go:build windows

package secureopen

import (
	"testing"
	"unsafe"
)

func TestBuildFileRenameInformationUsesSameDirectoryNoReplace(t *testing.T) {
	buffer, err := buildFileRenameInformation("artifact.bin")
	if err != nil {
		t.Fatalf("buildFileRenameInformation() error = %v", err)
	}
	info := (*fileRenameInformation)(unsafe.Pointer(&buffer[0]))
	if info.ReplaceIfExists != 0 {
		t.Fatalf("ReplaceIfExists = %d, want 0", info.ReplaceIfExists)
	}
	if info.RootDirectory != 0 {
		t.Fatalf("RootDirectory = %v, want null for same-directory and SMB renames", info.RootDirectory)
	}
	if info.FileNameLength != uint32(len("artifact.bin")*2) {
		t.Fatalf("FileNameLength = %d, want %d", info.FileNameLength, len("artifact.bin")*2)
	}
}
