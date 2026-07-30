package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// githubActionsDocFiles lists documentation pages whose GitHub Actions examples
// are copied into workflows that already have App Store Connect credentials
// configured. A ref-controlled expression inside `run:` becomes shell source in
// those workflows, so the examples must keep refs in `env:` instead.
var githubActionsDocFiles = []string{
	filepath.Join("configuration", "workflows.mdx"),
	filepath.Join("commands", "validate.mdx"),
}

// refNameInjectionPayload is a valid Git ref name (no space, `~`, `^`, `:`, `?`,
// `*`, `[`, or `\`) that appends a redirect if it ever reaches shell source.
const refNameInjectionPayload = "v1.0;>injected"

var actionsExpressionPattern = regexp.MustCompile(`\$\{\{\s*([^{}]+?)\s*\}\}`)

type docActionStep struct {
	File string
	Name string
	Env  map[string]string
	Run  string
}

func TestGitHubActionsDocExamplesKeepRefNameOutOfShellSource(t *testing.T) {
	for _, file := range githubActionsDocFiles {
		steps := gitHubActionsDocSteps(t, file)
		if len(steps) == 0 {
			t.Fatalf("%s: expected at least one GitHub Actions run step", file)
		}

		refInEnv := false
		for _, step := range steps {
			if match := actionsExpressionPattern.FindString(step.Run); match != "" {
				t.Errorf("%s: step %q interpolates %s inside run source; pass it through env instead", file, step.Name, match)
			}
			for name, value := range step.Env {
				if strings.Contains(value, "github.ref_name") {
					refInEnv = true
					if !referencesQuotedShellVariable(step.Run, name) {
						t.Errorf("%s: step %q sets env %s but does not reference it as a quoted shell variable", file, step.Name, name)
					}
				}
			}
		}
		if !refInEnv {
			t.Errorf("%s: expected a step-level env entry carrying github.ref_name", file)
		}
	}
}

func TestGitHubActionsDocExamplesTreatRefNameAsInertArgument(t *testing.T) {
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash is not installed; shell-safety of the documented examples cannot be executed here: %v", err)
	}

	for _, file := range githubActionsDocFiles {
		for _, step := range gitHubActionsDocSteps(t, file) {
			script, env := renderDocStepScript(t, file, step)
			if !strings.Contains(script, refNameInjectionPayload) && !stepEnvContains(env, refNameInjectionPayload) {
				continue
			}

			t.Run(file+"/"+step.Name, func(t *testing.T) {
				workDir := t.TempDir()
				argsFile := filepath.Join(workDir, "asc-args.txt")
				stubDir := writeASCStub(t, argsFile)

				cmd := exec.Command(bashPath, "-c", script)
				cmd.Dir = workDir
				cmd.Env = append(os.Environ(), env...)
				cmd.Env = append(cmd.Env, "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("run documented step: %v\n%s", err, output)
				}

				if entries, err := os.ReadDir(workDir); err == nil {
					for _, entry := range entries {
						if entry.Name() == "injected" {
							t.Fatalf("ref name executed shell syntax: %q was created", entry.Name())
						}
					}
				}

				args := recordedASCArgs(t, argsFile)
				if len(args) == 0 {
					t.Fatalf("expected the documented step to invoke asc")
				}
				sawFullRef := false
				for _, arg := range args {
					if !strings.Contains(arg, "v1.0") {
						continue
					}
					if !strings.Contains(arg, refNameInjectionPayload) {
						t.Fatalf("asc argument %q lost part of the ref; the ref must arrive as one inert argument containing %q", arg, refNameInjectionPayload)
					}
					sawFullRef = true
				}
				if !sawFullRef {
					t.Fatalf("expected asc to receive the ref name; got args %q", args)
				}
			})
		}
	}
}

