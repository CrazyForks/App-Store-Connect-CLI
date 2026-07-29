package install

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

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

	return runCommand(ctx, npxPath, "--yes", skillsInstallerPackage, "add", sourceDir, "--global", "--agent", "codex", "--yes")
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

func defaultRunCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
