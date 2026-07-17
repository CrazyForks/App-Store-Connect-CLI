package xcode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := GetVersion(context.Background(), ".", "")
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestSetVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := SetVersion(context.Background(), SetVersionOptions{ProjectDir: ".", Version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestBumpVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := BumpVersion(context.Background(), BumpVersionOptions{ProjectDir: ".", BumpType: BumpPatch})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestGetVersion_MissingAgvtool(t *testing.T) {
	prev := lookPathFn
	lookPathFn = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}
	defer func() { lookPathFn = prev }()

	prevOS := runtimeGOOS
	runtimeGOOS = "darwin"
	defer func() { runtimeGOOS = prevOS }()

	_, err := GetVersion(context.Background(), ".", "")
	if err == nil || !strings.Contains(err.Error(), "agvtool") {
		t.Fatalf("expected agvtool not found error, got: %v", err)
	}
}

func TestBumpVersionType_Validate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"major", true},
		{"minor", true},
		{"patch", true},
		{"build", true},
		{"MAJOR", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		_, err := ParseBumpType(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ParseBumpType(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ParseBumpType(%q) expected error", tt.input)
		}
	}
}

func TestIsVariableReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"$(MARKETING_VERSION)", true},
		{"1.2.3", false},
		{"$(CURRENT_PROJECT_VERSION)", true},
		{"", false},
	}

	for _, tt := range tests {
		if got := isVariableReference(tt.input); got != tt.want {
			t.Errorf("isVariableReference(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIncrementBuildString(t *testing.T) {
	tests := []struct {
		current string
		want    string
		wantErr bool
	}{
		{"42", "43", false},
		{"1", "2", false},
		{"1.2.3", "1.2.4", false},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			got, err := incrementBuildString(tt.current)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("incrementBuildString(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestParseAgvtoolVersionOutput_TargetFilter(t *testing.T) {
	multiTargetOutput := "App=1.2.3\nExtension=2.0.0\n"

	got, err := parseAgvtoolVersionOutput(multiTargetOutput, "Extension")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2.0.0" {
		t.Fatalf("expected Extension version, got %q", got)
	}
}

func TestParseAgvtoolVersionOutput_RequiresTargetForAmbiguousOutput(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nExtension=2.0.0\n", "")
	if err == nil || !strings.Contains(err.Error(), "use --target") {
		t.Fatalf("expected ambiguous target error, got %v", err)
	}
}

func TestParseAgvtoolVersionOutput_MissingTargetErrors(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nExtension=2.0.0\n", "Widget")
	if err == nil || !strings.Contains(err.Error(), `target "Widget" not found`) {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestParseAgvtoolBuildOutput_TargetFilter(t *testing.T) {
	got, err := parseAgvtoolBuildOutput("App=41\nExtension=7\n", "App")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "41" {
		t.Fatalf("expected App build number, got %q", got)
	}
}

func TestBuildSettingsTargetNames(t *testing.T) {
	output := `
Build settings for action build and target App:
    MARKETING_VERSION = 1.2.3

Build settings for action build and target App:
    CURRENT_PROJECT_VERSION = 42

Build settings for action build and target Extension:
    MARKETING_VERSION = 2.0.0
`

	targets := buildSettingsTargetNames(output)
	if len(targets) != 2 || targets[0] != "App" || targets[1] != "Extension" {
		t.Fatalf("expected target names [App Extension], got %v", targets)
	}
}

func TestFindXcodeprojAcceptsExplicitProjectPath(t *testing.T) {
	tempDir := t.TempDir()
	appProject := filepath.Join(tempDir, "App.xcodeproj")
	podsProject := filepath.Join(tempDir, "Pods.xcodeproj")
	if err := os.MkdirAll(appProject, 0o755); err != nil {
		t.Fatalf("mkdir app project: %v", err)
	}
	if err := os.MkdirAll(podsProject, 0o755); err != nil {
		t.Fatalf("mkdir pods project: %v", err)
	}

	got, err := findXcodeproj(appProject)
	if err != nil {
		t.Fatalf("expected explicit project path to succeed, got %v", err)
	}
	if got != appProject {
		t.Fatalf("expected %q, got %q", appProject, got)
	}
}

func TestFindXcodeprojMultipleProjectsSuggestsProjectFlag(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "App.xcodeproj"), 0o755); err != nil {
		t.Fatalf("mkdir app project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "Pods.xcodeproj"), 0o755); err != nil {
		t.Fatalf("mkdir pods project: %v", err)
	}

	_, err := findXcodeproj(tempDir)
	if err == nil || !strings.Contains(err.Error(), "use --project to pick one") {
		t.Fatalf("expected actionable multiple project error, got %v", err)
	}
}

func TestSetVersionTargetedWritesRequireStructuredProject(t *testing.T) {
	prevOS := runtimeGOOS
	prevLookPath := lookPathFn
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	defer func() {
		runtimeGOOS = prevOS
		lookPathFn = prevLookPath
	}()

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: ".",
		Target:     "App",
		Version:    "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "scoped edits require structured Xcode build settings") {
		t.Fatalf("expected structured-project requirement, got %v", err)
	}
}

func TestSetVersionLegacyMultiTargetUsesProjectWideWrite(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "Project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Cleanup(restore)

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: projectDir,
		Version:    "1.3.0",
	})
	if err != nil {
		t.Fatalf("expected project-wide edit to succeed, got %v", err)
	}
	if result.Version != "1.3.0" {
		t.Fatalf("expected edited version in result, got %#v", result)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read helper log: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "agvtool|new-marketing-version|1.3.0") {
		t.Fatalf("expected project-wide marketing version update, got %q", logText)
	}
	if strings.Contains(logText, "agvtool|what-version|-terse") {
		t.Fatalf("expected edit path to avoid ambiguous build-number reads, got %q", logText)
	}
}

func TestBumpVersionLegacyTargetScopeIsRejectedBeforeProjectWideWrite(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "Project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Cleanup(restore)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: projectDir,
		Target:     "Extension",
		BumpType:   BumpPatch,
	})
	if err == nil || !strings.Contains(err.Error(), "scoped or remote-safe bumps require structured") {
		t.Fatalf("expected structured-project requirement, got %v", err)
	}

	if logData, readErr := os.ReadFile(logPath); readErr == nil && strings.Contains(string(logData), "new-marketing-version") {
		t.Fatalf("legacy target scope performed a project-wide write: %q", logData)
	}
}

func TestBumpVersionString(t *testing.T) {
	tests := []struct {
		current  string
		bumpType BumpType
		want     string
		wantErr  bool
	}{
		{"1.2.3", BumpPatch, "1.2.4", false},
		{"1.2.3", BumpMinor, "1.3.0", false},
		{"1.2.3", BumpMajor, "2.0.0", false},
		{"1.0", BumpPatch, "", true},
		{"1.0", BumpMinor, "1.1", false},
		{"1.0", BumpMajor, "2.0", false},
		{"bad", BumpPatch, "", true},
		{"", BumpPatch, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+string(tt.bumpType), func(t *testing.T) {
			got, err := bumpVersionString(tt.current, tt.bumpType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpVersionString(%q, %s) = %q, want %q", tt.current, tt.bumpType, got, tt.want)
			}
		})
	}
}
