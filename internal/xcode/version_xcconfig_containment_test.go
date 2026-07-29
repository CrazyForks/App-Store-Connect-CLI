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

func TestSetVersionReportsSymlinkedXCConfigAsSymlinkRejection(t *testing.T) {
	projectPath := writeStructuredVersionProject(t, true)
	projectRoot := filepath.Dir(projectPath)
	sharedPath := filepath.Join(projectRoot, "Configs", "Shared.xcconfig")

	externalDir := t.TempDir()
	externalPath := filepath.Join(externalDir, "Shared.xcconfig")
	data := mustReadVersionTestFile(t, sharedPath)
	if err := os.WriteFile(externalPath, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Remove(sharedPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := os.Symlink(externalPath, sharedPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	before := mustReadVersionTestFile(t, externalPath)

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:  projectPath,
		Version:     "2.0.0",
		BuildNumber: "99",
	})
	if err == nil {
		t.Fatal("SetVersion() error = nil, want symlinked xcconfig rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SetVersion() error = %v, want symlink rejection", err)
	}
	if strings.Contains(err.Error(), "outside project directory") {
		t.Fatalf("SetVersion() error = %v, want the symlink-specific message, not the containment one", err)
	}
	// The write path rejects symlinked xcconfig files even when authorized, so
	// the error must not present --allow-external-xcconfig as a remedy.
	if strings.Contains(err.Error(), "--allow-external-xcconfig") {
		t.Fatalf("SetVersion() error = %v, want no --allow-external-xcconfig guidance for symlinks", err)
	}
	if after := mustReadVersionTestFile(t, externalPath); after != before {
		t.Fatal("symlink target was rewritten despite the rejection")
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
