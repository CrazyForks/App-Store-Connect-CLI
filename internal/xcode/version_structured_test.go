package xcode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuredVersion_TargetAndConfigurationScopedEditIsCrossPlatform(t *testing.T) {
	project := writeStructuredVersionProject(t, false)

	previousOS := runtimeGOOS
	runtimeGOOS = "linux"
	t.Cleanup(func() { runtimeGOOS = previousOS })

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Release",
		Version:       "2.0.0",
		BuildNumber:   "50",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if result.Target != "App" || result.Configuration != "Release" {
		t.Fatalf("unexpected scope: %#v", result)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected two setting changes, got %#v", result.Changes)
	}
	if len(result.ChangedFiles) != 1 || !strings.HasSuffix(result.ChangedFiles[0], "project.pbxproj") {
		t.Fatalf("expected only project.pbxproj to change, got %#v", result.ChangedFiles)
	}

	release, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Release",
	})
	if err != nil {
		t.Fatalf("GetVersionScoped(Release) error = %v", err)
	}
	if release.Version != "2.0.0" || release.BuildNumber != "50" {
		t.Fatalf("unexpected Release version: %#v", release)
	}

	debug, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Debug",
	})
	if err != nil {
		t.Fatalf("GetVersionScoped(Debug) error = %v", err)
	}
	if debug.Version != "1.2.3" || debug.BuildNumber != "42" {
		t.Fatalf("Debug should remain unchanged, got %#v", debug)
	}

	widget, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:    project,
		Target:        "Widget",
		Configuration: "Release",
	})
	if err != nil {
		t.Fatalf("GetVersionScoped(Widget) error = %v", err)
	}
	if widget.Version != "1.2.3" || widget.BuildNumber != "42" {
		t.Fatalf("Widget should remain unchanged, got %#v", widget)
	}
}

func TestStructuredVersion_ProjectWideEditUpdatesRecursiveXCConfigLosslessly(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:  project,
		Version:     "3.4.5",
		BuildNumber: "99",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if !containsPathSuffix(result.ChangedFiles, "Configs/Shared.xcconfig") {
		t.Fatalf("expected recursive xcconfig in changed files, got %#v", result.ChangedFiles)
	}
	for _, configuration := range []string{"Debug", "Release"} {
		for _, setting := range []string{marketingVersionSetting, currentProjectSetting} {
			if !containsVersionChange(result.Changes, "App", configuration, setting, "xcconfig") {
				t.Fatalf("missing exact xcconfig scope App/%s/%s in %#v", configuration, setting, result.Changes)
			}
		}
	}

	contents, err := os.ReadFile(sharedPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(contents)
	if !strings.Contains(got, "MARKETING_VERSION = 3.4.5 // keep this comment\r\n") {
		t.Fatalf("marketing version/comment/CRLF not preserved: %q", got)
	}
	if !strings.Contains(got, "CURRENT_PROJECT_VERSION[sdk=iphoneos*] = 99\r\n") {
		t.Fatalf("conditional build number not updated: %q", got)
	}
	if !strings.Contains(got, "/* MARKETING_VERSION = 8.8.8; */\r\n") {
		t.Fatalf("block comment changed unexpectedly: %q", got)
	}

	view, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Release",
	})
	if err != nil {
		t.Fatalf("GetVersionScoped() error = %v", err)
	}
	if view.Version != "3.4.5" || view.BuildNumber != "99" {
		t.Fatalf("unexpected xcconfig-backed view: %#v", view)
	}
	if !strings.HasSuffix(view.VersionSource, "Configs/Shared.xcconfig") {
		t.Fatalf("unexpected version source: %#v", view)
	}
}

func containsVersionChange(changes []VersionChange, target, configuration, setting, source string) bool {
	for _, change := range changes {
		if change.Target == target && change.Configuration == configuration && change.Setting == setting && change.Source == source {
			return true
		}
	}
	return false
}

func TestStructuredVersion_InvalidValueDoesNotMutateAnyFile(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")
	beforeProject := mustReadVersionTestFile(t, pbxprojPath)
	beforeShared := mustReadVersionTestFile(t, sharedPath)

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project,
		Version:    "1.2.3\nEVIL = YES",
	})
	if err == nil {
		t.Fatal("expected newline validation error")
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != beforeProject {
		t.Fatal("project.pbxproj changed after validation failure")
	}
	if got := mustReadVersionTestFile(t, sharedPath); got != beforeShared {
		t.Fatal("xcconfig changed after validation failure")
	}
}

