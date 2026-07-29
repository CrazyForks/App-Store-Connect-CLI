package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type recordedCommand struct {
	name string
	args []string
}

func stubLookups(t *testing.T, paths map[string]string) {
	t.Helper()

	originalLookup := lookupExecutable
	t.Cleanup(func() { lookupExecutable = originalLookup })

	lookupExecutable = func(name string) (string, error) {
		path, ok := paths[name]
		if !ok {
			return "", errors.New("executable not found: " + name)
		}
		return path, nil
	}
}

func recordCommands(t *testing.T) *[]recordedCommand {
	t.Helper()

	originalRun := runCommand
	t.Cleanup(func() { runCommand = originalRun })

	recorded := &[]recordedCommand{}
	runCommand = func(ctx context.Context, name string, args ...string) error {
		*recorded = append(*recorded, recordedCommand{name: name, args: append([]string(nil), args...)})
		return nil
	}
	return recorded
}

func TestSkillsSourceIsPinnedToImmutableInput(t *testing.T) {
	if !regexp.MustCompile(`^skills@\d+\.\d+\.\d+$`).MatchString(skillsInstallerPackage) {
		t.Fatalf("skillsInstallerPackage = %q, want an exact skills@MAJOR.MINOR.PATCH pin", skillsInstallerPackage)
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(skillsSourceCommit) {
		t.Fatalf("skillsSourceCommit = %q, want a full 40-character commit SHA", skillsSourceCommit)
	}
	if !strings.HasPrefix(skillsSourceRepositoryURL, "https://github.com/rorkai/app-store-connect-cli-skills") {
		t.Fatalf("skillsSourceRepositoryURL = %q, want the reviewed ASC skills repository", skillsSourceRepositoryURL)
	}
}

func TestInstallSkillsRunsPinnedInstallerAgainstPinnedCommit(t *testing.T) {
	stubLookups(t, map[string]string{"npx": "/bin/npx", "git": "/bin/git"})
	recorded := recordCommands(t)

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}

	if len(*recorded) != 5 {
		t.Fatalf("expected 5 subprocess calls, got %d: %#v", len(*recorded), *recorded)
	}

	sourceDir := (*recorded)[0].args[1]
	if !filepath.IsAbs(sourceDir) {
		t.Fatalf("expected an absolute checkout directory, got %q", sourceDir)
	}

	want := []recordedCommand{
		{name: "/bin/git", args: []string{"-C", sourceDir, "init", "--quiet"}},
		{name: "/bin/git", args: []string{"-C", sourceDir, "remote", "add", "origin", skillsSourceRepositoryURL}},
		{name: "/bin/git", args: []string{"-C", sourceDir, "fetch", "--quiet", "--depth", "1", "origin", skillsSourceCommit}},
		{name: "/bin/git", args: []string{"-C", sourceDir, "checkout", "--quiet", "--detach", skillsSourceCommit}},
		{name: "/bin/npx", args: []string{"--yes", skillsInstallerPackage, "add", sourceDir, "--global", "--agent", "codex", "--yes"}},
	}
	if !reflect.DeepEqual(*recorded, want) {
		t.Fatalf("subprocess calls =\n%#v\nwant\n%#v", *recorded, want)
	}

	if _, err := os.Stat(sourceDir); !os.IsNotExist(err) {
		t.Fatalf("expected the pinned checkout %q to be removed, stat error: %v", sourceDir, err)
	}
}

func TestInstallSkillsNeverPassesAMutableRefToTheInstaller(t *testing.T) {
	stubLookups(t, map[string]string{"npx": "/bin/npx", "git": "/bin/git"})
	recorded := recordCommands(t)

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}

	for _, call := range *recorded {
		for _, arg := range call.args {
			for _, mutableRef := range []string{"HEAD", "FETCH_HEAD", "main", "master", "@latest", "skills"} {
				if arg == mutableRef {
					t.Fatalf("argument %q is a mutable ref or unpinned package", arg)
				}
			}
		}
	}

	installerCall := (*recorded)[len(*recorded)-1]
	if installerCall.name != "/bin/npx" {
		t.Fatalf("last subprocess = %q, want the installer", installerCall.name)
	}
	for _, arg := range installerCall.args {
		if strings.Contains(arg, "github.com") || strings.Contains(arg, "app-store-connect-cli-skills") {
			t.Fatalf("installer argument %q resolves the skills source itself instead of using the pinned checkout", arg)
		}
	}
}

func TestInstallSkillsAppliesASCTimeout(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "1m")

	stubLookups(t, map[string]string{"npx": "/bin/npx", "git": "/bin/git"})

	originalRun := runCommand
	t.Cleanup(func() { runCommand = originalRun })
	calls := 0
	runCommand = func(ctx context.Context, name string, args ...string) error {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected install command context to have a deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > time.Minute {
			t.Fatalf("expected ASC_TIMEOUT deadline within 1m, got %s", remaining)
		}
		return nil
	}

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if calls == 0 {
		t.Fatal("expected at least one subprocess call")
	}
}

func TestInstallSkillsFailsWhenNpxMissing(t *testing.T) {
	stubLookups(t, map[string]string{"git": "/bin/git"})
	recorded := recordCommands(t)

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errNpxNotFound) {
		t.Fatalf("expected npx error, got %q", err.Error())
	}
	if len(*recorded) != 0 {
		t.Fatalf("expected no subprocess calls, got %#v", *recorded)
	}
}

func TestInstallSkillsFailsWhenGitMissing(t *testing.T) {
	stubLookups(t, map[string]string{"npx": "/bin/npx"})
	recorded := recordCommands(t)

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errGitNotFound) {
		t.Fatalf("expected git error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), skillsSourceCommit) {
		t.Fatalf("expected the error to name the pinned commit, got %q", err.Error())
	}
	if len(*recorded) != 0 {
		t.Fatalf("expected no subprocess calls, got %#v", *recorded)
	}
}

func TestInstallSkillsStopsWhenPinnedCommitCannotBeFetched(t *testing.T) {
	stubLookups(t, map[string]string{"npx": "/bin/npx", "git": "/bin/git"})

	originalRun := runCommand
	t.Cleanup(func() { runCommand = originalRun })

	var recorded []recordedCommand
	runCommand = func(ctx context.Context, name string, args ...string) error {
		recorded = append(recorded, recordedCommand{name: name, args: append([]string(nil), args...)})
		if len(args) > 2 && args[2] == "fetch" {
			return errors.New("could not read from remote repository")
		}
		return nil
	}

	cmd := InstallSkillsCommand()
	if err := cmd.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), skillsSourceCommit) {
		t.Fatalf("expected the error to name the pinned commit, got %q", err.Error())
	}
	for _, call := range recorded {
		if call.name == "/bin/npx" {
			t.Fatalf("installer ran without the pinned source: %#v", call)
		}
	}
}

func TestPinnedSkillsDocumentationMatchesConstants(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "..", "README.md"),
		filepath.Join("..", "..", "..", "installation.mdx"),
		filepath.Join("..", "..", "..", "index.mdx"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(data)

		if strings.Contains(content, "npx skills add rorkai/app-store-connect-cli-skills") {
			t.Errorf("%s documents an unpinned installer and skills source", path)
		}
		if !strings.Contains(content, skillsInstallerPackage) {
			t.Errorf("%s does not mention the pinned installer %q", path, skillsInstallerPackage)
		}
		if !strings.Contains(content, skillsSourceCommit) {
			t.Errorf("%s does not mention the pinned skills commit %q", path, skillsSourceCommit)
		}
	}
}
