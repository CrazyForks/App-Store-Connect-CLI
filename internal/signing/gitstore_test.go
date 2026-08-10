package signing

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestEnsureInsideDir(t *testing.T) {
	baseDir := t.TempDir()

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:   "allows base directory itself",
			target: baseDir,
		},
		{
			name:   "allows child path",
			target: filepath.Join(baseDir, "nested", "file.txt"),
		},
		{
			name:    "rejects parent directory escape",
			target:  filepath.Join(baseDir, "..", "escaped.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureInsideDir(baseDir, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("EnsureInsideDir(%q, %q) expected error, got nil", baseDir, tt.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("EnsureInsideDir(%q, %q) unexpected error: %v", baseDir, tt.target, err)
			}
		})
	}
}

func TestGitStoreWriteAndReadEncryptedFileRoundTrip(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	relPath := filepath.Join("profiles", "development", "com.example.app.mobileprovision")
	plaintext := []byte("profile-data")
	password := "test-password"

	if err := store.WriteEncryptedFile(relPath, plaintext, password); err != nil {
		t.Fatalf("WriteEncryptedFile: %v", err)
	}

	encryptedPath := filepath.Join(store.LocalDir, relPath+".enc")
	encrypted, err := os.ReadFile(encryptedPath)
	if err != nil {
		t.Fatalf("read encrypted output: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("encrypted file should not match plaintext bytes")
	}

	got, err := store.ReadEncryptedFile(relPath, password)
	if err != nil {
		t.Fatalf("ReadEncryptedFile: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted output mismatch: got %q, want %q", got, plaintext)
	}
}

func TestGitStoreWriteEncryptedFileRejectsPathEscape(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}

	err := store.WriteEncryptedFile(filepath.Join("..", "escaped"), []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected path escape error, got nil")
	}
	if !strings.Contains(err.Error(), "escapes base directory") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}

func TestGitStoreWriteEncryptedFileRejectsSymlinkedParentDirectory(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	outsideDir := t.TempDir()

	if err := os.Symlink(outsideDir, filepath.Join(store.LocalDir, "linked")); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	err := store.WriteEncryptedFile(filepath.Join("linked", "secret"), []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}

	_, statErr := os.Stat(filepath.Join(outsideDir, "secret.enc"))
	if statErr == nil {
		t.Fatal("did not expect write through symlinked parent directory")
	}
	if !os.IsNotExist(statErr) {
		t.Fatalf("stat outside write target: %v", statErr)
	}
}

func TestGitStoreWriteEncryptedFileRejectsSymlinkTarget(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "secret.enc")

	if err := os.WriteFile(outsidePath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write outside target: %v", err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(store.LocalDir, "secret.enc")); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}

	err := store.WriteEncryptedFile("secret", []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}

	got, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside target: %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("did not expect write through symlink target, got %q", got)
	}
}

