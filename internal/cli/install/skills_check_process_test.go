package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	detachedProcessHelperEnv      = "ASC_TEST_SKILLS_DETACHED_HELPER"
	detachedProcessStartedPathEnv = "ASC_TEST_SKILLS_DETACHED_STARTED"
	detachedProcessDonePathEnv    = "ASC_TEST_SKILLS_DETACHED_DONE"
)

func TestStartDetachedSkillsCheckProcessReturnsWithoutWaiting(t *testing.T) {
	if os.Getenv(detachedProcessHelperEnv) == "1" {
		startedPath := os.Getenv(detachedProcessStartedPathEnv)
		donePath := os.Getenv(detachedProcessDonePathEnv)
		if err := os.WriteFile(startedPath, []byte("started"), 0o600); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Second)
		if err := os.WriteFile(donePath, []byte("done"), 0o600); err != nil {
			os.Exit(3)
		}
		return
	}

	tmpDir := t.TempDir()
	startedPath := filepath.Join(tmpDir, "started")
	donePath := filepath.Join(tmpDir, "done")
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error: %v", err)
	}
	env := append(
		os.Environ(),
		detachedProcessHelperEnv+"=1",
		detachedProcessStartedPathEnv+"="+startedPath,
		detachedProcessDonePathEnv+"="+donePath,
	)

	startedAt := time.Now()
	err = startDetachedSkillsCheckProcess(
		executable,
		[]string{"-test.run=^TestStartDetachedSkillsCheckProcessReturnsWithoutWaiting$"},
		env,
	)
	if err != nil {
		t.Fatalf("startDetachedSkillsCheckProcess() error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 500*time.Millisecond {
		t.Fatalf("detached launcher waited %s, want under 500ms", elapsed)
	}

	waitForTestFile(t, startedPath, 5*time.Second)
	if _, err := os.Stat(donePath); err == nil {
		t.Fatal("helper completed before detached launcher returned")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat helper completion marker: %v", err)
	}
	waitForTestFile(t, donePath, 5*time.Second)
}

func TestSkillsCheckWorkerEnvironmentReplacesPrivateValues(t *testing.T) {
	spec := skillsCheckWorkerSpec{
		cachePath: "/new/cache",
		lockPath:  "/new/lock",
		token:     "new-token",
	}
	env := skillsCheckWorkerEnvironment([]string{
		"PATH=/bin",
		skillsWorkerEnvVar + "=old",
		skillsWorkerCacheEnvVar + "=/old/cache",
		skillsWorkerLockEnvVar + "=/old/lock",
		skillsWorkerTokenEnvVar + "=old-token",
	}, spec)

	want := map[string]string{
		skillsWorkerEnvVar:      "1",
		skillsWorkerCacheEnvVar: spec.cachePath,
		skillsWorkerLockEnvVar:  spec.lockPath,
		skillsWorkerTokenEnvVar: spec.token,
	}
	counts := make(map[string]int)
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		if wantValue, ok := want[key]; ok {
			counts[key]++
			if value != wantValue {
				t.Fatalf("%s = %q, want %q", key, value, wantValue)
			}
		}
	}
	for key := range want {
		if counts[key] != 1 {
			t.Fatalf("%s occurrences = %d, want 1", key, counts[key])
		}
	}
}

func waitForTestFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