// referencesQuotedShellVariable reports whether script expands name at least
// once and never expands it outside double quotes, so word splitting and glob
// expansion cannot act on the value.
func referencesQuotedShellVariable(script string, name string) bool {
	found := false
	inSingle := false
	inDouble := false

	for i := 0; i < len(script); i++ {
		switch c := script[i]; {
		case c == '\\' && !inSingle:
			i++
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\n':
			inSingle = false
			inDouble = false
		case c == '$' && !inSingle:
			rest := script[i+1:]
			braced := strings.HasPrefix(rest, "{"+name+"}")
			bare := strings.HasPrefix(rest, name) && !isShellNameByte(byteAt(rest, len(name)))
			if !braced && !bare {
				continue
			}
			if !inDouble {
				return false
			}
			found = true
		}
	}

	return found
}

func isShellNameByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func byteAt(value string, index int) byte {
	if index >= len(value) {
		return 0
	}
	return value[index]
}

func gitHubActionsDocSteps(t *testing.T, file string) []docActionStep {
	t.Helper()

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}

	var steps []docActionStep
	for _, block := range fencedBlocks(string(data), "yaml") {
		if !strings.Contains(block, "run:") {
			continue
		}
		var root yaml.Node
		if err := yaml.Unmarshal([]byte(block), &root); err != nil {
			t.Fatalf("%s: parse yaml example: %v\n%s", file, err, block)
		}
		collectDocActionSteps(file, &root, &steps)
	}
	return steps
}

func fencedBlocks(content string, language string) []string {
	var blocks []string
	var current []string
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
				inBlock = false
				continue
			}
			inBlock = strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(line, "```")), language)
			continue
		}
		if inBlock {
			current = append(current, line)
		}
	}
	return blocks
}

func collectDocActionSteps(file string, node *yaml.Node, steps *[]docActionStep) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		step := docActionStep{File: file}
		hasRun := false
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1]
			switch key {
			case "name":
				step.Name = value.Value
			case "run":
				step.Run = value.Value
				hasRun = true
			case "env":
				if value.Kind != yaml.MappingNode {
					continue
				}
				step.Env = make(map[string]string, len(value.Content)/2)
				for j := 0; j+1 < len(value.Content); j += 2 {
					step.Env[value.Content[j].Value] = value.Content[j+1].Value
				}
			}
		}
		if hasRun {
			*steps = append(*steps, step)
		}
	}
	for _, child := range node.Content {
		collectDocActionSteps(file, child, steps)
	}
}

// renderDocStepScript expands GitHub Actions expressions the way the runner
// does: expression results are substituted into the workflow text before the
// shell ever sees it.
func renderDocStepScript(t *testing.T, file string, step docActionStep) (string, []string) {
	t.Helper()

	env := make([]string, 0, len(step.Env))
	for name, value := range step.Env {
		env = append(env, name+"="+expandActionsExpressions(t, file, step, value))
	}
	return expandActionsExpressions(t, file, step, step.Run), env
}

func expandActionsExpressions(t *testing.T, file string, step docActionStep, value string) string {
	t.Helper()

	return actionsExpressionPattern.ReplaceAllStringFunc(value, func(match string) string {
		expression := strings.TrimSpace(actionsExpressionPattern.FindStringSubmatch(match)[1])
		switch {
		case expression == "github.ref_name":
			return refNameInjectionPayload
		case strings.HasPrefix(expression, "secrets."):
			return "123456789"
		default:
			t.Fatalf("%s: step %q uses unsupported expression %q", file, step.Name, expression)
			return ""
		}
	})
}

func stepEnvContains(env []string, value string) bool {
	for _, entry := range env {
		if strings.Contains(entry, value) {
			return true
		}
	}
	return false
}

func writeASCStub(t *testing.T, argsFile string) string {
	t.Helper()

	stubDir := t.TempDir()
	stub := "#!/bin/sh\nfor arg in \"$@\"; do printf '%s\\n' \"$arg\" >> " + argsFile + "; done\n"
	stubPath := filepath.Join(stubDir, "asc")
	if err := os.WriteFile(stubPath, []byte(stub), 0o700); err != nil {
		t.Fatalf("write asc stub: %v", err)
	}
	return stubDir
}

func recordedASCArgs(t *testing.T, argsFile string) []string {
	t.Helper()

	data, err := os.ReadFile(argsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read recorded asc args: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}