func TestGitStoreReadEncryptedFileRejectsSymlink(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	targetDir := t.TempDir()
	password := "test-password"

	encrypted, err := Encrypt([]byte("secret"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	targetPath := filepath.Join(targetDir, "secret.enc")
	if err := os.WriteFile(targetPath, encrypted, 0o600); err != nil {
		t.Fatalf("write target encrypted file: %v", err)
	}

	linkPath := filepath.Join(store.LocalDir, "secret.enc")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err = store.ReadEncryptedFile("secret", password)
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to read symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}
}

func TestGitStoreReadEncryptedFileRejectsSymlinkedParentDirectory(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	targetDir := t.TempDir()
	password := "test-password"

	encrypted, err := Encrypt([]byte("secret"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	targetPath := filepath.Join(targetDir, "secret.enc")
	if err := os.WriteFile(targetPath, encrypted, 0o600); err != nil {
		t.Fatalf("write target encrypted file: %v", err)
	}

	if err := os.Symlink(targetDir, filepath.Join(store.LocalDir, "linked")); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	_, err = store.ReadEncryptedFile(filepath.Join("linked", "secret"), password)
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}
}

func TestGitStoreListEncryptedFilesSkipsGitDirAndSymlinks(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}

	if err := os.MkdirAll(filepath.Join(store.LocalDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(store.LocalDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	write := func(rel string) {
		t.Helper()
		path := filepath.Join(store.LocalDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	write("root.enc")
	write(filepath.Join("nested", "child.enc"))
	write(filepath.Join(".git", "ignored.enc"))

	if err := os.Symlink(filepath.Join(store.LocalDir, "root.enc"), filepath.Join(store.LocalDir, "symlink.enc")); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}
	if err := os.Symlink(filepath.Join(store.LocalDir, "nested"), filepath.Join(store.LocalDir, "linked-dir")); err != nil {
		t.Fatalf("create dir symlink: %v", err)
	}

	got, err := store.ListEncryptedFiles()
	if err != nil {
		t.Fatalf("ListEncryptedFiles: %v", err)
	}

	gotSet := map[string]bool{}
	for _, rel := range got {
		gotSet[filepath.ToSlash(rel)] = true
	}

	if !gotSet["root"] {
		t.Fatalf("expected root file in list, got %v", got)
	}
	if !gotSet["nested/child"] {
		t.Fatalf("expected nested file in list, got %v", got)
	}
	if gotSet[".git/ignored"] {
		t.Fatalf("did not expect .git file in list, got %v", got)
	}
	if gotSet["symlink"] {
		t.Fatalf("did not expect symlink file in list, got %v", got)
	}
	if gotSet["linked-dir/child"] {
		t.Fatalf("did not expect symlinked directory contents in list, got %v", got)
	}
}

func TestGitStoreGitHelpersUseNonInteractiveExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	gitCapture := filepath.Join(t.TempDir(), "git-env.txt")
	sshCapture := filepath.Join(t.TempDir(), "ssh-args.txt")

	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
if [ "${1-}" = "config" ] && [ "${2-}" = "--get" ] && [ "${3-}" = "core.sshCommand" ]; then
  exit 1
fi
printf '%s|%s\n' "${GIT_TERMINAL_PROMPT-}" "${GIT_SSH_COMMAND-}" >> "$ASC_FAKE_GIT_CAPTURE"
sh -c "$GIT_SSH_COMMAND \"\$@\"" asc-fake-git "$@"
if [ "${1-}" = "status" ]; then
  printf 'clean\n'
fi
`)
	writeTestExecutable(t, filepath.Join(binDir, "ssh"), `#!/bin/sh
set -eu
{
  printf '%s\n' '--call--'
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} >> "$ASC_FAKE_SSH_CAPTURE"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASC_FAKE_GIT_CAPTURE", gitCapture)
	t.Setenv("ASC_FAKE_SSH_CAPTURE", sshCapture)
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("GIT_SSH_COMMAND", "")

	store := &GitStore{}
	if err := store.gitRun(context.Background(), "", "clone", "source", "destination"); err != nil {
		t.Fatalf("gitRun: %v", err)
	}
	output, err := store.gitOutput(context.Background(), "", "status", "--porcelain")
	if err != nil {
		t.Fatalf("gitOutput: %v", err)
	}
	if output != "clean\n" {
		t.Fatalf("gitOutput() = %q, want %q", output, "clean\n")
	}

	gitEnvironment, err := os.ReadFile(gitCapture)
	if err != nil {
		t.Fatalf("read fake git environment: %v", err)
	}
	if got, want := string(gitEnvironment), "0|ssh -o BatchMode=yes\n0|ssh -o BatchMode=yes\n"; got != want {
		t.Fatalf("fake git environment = %q, want %q", got, want)
	}

	sshArguments, err := os.ReadFile(sshCapture)
	if err != nil {
		t.Fatalf("read fake ssh arguments: %v", err)
	}
	wantCalls := "--call--\n-o\nBatchMode=yes\nclone\nsource\ndestination\n" +
		"--call--\n-o\nBatchMode=yes\nstatus\n--porcelain\n"
	if got := string(sshArguments); got != wantCalls {
		t.Fatalf("fake ssh arguments = %q, want %q", got, wantCalls)
	}
}

func TestNewGitCommandUsesNonInteractiveEnvironment(t *testing.T) {
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("GIT_SSH_COMMAND", " ")
	t.Setenv("GIT_SSH", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-gitconfig"))

	dir := t.TempDir()
	cmd, err := newGitCommand(context.Background(), dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("newGitCommand: %v", err)
	}

	if cmd.Dir != dir {
		t.Fatalf("newGitCommand() dir = %q, want %q", cmd.Dir, dir)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_TERMINAL_PROMPT"); len(got) != 1 || got[0] != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT values = %v, want [0]", got)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_SSH_COMMAND"); len(got) != 1 || got[0] != "ssh -o BatchMode=yes" {
		t.Fatalf("GIT_SSH_COMMAND values = %v, want [ssh -o BatchMode=yes]", got)
	}
}

func TestNewGitCommandPreservesGitSSHTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	transportCapture := filepath.Join(t.TempDir(), "transport-args.txt")
	transport := filepath.Join(t.TempDir(), "team-ssh")
	writeTestExecutable(t, transport, `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ASC_FAKE_SSH_CAPTURE"
exit 17
`)

	t.Setenv("ASC_FAKE_SSH_CAPTURE", transportCapture)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-gitconfig"))
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", transport)
	t.Setenv("GIT_SSH_VARIANT", "ssh")

	cmd, err := newGitCommand(context.Background(), "", "ls-remote", "ssh://git@127.0.0.1:1/repository")
	if err != nil {
		t.Fatalf("newGitCommand: %v", err)
	}
	if err := cmd.Run(); err == nil {
		t.Fatal("expected fake SSH transport failure")
	}
	if _, err := os.Stat(transportCapture); err != nil {
		t.Fatalf("expected inherited GIT_SSH transport to run: %v", err)
	}
}

func TestNewGitCommandPreservesConfiguredSSHTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	tests := []struct {
		name      string
		configure func(t *testing.T, transport string) string
	}{
		{
			name: "global config",
			configure: func(t *testing.T, transport string) string {
				configPath := filepath.Join(t.TempDir(), "gitconfig")
				configureGitSSHCommand(t, configPath, transport)
				t.Setenv("GIT_CONFIG_GLOBAL", configPath)
				return t.TempDir()
			},
		},
		{
			name: "repository config",
			configure: func(t *testing.T, transport string) string {
				t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-gitconfig"))
				repositoryDir := t.TempDir()
				runTestGit(t, "init", "--quiet", repositoryDir)
				runTestGit(t, "-C", repositoryDir, "config", "core.sshCommand", transport)
				return repositoryDir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneEnvironment := standaloneTestGitEnvironment(t)
			sentinelRepository := filepath.Join(t.TempDir(), "sentinel")
			runStandaloneTestGit(t, standaloneEnvironment, "init", "--quiet", sentinelRepository)
			transportCapture := filepath.Join(t.TempDir(), "transport-args.txt")
			transport := filepath.Join(t.TempDir(), "configured-ssh")
			writeTestExecutable(t, transport, `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ASC_FAKE_SSH_CAPTURE"
exit 17
`)

			t.Setenv("ASC_FAKE_SSH_CAPTURE", transportCapture)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_SSH_COMMAND", "")
			t.Setenv("GIT_SSH", "")
			t.Setenv("GIT_SSH_VARIANT", "ssh")
			t.Setenv("GIT_DIR", filepath.Join(sentinelRepository, ".git"))
			t.Setenv("GIT_WORK_TREE", sentinelRepository)
			t.Setenv("GIT_COMMON_DIR", filepath.Join(sentinelRepository, ".git"))
			dir := tt.configure(t, transport)

			cmd, err := newGitCommand(context.Background(), dir, "ls-remote", "ssh://git@127.0.0.1:1/repository")
			if err != nil {
				t.Fatalf("newGitCommand: %v", err)
			}
			if err := cmd.Run(); err == nil {
				t.Fatal("expected fake SSH transport failure")
			}
			if _, err := os.Stat(transportCapture); err != nil {
				t.Fatalf("expected configured SSH transport to run: %v", err)
			}
		})
	}
}

func TestNewGitCommandDefaultsForBlankConfiguredSSHCommand(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gitconfig")
	configureGitSSHCommand(t, configPath, "")

	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")

	cmd, err := newGitCommand(context.Background(), t.TempDir(), "clone", "source", "destination")
	if err != nil {
		t.Fatalf("newGitCommand: %v", err)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_SSH_COMMAND"); len(got) != 1 || got[0] != "ssh -o BatchMode=yes" {
		t.Fatalf("GIT_SSH_COMMAND values = %v, want [ssh -o BatchMode=yes]", got)
	}
}

func TestNewGitCommandFailsClosedWhenSSHConfigLookupFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	mainCommandCapture := filepath.Join(t.TempDir(), "main-command.txt")
	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
if [ "${1-}" = "config" ] && [ "${2-}" = "--get" ] && [ "${3-}" = "core.sshCommand" ]; then
  printf 'invalid git configuration\n' >&2
  exit 2
fi
printf 'invoked\n' > "$ASC_FAKE_GIT_CAPTURE"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASC_FAKE_GIT_CAPTURE", mainCommandCapture)
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")

	cmd, err := newGitCommand(context.Background(), "", "clone", "source", "destination")
	if err == nil {
		t.Fatalf("newGitCommand() = %v, want SSH config lookup error", cmd)
	}
	if !strings.Contains(err.Error(), "core.sshCommand") {
		t.Fatalf("newGitCommand() error = %q, want core.sshCommand context", err)
	}
	if _, statErr := os.Stat(mainCommandCapture); !os.IsNotExist(statErr) {
		t.Fatalf("network-capable Git command ran after lookup failure: %v", statErr)
	}
}

func TestNewGitCommandIgnoresInheritedRepositorySelectors(t *testing.T) {
	sentinelRepository := filepath.Join(t.TempDir(), "sentinel")
	targetRepository := filepath.Join(t.TempDir(), "target")
	standaloneEnvironment := standaloneTestGitEnvironment(t)
	runStandaloneTestGit(t, standaloneEnvironment, "init", "--quiet", sentinelRepository)
	runStandaloneTestGit(t, standaloneEnvironment, "init", "--quiet", targetRepository)

	globalConfig := filepath.Join(t.TempDir(), "global-config")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "missing-system-config"))
	t.Setenv("GIT_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_WORK_TREE", sentinelRepository)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(sentinelRepository, ".git", "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(sentinelRepository, ".git", "objects"))
	t.Setenv("GIT_SSH", "/opt/team/bin/ssh-wrapper")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")

	cmd, err := newGitCommand(context.Background(), targetRepository, "config", "core.testMarker", "target-command")
	if err != nil {
		t.Fatalf("newGitCommand: %v", err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("run Git command: %v", err)
	}

	targetValue, targetConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, targetRepository, "core.testMarker")
	if !targetConfigured || targetValue != "target-command" {
		t.Errorf("target core.testMarker = %q, configured = %t; want target-command in target repository", targetValue, targetConfigured)
	}
	sentinelValue, sentinelConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, sentinelRepository, "core.testMarker")
	if sentinelConfigured {
		t.Errorf("sentinel core.testMarker = %q; inherited repository selectors redirected Git", sentinelValue)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_DIR"); len(got) != 0 {
		t.Errorf("GIT_DIR values = %v, want none", got)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_CONFIG_GLOBAL"); len(got) != 1 || got[0] != globalConfig {
		t.Errorf("GIT_CONFIG_GLOBAL values = %v, want [%s]", got, globalConfig)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_SSH"); len(got) != 1 || got[0] != "/opt/team/bin/ssh-wrapper" {
		t.Errorf("GIT_SSH values = %v, want preserved transport", got)
	}
	if got := commandEnvironmentValues(cmd.Env, "GIT_TERMINAL_PROMPT"); len(got) != 1 || got[0] != "0" {
		t.Errorf("GIT_TERMINAL_PROMPT values = %v, want [0]", got)
	}
}

