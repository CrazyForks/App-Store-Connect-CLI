package docs

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/rootfs"
)

const ascReferenceFile = "ASC.md"

var (
	// ErrASCReferenceExists indicates ASC.md already exists and --force was not set.
	ErrASCReferenceExists = errors.New("ASC.md already exists")
	// ErrInvalidASCReferencePath indicates --path does not target ASC.md or a directory.
	ErrInvalidASCReferencePath = errors.New("path must target ASC.md or a directory")
)

// InitOptions controls ASC reference generation.
type InitOptions struct {
	Path  string
	Force bool
	Link  bool
}

// InitResult describes the output of an init run.
type InitResult struct {
	Path        string   `json:"path"`
	Created     bool     `json:"created"`
	Overwritten bool     `json:"overwritten"`
	Linked      []string `json:"linked,omitempty"`
}

// NewInitReferenceCommand builds an init-style command that writes ASC.md references.
func NewInitReferenceCommand(flagSetName, commandName, shortUsage, shortHelp, longHelp, errorPrefix string) *ffcli.Command {
	fs := flag.NewFlagSet(flagSetName, flag.ExitOnError)

	path := fs.String("path", "", "Output path for ASC.md (default: repo root or current directory)")
	force := fs.Bool("force", false, "Overwrite existing ASC.md")
	link := fs.Bool("link", true, "Update AGENTS.md and CLAUDE.md to reference ASC.md")

	return &ffcli.Command{
		Name:       commandName,
		ShortUsage: shortUsage,
		ShortHelp:  shortHelp,
		LongHelp:   longHelp,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			result, err := InitReference(InitOptions{
				Path:  *path,
				Force: *force,
				Link:  *link,
			})
			if err != nil {
				return fmt.Errorf("%s: %w", errorPrefix, err)
			}
			return shared.PrintOutput(result, "json", false)
		},
	}
}

// DocsInitCommand returns the docs init subcommand.
func DocsInitCommand() *ffcli.Command {
	return NewInitReferenceCommand(
		"docs init",
		"init",
		"asc docs init [flags]",
		"Create an ASC.md command reference for the asc cli in the current repo.",
		`Create an ASC.md command reference for the asc cli in the current repo.

Examples:
  asc docs init
  asc docs init --path ./ASC.md
  asc docs init --force --link=false`,
		"docs init",
	)
}

// InitReference generates ASC.md in the target repo and links agent files.
// Every validation, including the symlink containment checks on the ASC.md
// destination and the agent files, runs before the first write, so a failed
// init leaves the repository untouched.
func InitReference(opts InitOptions) (InitResult, error) {
	targetPath, linkRoot, err := resolveOutputPath(opts.Path)
	if err != nil {
		return InitResult{}, err
	}

	ascPlan, err := planASCReference(targetPath, opts.Force)
	if err != nil {
		return InitResult{}, err
	}

	linkPlan := agentLinkPlan{}
	if opts.Link {
		relRef, err := filepath.Rel(linkRoot, targetPath)
		if err != nil {
			relRef = ascReferenceFile
		}
		relRef = normalizeReferencePath(relRef)
		linkPlan, err = planAgentFileLinks(linkRoot, relRef)
		if err != nil {
			return InitResult{}, err
		}
	}

	created, overwritten, err := writeASCReference(ascPlan)
	if err != nil {
		return InitResult{}, err
	}

	linked, err := applyAgentFileLinks(linkPlan)
	if err != nil {
		return InitResult{}, err
	}

	return InitResult{
		Path:        targetPath,
		Created:     created,
		Overwritten: overwritten,
		Linked:      linked,
	}, nil
}

