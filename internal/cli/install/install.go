package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// Pinned installer and skills source. Both are reviewed inputs: bump
// skillsInstallerPackage only after reviewing the `skills` release, and
// skillsSourceCommit only after reviewing the ASC skills diff at that commit.
//
// The upstream installer cannot consume an immutable ref itself: `skills add
// owner/repo#<ref>` resolves the ref with `git clone --depth 1 --branch <ref>`,
// which accepts only branch and tag names and rejects commit SHAs, and the
// ASC skills repository publishes no tags. The pinned commit is therefore
// materialized locally with git, which verifies object hashes, and handed to
// the installer as a local path. Nothing here falls back to a mutable ref.
const (
	skillsInstallerPackage    = "skills@1.5.20"
	skillsSourceRepositoryURL = "https://github.com/rorkai/app-store-connect-cli-skills.git"
	skillsSourceCommit        = "e30039abddbe388179324d0f9cdccb66c3843115"
)

var (
	lookupExecutable = exec.LookPath
	runCommand       = defaultRunCommand
	runCommandInDir  = defaultRunCommandInDir
	errNpxNotFound   = errors.New("npx not found")
	errGitNotFound   = errors.New("git not found")
)

// InstallSkillsCommand returns the top-level `install-skills` command.
func InstallSkillsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("install-skills", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "install-skills",
		ShortUsage: "asc install-skills",
		ShortHelp:  "Install the asc skill pack globally for App Store Connect workflows.",
		LongHelp: fmt.Sprintf(`Install the asc skill pack globally for App Store Connect workflows.

Runs the pinned installer %s against the reviewed skills commit
%s.

Both pins live in the asc source, so upgrading the skill pack is an explicit
reviewed change rather than an automatic update.

Requires Node.js (npx) and git.

Examples:
  asc install-skills`, skillsInstallerPackage, skillsSourceCommit),
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := installSkills(ctx); err != nil {
				return fmt.Errorf("install skills: %w", err)
			}
			return nil
		},
	}
}

func installSkills(ctx context.Context) error {
	ctx, cancel := shared.ContextWithTimeout(ctx)
	defer cancel()

	npxPath, err := lookupExecutable("npx")
	if err != nil {
		return fmt.Errorf("%w; install Node.js to continue", errNpxNotFound)
	}

	gitPath, err := lookupExecutable("git")
	if err != nil {
		return fmt.Errorf("%w; git is required to check out the pinned skills commit %s", errGitNotFound, skillsSourceCommit)
	}

	sourceDir, err := os.MkdirTemp("", "asc-skills-")
	if err != nil {
		return fmt.Errorf("failed to create a checkout directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(sourceDir)
	}()

	if err := checkoutPinnedSkills(ctx, gitPath, sourceDir); err != nil {
		return err
	}

	// npx prefers a local dependency whose name and version match the requested
	// package, so running it inside a caller project that carries
	// node_modules/skills at the pinned version would execute
	// repository-controlled code instead of the reviewed registry package. The
	// installer therefore runs in a fresh empty directory with no node_modules
	// in reach.
	runDir, err := isolatedInstallerDir()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(runDir)
	}()

	return runCommandInDir(ctx, runDir, npxPath, "--yes", skillsInstallerPackage, "add", sourceDir, "--global", "--agent", "codex", "--yes")
}

// npmProjectMarkers are the entries that make npm treat a directory as a
// project root: node_modules can satisfy a package specifier with a local
// dependency, and package.json or .npmrc supply project configuration such as
// a replacement registry.
var npmProjectMarkers = []string{"node_modules", "package.json", ".npmrc"}

// isolatedInstallerDir creates the working directory for the pinned installer
// and verifies that no ancestor carries an npm project marker. npm resolves
// package specifiers against local dependencies and reads project
// configuration by walking up from the working directory, so TMPDIR pointing
// inside an npm project would let repository-controlled files shadow or
// redirect the pinned registry package.
func isolatedInstallerDir() (string, error) {
	runDir, err := os.MkdirTemp("", "asc-skills-npx-")
	if err != nil {
		return "", fmt.Errorf("failed to create an isolated working directory for the installer: %w", err)
	}
	if ancestor, marker := nearestNpmProjectAncestor(runDir); ancestor != "" {
		_ = os.RemoveAll(runDir)
		return "", fmt.Errorf("temporary directory %s sits inside an npm project (%s contains %s), which could shadow or redirect the pinned installer; point TMPDIR outside any npm project and retry", runDir, ancestor, marker)
	}
	return runDir, nil
}

// nearestNpmProjectAncestor returns the closest directory at or above dir that
// contains an npm project marker together with the marker it found, or empty
// strings when there is none.
func nearestNpmProjectAncestor(dir string) (string, string) {
	for current := dir; ; {
		for _, marker := range npmProjectMarkers {
			if _, err := os.Stat(filepath.Join(current, marker)); err == nil {
				return current, marker
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", ""
		}
		current = parent
	}
}

// checkoutPinnedSkills fetches exactly skillsSourceCommit into sourceDir. Git
// validates that the received objects hash to the requested commit, so the
// checkout is verified as well as immutable.
func checkoutPinnedSkills(ctx context.Context, gitPath string, sourceDir string) error {
	steps := [][]string{
		{"init", "--quiet"},
		{"remote", "add", "origin", skillsSourceRepositoryURL},
		{"fetch", "--quiet", "--depth", "1", "origin", skillsSourceCommit},
		{"checkout", "--quiet", "--detach", skillsSourceCommit},
	}

	for _, step := range steps {
		args := append([]string{"-C", sourceDir}, step...)
		if err := runCommand(ctx, gitPath, args...); err != nil {
			return fmt.Errorf("failed to check out pinned skills commit %s from %s: %w", skillsSourceCommit, skillsSourceRepositoryURL, err)
		}
	}

	return nil
}

func defaultRunCommandInDir(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func defaultRunCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