func TestRunTestGitIsolatesRepositoryEnvironment(t *testing.T) {
	sentinelRepository := filepath.Join(t.TempDir(), "sentinel")
	targetRepository := filepath.Join(t.TempDir(), "target")
	standaloneEnvironment := standaloneTestGitEnvironment(t)
	runStandaloneTestGit(t, standaloneEnvironment, "init", "--quiet", sentinelRepository)
	runStandaloneTestGit(t, standaloneEnvironment, "init", "--quiet", targetRepository)

	t.Setenv("GIT_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_WORK_TREE", sentinelRepository)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-global-config"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "missing-system-config"))
	t.Setenv("HOME", t.TempDir())
	runTestGit(t, "-C", targetRepository, "config", "core.sshCommand", "target-command")

	targetValue, targetConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, targetRepository, "core.sshCommand")
	if !targetConfigured || targetValue != "target-command" {
		t.Errorf("target core.sshCommand = %q, configured = %t; want target-command in target repository", targetValue, targetConfigured)
	}
	sentinelValue, sentinelConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, sentinelRepository, "core.sshCommand")
	if sentinelConfigured {
		t.Fatalf("sentinel core.sshCommand = %q; inherited repository selectors escaped test isolation", sentinelValue)
	}
}

func TestGitCommandEnvironmentPreservesCustomSSHCommand(t *testing.T) {
	tests := []struct {
		name       string
		sshCommand string
	}{
		{
			name:       "quoted identity path with batch mode",
			sshCommand: `ssh -i '/tmp/team key' -o BatchMode=yes`,
		},
		{
			name:       "wrapper command",
			sshCommand: `team-ssh-wrapper --identity 'release key'`,
		},
		{
			name:       "explicit interactive override",
			sshCommand: `ssh -o BatchMode=no`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := gitCommandEnvironment([]string{
				"PATH=/usr/bin",
				"GIT_TERMINAL_PROMPT=1",
				"GIT_TERMINAL_PROMPT=true",
				"GIT_SSH_COMMAND=" + tt.sshCommand,
			})

			if got := commandEnvironmentValues(environment, "GIT_TERMINAL_PROMPT"); len(got) != 1 || got[0] != "0" {
				t.Fatalf("GIT_TERMINAL_PROMPT values = %v, want [0]", got)
			}
			if got := commandEnvironmentValues(environment, "GIT_SSH_COMMAND"); len(got) != 1 || got[0] != tt.sshCommand {
				t.Fatalf("GIT_SSH_COMMAND values = %v, want [%s]", got, tt.sshCommand)
			}
		})
	}
}

