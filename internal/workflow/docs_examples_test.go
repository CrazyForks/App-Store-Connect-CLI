package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docWorkflowPages are the pages that ship .asc/workflow.json examples.
// Keep this list in sync with the workflow documentation set.
var docWorkflowPages = []string{
	"concepts/workflows.mdx",
	"configuration/workflows.mdx",
	"guides/automation.mdx",
	"docs/WORKFLOWS.md",
}

var (
	docFencePattern    = regexp.MustCompile("(?s)```(?:json|jsonc)[^\n]*\n(.*?)```")
	docIfFieldPattern  = regexp.MustCompile(`"if"\s*:\s*"([^"]*)"`)
	docIdentifierRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type docJSONBlock struct {
	page string
	body string
}

func docJSONBlocks(t *testing.T) []docJSONBlock {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	var blocks []docJSONBlock
	for _, page := range docWorkflowPages {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		for _, match := range docFencePattern.FindAllStringSubmatch(string(data), -1) {
			blocks = append(blocks, docJSONBlock{page: page, body: strings.TrimSpace(match[1])})
		}
	}
	if len(blocks) == 0 {
		t.Fatal("expected the workflow documentation to ship JSON examples")
	}
	return blocks
}

// isCompleteExample reports whether a documented block is a full workflow file
// rather than a schema fragment or a single-step snippet.
func isCompleteExample(body string) bool {
	return strings.HasPrefix(body, "{") &&
		strings.Contains(body, `"workflows"`) &&
		!strings.Contains(body, "...")
}

func TestDocWorkflowExamplesLoadAndValidate(t *testing.T) {
	complete := 0
	for _, block := range docJSONBlocks(t) {
		if !isCompleteExample(block.body) {
			continue
		}
		complete++

		path := filepath.Join(t.TempDir(), "workflow.json")
		if err := os.WriteFile(path, []byte(block.body), 0o600); err != nil {
			t.Fatalf("write example: %v", err)
		}
		if _, err := Load(path); err != nil {
			t.Errorf("%s: documented workflow example does not load and validate: %v\n%s", block.page, err, block.body)
		}
	}
	if complete == 0 {
		t.Fatal("expected the workflow documentation to ship at least one complete example")
	}
}

// TestDocWorkflowExamplesUseVariableNamesForIf pins the documented `if` contract
// to the engine: `if` names a variable, so a shell expression such as
// `test "$AUTO_SUBMIT" = "true"` is used verbatim as a variable name, never
// resolves, and silently skips the step.
func TestDocWorkflowExamplesUseVariableNamesForIf(t *testing.T) {
	for _, block := range docJSONBlocks(t) {
		for _, match := range docIfFieldPattern.FindAllStringSubmatch(block.body, -1) {
			if !docIdentifierRegex.MatchString(strings.TrimSpace(match[1])) {
				t.Errorf("%s: %s is not a variable name; `if` takes an env-var name, not an expression", block.page, match[0])
			}
		}
	}
}

// TestDocWorkflowExamplesDemonstrateTruthyGates guards the failure the audit
// found: a complete example gating a mutating step on a value that is never
// truthy (an ID, a path) runs green while doing nothing. Every gate in a
// complete example must be shown with a truthy value on the same page.
func TestDocWorkflowExamplesDemonstrateTruthyGates(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	gates := 0
	for _, page := range docWorkflowPages {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		text := string(data)

		for _, match := range docFencePattern.FindAllStringSubmatch(text, -1) {
			body := strings.TrimSpace(match[1])
			if !isCompleteExample(body) {
				continue
			}
			for _, ifMatch := range docIfFieldPattern.FindAllStringSubmatch(body, -1) {
				name := strings.TrimSpace(ifMatch[1])
				if !docIdentifierRegex.MatchString(name) {
					continue // reported by TestDocWorkflowExamplesUseVariableNamesForIf
				}
				gates++
				if !pageShowsTruthyValue(text, name) {
					t.Errorf("%s: step gated on %q but the page never shows a truthy value for it; the step is skipped on every documented invocation", page, name)
				}
			}
		}
	}
	if gates == 0 {
		t.Fatal("expected at least one complete documented example to gate a step with `if`")
	}
}

// TestDocsDescribeAfterAllAsSuccessOnly keeps the pages aligned with Run: a step
// failure returns before the after_all block, so after_all never runs on the
// failure path (see TestRun_AfterAllDoesNotRunOnStepFailure). Documenting it as
// a "success or failure" hook sends cleanup into the one path it never covers.
func TestDocsDescribeAfterAllAsSuccessOnly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	for _, page := range docWorkflowPages {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		inAfterAllSection := false
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				inAfterAllSection = mentionsAfterAll(line)
			}
			if !inAfterAllSection && !mentionsAfterAll(line) {
				continue
			}
			if strings.Contains(strings.ToLower(line), "or failure") {
				t.Errorf("%s:%d claims after_all runs on failure, but the runner skips it on the failure path: %s", page, i+1, strings.TrimSpace(line))
			}
		}
	}
}

func mentionsAfterAll(line string) bool {
	return strings.Contains(line, "after_all") || strings.Contains(line, `after\_all`)
}

func pageShowsTruthyValue(page, name string) bool {
	for _, truthy := range []string{"1", "true", "yes", "y", "on"} {
		if strings.Contains(page, name+":"+truthy) || strings.Contains(page, name+"="+truthy) {
			return true
		}
	}
	return false
}