func TestStructuredVersion_SyntaxChangingValuesDoNotMutateAnyFile(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "line comment", value: "1.2.3 // ignored"},
		{name: "block comment", value: "1.2.3 /* ignored */"},
		{name: "parenthesized setting", value: "$(UNDEFINED)"},
		{name: "braced setting", value: "${UNDEFINED}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := writeStructuredVersionProject(t, true)
			pbxprojPath := filepath.Join(project, "project.pbxproj")
			sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")
			beforeProject := mustReadVersionTestFile(t, pbxprojPath)
			beforeShared := mustReadVersionTestFile(t, sharedPath)

			_, err := SetVersion(context.Background(), SetVersionOptions{
				ProjectDir: project,
				Version:    test.value,
			})
			if err == nil {
				t.Fatalf("expected %q to be rejected", test.value)
			}
			if got := mustReadVersionTestFile(t, pbxprojPath); got != beforeProject {
				t.Fatal("project.pbxproj changed after value validation failure")
			}
			if got := mustReadVersionTestFile(t, sharedPath); got != beforeShared {
				t.Fatal("xcconfig changed after value validation failure")
			}
		})
	}
}

func TestStructuredVersion_MissingScopeErrorsWithoutMutation(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	before := mustReadVersionTestFile(t, pbxprojPath)

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:    project,
		Target:        "Missing",
		Configuration: "Release",
		Version:       "2.0.0",
	})
	if err == nil {
		t.Fatal("expected missing target error")
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != before {
		t.Fatal("project.pbxproj changed after scope validation failure")
	}
}

func TestStructuredVersion_TargetScopeUpdatesAllTargetConfigurations(t *testing.T) {
	project := writeStructuredVersionProject(t, false)

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project,
		Target:     "App",
		Version:    "2.0.0",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected App Debug and Release changes, got %#v", result.Changes)
	}
	for _, configuration := range []string{"Debug", "Release"} {
		view := mustGetStructuredVersion(t, project, "App", configuration)
		if view.Version != "2.0.0" {
			t.Fatalf("App %s version = %q", configuration, view.Version)
		}
	}
	if view := mustGetStructuredVersion(t, project, "Widget", "Release"); view.Version != "1.2.3" {
		t.Fatalf("Widget changed outside target scope: %#v", view)
	}
}

func TestStructuredVersion_ConfigurationScopeUpdatesMatchingTargetsOnly(t *testing.T) {
	project := writeStructuredVersionProject(t, false)

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:    project,
		Configuration: "Debug",
		BuildNumber:   "55",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected App and Widget Debug changes, got %#v", result.Changes)
	}
	for _, target := range []string{"App", "Widget"} {
		if view := mustGetStructuredVersion(t, project, target, "Debug"); view.BuildNumber != "55" {
			t.Fatalf("%s Debug build = %q", target, view.BuildNumber)
		}
		if view := mustGetStructuredVersion(t, project, target, "Release"); view.BuildNumber != "42" {
			t.Fatalf("%s Release changed outside configuration scope: %#v", target, view)
		}
	}
}

func TestStructuredVersion_ConfigurationScopeDoesNotMaterializeRedundantInheritedOverrides(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"999999999999999999999992 /* Project Release */ = {isa = XCBuildConfiguration; buildSettings = {}; name = Release; };",
		"999999999999999999999992 /* Project Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };", 1)
	contents = strings.Replace(contents,
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = {}; name = Release; };", 1)
	contents = strings.Replace(contents,
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = {}; name = Release; };", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project, Configuration: "Release", Version: "2.0.0",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.Changes) != 1 || result.Changes[0].Target != "" || result.Changes[0].Configuration != "Release" {
		t.Fatalf("expected one project-level inherited change, got %#v", result.Changes)
	}
	for _, target := range []string{"App", "Widget"} {
		if view := mustGetStructuredVersion(t, project, target, "Release"); view.Version != "2.0.0" {
			t.Fatalf("%s did not inherit updated project value: %#v", target, view)
		}
	}
}

func TestStructuredVersion_DefaultViewSelectsApplicationAndDefaultConfiguration(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	view, err := GetVersionScoped(context.Background(), GetVersionOptions{ProjectDir: project})
	if err != nil {
		t.Fatalf("GetVersionScoped() error = %v", err)
	}
	if view.Target != "App" || view.Configuration != "Release" || view.Version != "1.2.3" || view.BuildNumber != "42" {
		t.Fatalf("unexpected default view: %#v", view)
	}
}