func TestGitCommandEnvironmentPreservesGitSSH(t *testing.T) {
	environment := gitCommandEnvironment([]string{
		"PATH=/usr/bin",
		"GIT_TERMINAL_PROMPT=1",
		"GIT_SSH_COMMAND= \t",
		"GIT_SSH=/opt/team/bin/ssh-wrapper",
	})
	want := []string{
		"PATH=/usr/bin",
		"GIT_SSH=/opt/team/bin/ssh-wrapper",
		"GIT_TERMINAL_PROMPT=0",
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitCommandEnvironment() = %v, want %v", environment, want)
	}
}

func TestGitCommandEnvironmentDefaultsMissingOrBlankSSHCommand(t *testing.T) {
	tests := []struct {
		name        string
		environment []string
	}{
		{
			name:        "missing",
			environment: []string{"PATH=/usr/bin"},
		},
		{
			name:        "empty",
			environment: []string{"PATH=/usr/bin", "GIT_SSH_COMMAND="},
		},
		{
			name:        "whitespace",
			environment: []string{"PATH=/usr/bin", "GIT_SSH_COMMAND= \t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := gitCommandEnvironment(tt.environment)
			if got := commandEnvironmentValues(environment, "GIT_SSH_COMMAND"); len(got) != 1 || got[0] != "ssh -o BatchMode=yes" {
				t.Fatalf("GIT_SSH_COMMAND values = %v, want [ssh -o BatchMode=yes]", got)
			}
		})
	}
}

