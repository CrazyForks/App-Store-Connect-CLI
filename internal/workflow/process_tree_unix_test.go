//go:build !windows

package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRun_TimeoutTerminatesProcessTree(t *testing.T) {
	dir := t.TempDir()
	pidPath := dir + "/child.pid"
	def, _ := loadWorkflowForRetryTest(t, fmt.Sprintf(`{
		"env": {"PID_PATH": %q},
		"workflows": {
			"main": {"steps": [{
				"name": "tree",
				"run": "sleep 10 & child=$!; printf '%%s' \"$child\" > \"$PID_PATH\"; wait \"$child\"",
				"timeout": "50ms"
			}]}
		}
	}`, pidPath))

	result, err := Run(context.Background(), def, runOpts("main"))
	if err == nil {
		t.Fatal("expected timeout")
	}
	if result.Steps[0].FailureReason != "timeout" {
		t.Fatalf("step = %+v", result.Steps[0])
	}
	data, readErr := os.ReadFile(pidPath)
	if readErr != nil {
		t.Fatalf("read child pid: %v", readErr)
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
	if parseErr != nil {
		t.Fatalf("parse child pid %q: %v", data, parseErr)
	}

	deadline := time.Now().Add(time.Second)
	for {
		killErr := syscall.Kill(pid, 0)
		if errors.Is(killErr, syscall.ESRCH) {
			break
		}
		if killErr != nil {
			t.Fatalf("probe child process %d: %v", pid, killErr)
		}
		if time.Now().After(deadline) {
			t.Fatalf("child process %d survived timeout", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