func TestStructuredVersion_DirectInheritedValueUsesNextLowerLayer(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"999999999999999999999992 /* Project Release */ = {isa = XCBuildConfiguration; buildSettings = {}; name = Release; };",
		"999999999999999999999992 /* Project Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 7.8.9; }; name = Release; };", 1)
	contents = strings.Replace(contents,
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = { MARKETING_VERSION = \"$(inherited)\"; CURRENT_PROJECT_VERSION = 42; }; name = Release; };", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	view := mustGetStructuredVersion(t, project, "App", "Release")
	if view.Version != "7.8.9" {
		t.Fatalf("inherited version = %q, want project-level 7.8.9", view.Version)
	}
}

func TestStructuredVersion_UnscopedBumpSupportsMultipleApplicationTargets(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		`productType = "com.apple.product-type.app-extension";`,
		`productType = "com.apple.product-type.application";`, 1)
	contents = strings.Replace(contents,
		`explicitFileType = "wrapper.app-extension"; path = Widget.appex;`,
		`explicitFileType = wrapper.application; path = Widget.app;`, 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	result, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: project,
		BumpType:   BumpPatch,
	})
	if err != nil {
		t.Fatalf("BumpVersion() error = %v", err)
	}
	if result.OldVersion != "1.2.3" || result.NewVersion != "1.2.4" {
		t.Fatalf("unexpected unscoped bump result: %#v", result)
	}
	for _, target := range []string{"App", "Widget"} {
		for _, configuration := range []string{"Debug", "Release"} {
			if view := mustGetStructuredVersion(t, project, target, configuration); view.Version != "1.2.4" {
				t.Fatalf("%s/%s version = %q, want 1.2.4", target, configuration, view.Version)
			}
		}
	}
}

func TestStructuredVersion_BumpRejectsMissingSettingWithoutMutation(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { CURRENT_PROJECT_VERSION = 42; }; name = Release; };", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}
	before := mustReadVersionTestFile(t, pbxprojPath)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: project,
		BumpType:   BumpPatch,
	})
	if err == nil || !strings.Contains(err.Error(), "MARKETING_VERSION") {
		t.Fatalf("expected missing MARKETING_VERSION error, got %v", err)
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != before {
		t.Fatal("project changed after missing-setting validation failure")
	}
}

func TestStructuredVersion_EditRejectsMissingSettingWithoutMutation(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { CURRENT_PROJECT_VERSION = 42; }; name = Release; };", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}
	before := mustReadVersionTestFile(t, pbxprojPath)

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project,
		Version:    "2.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "MARKETING_VERSION") {
		t.Fatalf("expected missing MARKETING_VERSION error, got %v", err)
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != before {
		t.Fatal("project changed after missing-setting edit validation failure")
	}
}

func TestStructuredVersion_RejectsConditionalOnlyPBXProjValue(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };",
		"999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration;  buildSettings = { MARKETING_VERSION = 1.2.3; \"CURRENT_PROJECT_VERSION[sdk=iphoneos*]\" = 42; \"CURRENT_PROJECT_VERSION[sdk=macosx*]\" = 43; }; name = Release; };", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	_, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir: project, Target: "App", Configuration: "Release",
	})
	if err == nil || !strings.Contains(err.Error(), "conditional") {
		t.Fatalf("expected conditional-only build-setting error, got %v", err)
	}

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project, Target: "App", Configuration: "Release", BuildNumber: "50",
	})
	if err != nil {
		t.Fatalf("conditional-only direct edit should not need SDK resolution: %v", err)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected both conditional assignments to change, got %#v", result.Changes)
	}
	updated := mustReadVersionTestFile(t, pbxprojPath)
	if !strings.Contains(updated, `"CURRENT_PROJECT_VERSION[sdk=iphoneos*]" = 50;`) ||
		!strings.Contains(updated, `"CURRENT_PROJECT_VERSION[sdk=macosx*]" = 50;`) {
		t.Fatalf("conditional assignments were not both edited: %s", updated)
	}
}

func TestStructuredVersion_PartialBuildSettingsUseLegacyRouting(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := strings.ReplaceAll(mustReadVersionTestFile(t, pbxprojPath), "CURRENT_PROJECT_VERSION = 42;", "")
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	_, err := openStructuredVersionProject(project)
	if !errors.Is(err, errStructuredVersionUnavailable) {
		t.Fatalf("expected partial project to use legacy routing, got %v", err)
	}
}