func TestGitCommandEnvironmentForWindowsMatchesKeysCaseInsensitively(t *testing.T) {
	environment := gitCommandEnvironmentForGOOS([]string{
		"PATH=C:\\Windows\\System32",
		"git_terminal_prompt=1",
		`git_ssh_command=team-ssh-wrapper --identity 'release key'`,
	}, "windows")
	want := []string{
		"PATH=C:\\Windows\\System32",
		"GIT_TERMINAL_PROMPT=0",
		`GIT_SSH_COMMAND=team-ssh-wrapper --identity 'release key'`,
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitCommandEnvironmentForGOOS() = %v, want %v", environment, want)
	}
}

func TestGitCommandEnvironmentForWindowsPreservesGitSSH(t *testing.T) {
	environment := gitCommandEnvironmentForGOOS([]string{
		"PATH=C:\\Windows\\System32",
		"git_terminal_prompt=1",
		"git_ssh_command= ",
		`git_ssh=C:\tools\team-ssh.exe`,
	}, "windows")
	want := []string{
		"PATH=C:\\Windows\\System32",
		`git_ssh=C:\tools\team-ssh.exe`,
		"GIT_TERMINAL_PROMPT=0",
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitCommandEnvironmentForGOOS() = %v, want %v", environment, want)
	}
}