func resolveOutputPath(path string) (string, string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed != "" {
		abs, err := filepath.Abs(trimmed)
		if err != nil {
			return "", "", err
		}
		targetPath := ""
		linkBase := ""
		if info, err := os.Stat(abs); err == nil {
			if info.IsDir() {
				targetPath = filepath.Join(abs, ascReferenceFile)
				linkBase = abs
			} else if looksLikeMarkdown(abs) {
				if !isASCReferencePath(abs) {
					return "", "", fmt.Errorf("%w: %s", ErrInvalidASCReferencePath, abs)
				}
				targetPath = abs
				linkBase = filepath.Dir(abs)
			} else {
				return "", "", fmt.Errorf("%w: %s is not a directory or markdown file", ErrInvalidASCReferencePath, abs)
			}
		} else if !os.IsNotExist(err) {
			return "", "", err
		} else if looksLikeMarkdown(abs) || hasFileExtension(abs) {
			if !isASCReferencePath(abs) {
				return "", "", fmt.Errorf("%w: %s", ErrInvalidASCReferencePath, abs)
			}
			targetPath = abs
			linkBase = filepath.Dir(abs)
		} else {
			targetPath = filepath.Join(abs, ascReferenceFile)
			linkBase = abs
		}
		root, err := findRepoRoot(linkBase)
		if err != nil {
			return "", "", err
		}
		if root == "" {
			root = linkBase
		}
		return targetPath, root, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	root, err := findRepoRoot(cwd)
	if err != nil {
		return "", "", err
	}
	if root == "" {
		root = cwd
	}
	return filepath.Join(root, ascReferenceFile), root, nil
}

func looksLikeMarkdown(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(strings.ToLower(base), ".md")
}

func hasFileExtension(path string) bool {
	return filepath.Ext(filepath.Base(path)) != ""
}

func isASCReferencePath(path string) bool {
	return strings.EqualFold(filepath.Base(path), ascReferenceFile)
}

func normalizeReferencePath(path string) string {
	trimmed := strings.TrimSpace(filepath.ToSlash(path))
	if trimmed == "" || trimmed == "." {
		return ascReferenceFile
	}
	return trimmed
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if dir == "" {
			return "", nil
		}
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// ascReferencePlan is a fully validated, not-yet-applied ASC.md write.
type ascReferencePlan struct {
	root   rootfs.Root
	name   string
	exists bool
}

// planASCReference validates the ASC.md destination without writing anything.
func planASCReference(path string, force bool) (ascReferencePlan, error) {
	root, err := rootfs.New(filepath.Dir(path))
	if err != nil {
		return ascReferencePlan{}, err
	}
	name := filepath.Base(path)

	// Lstat, not Stat, so a dangling symlink still counts as an existing entry
	// and is never followed.
	exists := false
	if _, err := os.Lstat(path); err == nil {
		exists = true
	} else if !os.IsNotExist(err) {
		return ascReferencePlan{}, err
	}

	if exists && !force {
		return ascReferencePlan{}, fmt.Errorf("%w: %s (use --force to overwrite)", ErrASCReferenceExists, path)
	}

	// Reject a symlinked destination while planning so the failure happens
	// before any file is written; the rooted write re-checks at write time.
	if err := root.CheckContained(name); err != nil {
		return ascReferencePlan{}, err
	}

	return ascReferencePlan{root: root, name: name, exists: exists}, nil
}

func writeASCReference(plan ascReferencePlan) (bool, bool, error) {
	content := ascTemplate
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := plan.root.WriteFile(plan.name, []byte(content), 0o644); err != nil {
		return false, false, err
	}

	if plan.exists {
		return false, true, nil
	}
	return true, false, nil
}

// agentFileUpdate is one planned agent-file rewrite.
type agentFileUpdate struct {
	name    string
	content string
}

// agentLinkPlan holds every agent-file update computed during planning.
type agentLinkPlan struct {
	root    rootfs.Root
	rootDir string
	updates []agentFileUpdate
}

// planAgentFileLinks computes every agent-file update up front, so a symlinked
// or unreadable agent file is rejected before anything is written.
func planAgentFileLinks(rootDir string, relRef string) (agentLinkPlan, error) {
	root, err := rootfs.New(rootDir)
	if err != nil {
		return agentLinkPlan{}, err
	}
	plan := agentLinkPlan{root: root, rootDir: rootDir}

	agentsName := "AGENTS.md"
	if !entryExists(filepath.Join(rootDir, agentsName)) {
		agentsName = "Agents.md"
	}
	agentsContent, agentsChanged, err := planAgentsLink(root, agentsName, relRef)
	if err != nil {
		return agentLinkPlan{}, err
	}
	if agentsChanged {
		plan.updates = append(plan.updates, agentFileUpdate{name: agentsName, content: agentsContent})
	}

	claudeName := "CLAUDE.md"
	claudeContent, claudeChanged, err := planClaudeLink(root, claudeName, relRef)
	if err != nil {
		return agentLinkPlan{}, err
	}
	if claudeChanged {
		plan.updates = append(plan.updates, agentFileUpdate{name: claudeName, content: claudeContent})
	}

	return plan, nil
}

func applyAgentFileLinks(plan agentLinkPlan) ([]string, error) {
	linked := []string{}
	for _, update := range plan.updates {
		if err := plan.root.WriteFile(update.name, []byte(update.content), 0o644); err != nil {
			return nil, err
		}
		linked = append(linked, filepath.Join(plan.rootDir, update.name))
	}
	return linked, nil
}

// entryExists reports whether path exists without following a final symlink, so
// a symlinked agent file is still selected and then rejected by the rooted read.
func entryExists(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Lstat(path); err == nil {
		return true
	}
	return false
}

// planAgentsLink computes the updated AGENTS.md content without writing it.
func planAgentsLink(root rootfs.Root, name string, relRef string) (string, bool, error) {
	data, found, err := root.ReadFileOptional(name)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}

	desiredLine := fmt.Sprintf("See `%s` for the command catalog and workflows.", relRef)

	lines := strings.Split(string(data), "\n")
	foundReference := false
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isAgentsReferenceLine(trimmed) {
			continue
		}
		if foundReference {
			lines[i] = ""
			changed = true
			continue
		}
		foundReference = true
		if line != desiredLine {
			lines[i] = desiredLine
			changed = true
		}
	}
	if foundReference {
		if !changed {
			return "", false, nil
		}
		return plannedContent(string(data), strings.Join(lines, "\n"))
	}

	section := fmt.Sprintf("## asc cli reference\n\n%s", desiredLine)
	return plannedContent(string(data), appendSection(string(data), section))
}