func TestStructuredVersion_GroupRelativeXCConfigReference(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	contents := mustReadVersionTestFile(t, pbxprojPath)
	contents = strings.Replace(contents,
		"path = Configs/App.xcconfig; sourceTree = SOURCE_ROOT;",
		"path = App.xcconfig; sourceTree = \"<group>\";", 1)
	contents = strings.Replace(contents,
		"AAAAAAAAAAAAAAAAAAAAAAAA /* App.xcconfig */ =",
		"ABABABABABABABABABABABAB /* Configs */ = {isa = PBXGroup; children = (AAAAAAAAAAAAAAAAAAAAAAAA,); path = Configs; sourceTree = \"<group>\"; };\n\t\tAAAAAAAAAAAAAAAAAAAAAAAA /* App.xcconfig */ =", 1)
	if err := os.WriteFile(pbxprojPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(project) error = %v", err)
	}

	view := mustGetStructuredVersion(t, project, "App", "Release")
	if view.Version != "1.2.3" || !strings.HasSuffix(view.VersionSource, "Configs/Shared.xcconfig") {
		t.Fatalf("group-relative xcconfig was not resolved: %#v", view)
	}
}

func TestStructuredVersion_ScopedBumps(t *testing.T) {
	t.Run("patch", func(t *testing.T) {
		project := writeStructuredVersionProject(t, false)
		result, err := BumpVersion(context.Background(), BumpVersionOptions{
			ProjectDir: project, Target: "App", Configuration: "Release", BumpType: BumpPatch,
		})
		if err != nil {
			t.Fatalf("BumpVersion() error = %v", err)
		}
		if result.OldVersion != "1.2.3" || result.NewVersion != "1.2.4" || len(result.Changes) != 1 {
			t.Fatalf("unexpected patch result: %#v", result)
		}
		if view := mustGetStructuredVersion(t, project, "App", "Debug"); view.Version != "1.2.3" {
			t.Fatalf("patch leaked to Debug: %#v", view)
		}
	})

	t.Run("remote build override", func(t *testing.T) {
		project := writeStructuredVersionProject(t, false)
		result, err := BumpVersion(context.Background(), BumpVersionOptions{
			ProjectDir: project, Target: "Widget", Configuration: "Debug", BumpType: BumpBuild, BuildNumber: "108",
		})
		if err != nil {
			t.Fatalf("BumpVersion() error = %v", err)
		}
		if result.OldBuild != "42" || result.NewBuild != "108" || len(result.Changes) != 1 {
			t.Fatalf("unexpected build result: %#v", result)
		}
		if view := mustGetStructuredVersion(t, project, "Widget", "Debug"); view.BuildNumber != "108" {
			t.Fatalf("remote override not applied: %#v", view)
		}
	})
}

func TestStructuredVersion_BumpRefusesToFlattenDifferentValues(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	if _, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project, Target: "App", Configuration: "Debug", Version: "9.0.0",
	}); err != nil {
		t.Fatalf("prepare divergent version: %v", err)
	}
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	before := mustReadVersionTestFile(t, pbxprojPath)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: project, Target: "App", BumpType: BumpPatch,
	})
	if err == nil {
		t.Fatal("expected mixed-value bump to fail")
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != before {
		t.Fatal("project changed after inconsistent bump validation")
	}
}

func TestStructuredVersion_RemoteBuildBumpRefusesToFlattenDifferentValues(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	if _, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: project, Target: "App", Configuration: "Debug", BuildNumber: "9",
	}); err != nil {
		t.Fatalf("prepare divergent build: %v", err)
	}
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	before := mustReadVersionTestFile(t, pbxprojPath)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: project, BumpType: BumpBuild, BuildNumber: "108",
	})
	if err == nil || !strings.Contains(err.Error(), "differing values") {
		t.Fatalf("expected remote mixed-value bump to fail, got %v", err)
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != before {
		t.Fatal("project changed after remote consistency validation")
	}
}