func TestGitEnvironmentWithoutRepositorySelectorsMatchesWindowsKeysCaseInsensitively(t *testing.T) {
	environment := gitEnvironmentWithoutRepositorySelectors([]string{
		"PATH=C:\\Windows\\System32",
		`git_dir=C:\sensitive\repo\.git`,
		`Git_Work_Tree=C:\sensitive\repo`,
		`GIT_COMMON_DIR=C:\sensitive\repo\.git`,
		`GIT_CONFIG_GLOBAL=C:\config\gitconfig`,
		`GIT_SSH=C:\tools\team-ssh.exe`,
	}, "windows")
	want := []string{
		"PATH=C:\\Windows\\System32",
		`GIT_CONFIG_GLOBAL=C:\config\gitconfig`,
		`GIT_SSH=C:\tools\team-ssh.exe`,
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitEnvironmentWithoutRepositorySelectors() = %v, want %v", environment, want)
	}
}

func TestGitCommandEnvironmentForPOSIXKeepsKeysCaseSensitive(t *testing.T) {
	environment := gitCommandEnvironmentForGOOS([]string{
		"PATH=/usr/bin",
		"git_terminal_prompt=1",
		"git_ssh_command=team-ssh-wrapper",
	}, "linux")
	want := []string{
		"PATH=/usr/bin",
		"git_terminal_prompt=1",
		"git_ssh_command=team-ssh-wrapper",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitCommandEnvironmentForGOOS() = %v, want %v", environment, want)
	}
}

func commandEnvironmentValues(environment []string, key string) []string {
	prefix := key + "="
	var values []string
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values
}

func writeTestExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func configureGitSSHCommand(t *testing.T, configPath, command string) {
	t.Helper()
	runTestGit(t, "config", "--file", configPath, "core.sshCommand", command)
}

func runTestGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = t.TempDir()
	cmd.Env = standaloneTestGitEnvironment(t)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func standaloneTestGitEnvironment(t *testing.T) []string {
	t.Helper()
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "missing-global-config"),
		"GIT_CONFIG_SYSTEM=" + filepath.Join(t.TempDir(), "missing-system-config"),
	}
}

func runStandaloneTestGit(t *testing.T, environment []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = t.TempDir()
	cmd.Env = environment
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func standaloneTestGitConfigValue(t *testing.T, environment []string, repository, key string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", filepath.Join(repository, ".git"), "config", "--get", key)
	cmd.Dir = t.TempDir()
	cmd.Env = environment
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), true
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", false
	}
	t.Fatalf("standalone git config --get %s: %v", key, err)
	return "", false
}
