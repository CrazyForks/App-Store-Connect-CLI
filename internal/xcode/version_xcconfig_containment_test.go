package xcode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// externalXCConfigProject builds an xcconfig-backed project whose base
// configuration reference points at a directory outside the project root.
func externalXCConfigProject(t *testing.T) (projectPath string, externalDir string) {
	t.Helper()

	projectPath = writeStructuredVersionProject(t, true)
	projectRoot := filepath.Dir(projectPath)
	externalDir = t.TempDir()

	for _, name := range []string{"App.xcconfig", "Shared.xcconfig"} {
		source := filepath.Join(projectRoot, "Configs", name)
		data := mustReadVersionTestFile(t, source)
		if err := os.WriteFile(filepath.Join(externalDir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
		if err := os.Remove(source); err != nil {
			t.Fatalf("Remove(%s) error = %v", name, err)
		}
	}

	pbxprojPath := filepath.Join(projectPath, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	old := "path = Configs/App.xcconfig; sourceTree = SOURCE_ROOT;"
	if count := strings.Count(contents, old); count != 1 {
		t.Fatalf("expected one App.xcconfig reference, found %d", count)
	}
	contents = strings.Replace(contents, old,
		`path = "`+filepath.Join(externalDir, "App.xcconfig")+`"; sourceTree = "<absolute>";`, 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	return projectPath, externalDir
}

func TestSetVersionRefusesExternalXCConfigByDefault(t *testing.T) {
	projectPath, externalDir := externalXCConfigProject(t)
	externalPath := filepath.Join(externalDir, "Shared.xcconfig")
	before := mustReadVersionTestFile(t, externalPath)

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:  projectPath,
		Version:     "2.0.0",
		BuildNumber: "99",
	})
	if err == nil {
		t.Fatal("SetVersion() error = nil, want external xcconfig rejection")
	}
	if !strings.Contains(err.Error(), "--allow-external-xcconfig") {
		t.Fatalf("SetVersion() error = %v, want migration guidance naming --allow-external-xcconfig", err)
	}
	if after := mustReadVersionTestFile(t, externalPath); after != before {
		t.Fatal("external xcconfig was rewritten despite the rejection")
	}
}

func TestValidateSetVersionRefusesExternalXCConfigByDefault(t *testing.T) {
	projectPath, externalDir := externalXCConfigProject(t)
	externalPath := filepath.Join(externalDir, "Shared.xcconfig")
	before := mustReadVersionTestFile(t, externalPath)

	err := ValidateSetVersion(SetVersionOptions{
		ProjectDir:  projectPath,
		BuildNumber: "1",
	})
	if err == nil {
		t.Fatal("ValidateSetVersion() error = nil, want external xcconfig rejection")
	}
	if !strings.Contains(err.Error(), "--allow-external-xcconfig") {
		t.Fatalf("ValidateSetVersion() error = %v, want migration guidance", err)
	}
	if after := mustReadVersionTestFile(t, externalPath); after != before {
		t.Fatal("external xcconfig changed during validation")
	}
}

func TestBumpVersionRefusesExternalXCConfigByDefault(t *testing.T) {
	projectPath, externalDir := externalXCConfigProject(t)
	externalPath := filepath.Join(externalDir, "Shared.xcconfig")
	before := mustReadVersionTestFile(t, externalPath)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: projectPath,
		BumpType:   BumpPatch,
	})
	if err == nil {
		t.Fatal("BumpVersion() error = nil, want external xcconfig rejection")
	}
	if !strings.Contains(err.Error(), "--allow-external-xcconfig") {
		t.Fatalf("BumpVersion() error = %v, want migration guidance", err)
	}
	if after := mustReadVersionTestFile(t, externalPath); after != before {
		t.Fatal("external xcconfig was rewritten despite the rejection")
	}
}

func TestSetVersionAllowsExternalXCConfigWhenAuthorized(t *testing.T) {
	projectPath, externalDir := externalXCConfigProject(t)
	externalPath := filepath.Join(externalDir, "Shared.xcconfig")

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:            projectPath,
		Version:               "2.0.0",
		BuildNumber:           "99",
		AllowExternalXCConfig: true,
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.ChangedFiles) == 0 {
		t.Fatalf("SetVersion() result = %#v, want changed files", result)
	}
	if after := mustReadVersionTestFile(t, externalPath); !strings.Contains(after, "MARKETING_VERSION = 2.0.0") {
		t.Fatalf("external xcconfig content = %q, want authorized update", after)
	}
}

func TestSetVersionStillEditsXCConfigInsideProjectRoot(t *testing.T) {
	projectPath := writeStructuredVersionProject(t, true)
	sharedPath := filepath.Join(filepath.Dir(projectPath), "Configs", "Shared.xcconfig")

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:  projectPath,
		Version:     "2.0.0",
		BuildNumber: "99",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.ChangedFiles) == 0 {
		t.Fatalf("SetVersion() result = %#v, want changed files", result)
	}
	if after := mustReadVersionTestFile(t, sharedPath); !strings.Contains(after, "MARKETING_VERSION = 2.0.0") {
		t.Fatalf("in-root xcconfig content = %q, want update", after)
	}
}