func TestStructuredVersion_ScopedSharedXCConfigCreatesOverrideWithoutLeaking(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")
	before := mustReadVersionTestFile(t, sharedPath)

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Release",
		Version:       "4.0.0",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.ChangedFiles) != 1 || !strings.HasSuffix(result.ChangedFiles[0], "project.pbxproj") {
		t.Fatalf("expected a scoped pbxproj override, got %#v", result.ChangedFiles)
	}
	if got := mustReadVersionTestFile(t, sharedPath); got != before {
		t.Fatal("shared xcconfig changed despite unselected consumers")
	}
	if view := mustGetStructuredVersion(t, project, "App", "Release"); view.Version != "4.0.0" {
		t.Fatalf("Release override was not applied: %#v", view)
	}
	if view := mustGetStructuredVersion(t, project, "App", "Debug"); view.Version != "1.2.3" {
		t.Fatalf("Debug leaked scoped change: %#v", view)
	}
}

func TestStructuredVersion_NoOpReportsNoFilesOrChanges(t *testing.T) {
	project := writeStructuredVersionProject(t, false)
	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir:    project,
		Target:        "App",
		Configuration: "Release",
		Version:       "1.2.3",
		BuildNumber:   "42",
	})
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	if len(result.ChangedFiles) != 0 || len(result.Changes) != 0 {
		t.Fatalf("no-op reported mutations: %#v", result)
	}
}

func TestStructuredVersion_XCConfigPermissionsArePreserved(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")

	if _, err := SetVersion(context.Background(), SetVersionOptions{ProjectDir: project, Version: "5.0.0"}); err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}
	info, err := os.Stat(sharedPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("xcconfig mode = %o, want 640", got)
	}
}

func TestStructuredVersion_RefusesXCConfigSymlinkBeforeMutation(t *testing.T) {
	project := writeStructuredVersionProject(t, true)
	pbxprojPath := filepath.Join(project, "project.pbxproj")
	sharedPath := filepath.Join(filepath.Dir(project), "Configs", "Shared.xcconfig")
	targetPath := filepath.Join(filepath.Dir(project), "outside.xcconfig")
	beforeProject := mustReadVersionTestFile(t, pbxprojPath)
	shared := mustReadVersionTestFile(t, sharedPath)
	if err := os.WriteFile(targetPath, []byte(shared), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.Remove(sharedPath); err != nil {
		t.Fatalf("Remove(shared) error = %v", err)
	}
	if err := os.Symlink(targetPath, sharedPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := SetVersion(context.Background(), SetVersionOptions{ProjectDir: project, Version: "6.0.0"})
	if err == nil {
		t.Fatal("expected symlink refusal")
	}
	if got := mustReadVersionTestFile(t, pbxprojPath); got != beforeProject {
		t.Fatal("project changed before symlink validation completed")
	}
	if got := mustReadVersionTestFile(t, targetPath); got != shared {
		t.Fatal("symlink target changed")
	}
}

func TestStructuredVersion_CommitFailureRollsBackEarlierFiles(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "a.xcconfig")
	secondPath := filepath.Join(root, "b.xcconfig")
	if err := os.WriteFile(firstPath, []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte("old-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	injectedErr := errors.New("injected write failure")
	originalWriter := atomicWriteVersionFileFn
	atomicWriteVersionFileFn = func(path string, data []byte, mode os.FileMode) error {
		if path == secondPath {
			return injectedErr
		}
		return atomicWriteVersionFile(path, data, mode)
	}
	t.Cleanup(func() { atomicWriteVersionFileFn = originalWriter })

	err := commitVersionWrites([]preparedVersionWrite{
		{path: secondPath, original: []byte("old-b"), updated: []byte("new-b"), mode: 0o644},
		{path: firstPath, original: []byte("old-a"), updated: []byte("new-a"), mode: 0o644},
	})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("expected injected failure, got %v", err)
	}
	if got := mustReadVersionTestFile(t, firstPath); got != "old-a" {
		t.Fatalf("first file was not rolled back: %q", got)
	}
	if got := mustReadVersionTestFile(t, secondPath); got != "old-b" {
		t.Fatalf("second file changed: %q", got)
	}
}

func mustGetStructuredVersion(t *testing.T, project, target, configuration string) *VersionInfo {
	t.Helper()
	view, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir: project, Target: target, Configuration: configuration,
	})
	if err != nil {
		t.Fatalf("GetVersionScoped(%s/%s) error = %v", target, configuration, err)
	}
	return view
}

func containsPathSuffix(paths []string, suffix string) bool {
	for _, path := range paths {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return true
		}
	}
	return false
}

func mustReadVersionTestFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(contents)
}

