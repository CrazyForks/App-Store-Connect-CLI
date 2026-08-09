//go:build windows

package workflow

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestConfigureProcessTreeWindows(t *testing.T) {
	command := exec.Command("cmd", "/c", "exit 0")
	configureProcessTree(command)
	if command.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if command.SysProcAttr.CreationFlags&syscall.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NEW_PROCESS_GROUP", command.SysProcAttr.CreationFlags)
	}
	if !command.SysProcAttr.HideWindow {
		t.Fatal("HideWindow = false, want true")
	}
	if command.Cancel == nil {
		t.Fatal("Cancel is nil")
	}
}