// planClaudeLink computes the updated CLAUDE.md content without writing it.
func planClaudeLink(root rootfs.Root, name string, relRef string) (string, bool, error) {
	data, found, err := root.ReadFileOptional(name)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}

	desiredLine := "@" + relRef

	lines := strings.Split(string(data), "\n")
	updatedLines := make([]string, 0, len(lines))
	foundReference := false
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !isASCReferenceDirective(trimmed) {
			updatedLines = append(updatedLines, line)
			continue
		}
		if foundReference {
			changed = true
			continue
		}
		foundReference = true
		if line != desiredLine {
			changed = true
		}
		updatedLines = append(updatedLines, desiredLine)
	}
	if foundReference {
		if !changed {
			return "", false, nil
		}
		return plannedContent(string(data), strings.Join(updatedLines, "\n"))
	}

	updated := strings.TrimRight(string(data), "\n")
	if updated != "" {
		updated += "\n"
	}
	updated += desiredLine + "\n"

	return plannedContent(string(data), updated)
}

func isAgentsReferenceLine(line string) bool {
	return strings.HasPrefix(line, "See `") &&
		strings.HasSuffix(line, "` for the command catalog and workflows.")
}

func isASCReferenceDirective(line string) bool {
	if !strings.HasPrefix(line, "@") {
		return false
	}
	ref := strings.TrimSpace(strings.TrimPrefix(line, "@"))
	return strings.EqualFold(filepath.Base(ref), ascReferenceFile)
}

func appendSection(content, section string) string {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return section + "\n"
	}
	return trimmed + "\n\n" + section + "\n"
}

// plannedContent reports updated as a pending write when it differs from the
// existing content.
func plannedContent(existing, updated string) (string, bool, error) {
	if existing == updated {
		return "", false, nil
	}
	return updated, true, nil
}