func writeStructuredVersionProject(t *testing.T, xcconfigBacked bool) string {
	t.Helper()
	root := t.TempDir()
	projectPath := filepath.Join(root, "Demo.xcodeproj")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(project) error = %v", err)
	}

	appSettings := "MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42;"
	baseConfiguration := ""
	fileReference := ""
	if xcconfigBacked {
		appSettings = ""
		baseConfiguration = "baseConfigurationReference = AAAAAAAAAAAAAAAAAAAAAAAA;"
		fileReference = `
		AAAAAAAAAAAAAAAAAAAAAAAA /* App.xcconfig */ = {isa = PBXFileReference; lastKnownFileType = text.xcconfig; path = Configs/App.xcconfig; sourceTree = SOURCE_ROOT; };`
		configDir := filepath.Join(root, "Configs")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(config) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "App.xcconfig"), []byte("#include \"Shared.xcconfig\"\r\nOTHER_SETTING = YES\r\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(App.xcconfig) error = %v", err)
		}
		shared := "// leading comment\r\nMARKETING_VERSION = 1.2.3 // keep this comment\r\nCURRENT_PROJECT_VERSION = 42\r\nCURRENT_PROJECT_VERSION[sdk=iphoneos*] = 42\r\n/* MARKETING_VERSION = 8.8.8; */\r\n"
		if err := os.WriteFile(filepath.Join(configDir, "Shared.xcconfig"), []byte(shared), 0o640); err != nil {
			t.Fatalf("WriteFile(Shared.xcconfig) error = %v", err)
		}
	}

	project := `// !$*UTF8*$!
{
	archiveVersion = 1;
	classes = {};
	objectVersion = 77;
	objects = {
` + fileReference + `
		111111111111111111111111 /* Project object */ = {
			isa = PBXProject;
			attributes = {};
			buildConfigurationList = 222222222222222222222222;
			targets = (
				333333333333333333333333,
				444444444444444444444444,
			);
		};
		333333333333333333333333 /* App */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = 555555555555555555555555;
			buildPhases = ();
			dependencies = ();
			name = App;
			productName = App;
			productReference = 666666666666666666666666;
			productType = "com.apple.product-type.application";
		};
		444444444444444444444444 /* Widget */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = 777777777777777777777777;
			buildPhases = ();
			dependencies = ();
			name = Widget;
			productName = Widget;
			productReference = 888888888888888888888888;
			productType = "com.apple.product-type.app-extension";
		};
		666666666666666666666666 /* App.app */ = {isa = PBXFileReference; explicitFileType = wrapper.application; path = App.app; sourceTree = BUILT_PRODUCTS_DIR; };
		888888888888888888888888 /* Widget.appex */ = {isa = PBXFileReference; explicitFileType = "wrapper.app-extension"; path = Widget.appex; sourceTree = BUILT_PRODUCTS_DIR; };
		999999999999999999999991 /* Project Debug */ = {isa = XCBuildConfiguration; buildSettings = {}; name = Debug; };
		999999999999999999999992 /* Project Release */ = {isa = XCBuildConfiguration; buildSettings = {}; name = Release; };
		999999999999999999999993 /* App Debug */ = {isa = XCBuildConfiguration; ` + baseConfiguration + ` buildSettings = { ` + appSettings + ` }; name = Debug; };
		999999999999999999999994 /* App Release */ = {isa = XCBuildConfiguration; ` + baseConfiguration + ` buildSettings = { ` + appSettings + ` }; name = Release; };
		999999999999999999999995 /* Widget Debug */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Debug; };
		999999999999999999999996 /* Widget Release */ = {isa = XCBuildConfiguration; buildSettings = { MARKETING_VERSION = 1.2.3; CURRENT_PROJECT_VERSION = 42; }; name = Release; };
		222222222222222222222222 /* Project configuration list */ = {isa = XCConfigurationList; buildConfigurations = (999999999999999999999991, 999999999999999999999992); defaultConfigurationIsVisible = 0; defaultConfigurationName = Release; };
		555555555555555555555555 /* App configuration list */ = {isa = XCConfigurationList; buildConfigurations = (999999999999999999999993, 999999999999999999999994); defaultConfigurationIsVisible = 0; defaultConfigurationName = Release; };
		777777777777777777777777 /* Widget configuration list */ = {isa = XCConfigurationList; buildConfigurations = (999999999999999999999995, 999999999999999999999996); defaultConfigurationIsVisible = 0; defaultConfigurationName = Release; };
	};
	rootObject = 111111111111111111111111 /* Project object */;
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "project.pbxproj"), []byte(project), 0o644); err != nil {
		t.Fatalf("WriteFile(project.pbxproj) error = %v", err)
	}
	return projectPath
}
