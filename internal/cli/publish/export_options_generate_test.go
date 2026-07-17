package publish

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	localxcode "github.com/rudrankriyam/App-Store-Connect-CLI/internal/xcode"
)

func TestPublishLocalBuildExportOptionsPrecedence(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	defaultPath := filepath.Join(workDir, defaultPublishExportOptionsPath)
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0o755); err != nil {
		t.Fatalf("create default export-options directory: %v", err)
	}
	if err := os.WriteFile(defaultPath, []byte("default plist"), 0o600); err != nil {
		t.Fatalf("write default export options: %v", err)
	}

	for _, tc := range []struct {
		name      string
		explicit  string
		defaultOK bool
		wantPath  string
	}{
		{
			name:      "testflight explicit path wins over existing default",
			explicit:  "ExplicitExportOptions.plist",
			defaultOK: true,
			wantPath:  "ExplicitExportOptions.plist",
		},
		{
			name:      "appstore existing default is reused",
			defaultOK: true,
			wantPath:  defaultPublishExportOptionsPath,
		},
		{
			name:      "appstore missing options are deferred for generation after archive",
			defaultOK: false,
			wantPath:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.defaultOK {
				if err := os.WriteFile(defaultPath, []byte("default plist"), 0o600); err != nil {
					t.Fatalf("restore default export options: %v", err)
				}
			} else if err := os.Remove(defaultPath); err != nil {
				t.Fatalf("remove default export options: %v", err)
			}

			config, err := resolveLocalBuildConfig(
				publishExportOptionsTestFlagValues(t, tc.explicit),
				"IOS",
				"1.2.3",
				"42",
			)
			if err != nil {
				t.Fatalf("resolveLocalBuildConfig() error: %v", err)
			}
			if config.ExportOptionsPath != tc.wantPath {
				t.Fatalf("expected export options path %q, got %q", tc.wantPath, config.ExportOptionsPath)
			}
		})
	}
}

func TestPublishLocalBuildGeneratesExportOptionsAfterArchive(t *testing.T) {
	restore := overridePublishCommandTestHooks(t)
	defer restore()

	originalGenerate := generatePublishExportOptionsFn
	t.Cleanup(func() {
		generatePublishExportOptionsFn = originalGenerate
	})

	artifactsDir := t.TempDir()
	archivePath := filepath.Join(artifactsDir, "Demo-IOS-1.2.3-42.xcarchive")
	ipaPath := filepath.Join(artifactsDir, "Demo-IOS-1.2.3-42.ipa")
	deterministicPath := localxcode.DefaultExportOptionsPathForArchive(archivePath)
	deterministicContents := []byte("pre-existing archive-adjacent options\n")
	if err := os.WriteFile(deterministicPath, deterministicContents, 0o600); err != nil {
		t.Fatalf("write deterministic export options: %v", err)
	}
	var generatedPath string
	callOrder := make([]string, 0, 4)

	runPublishArchiveFn = func(_ context.Context, opts localxcode.ArchiveOptions) (*localxcode.ArchiveResult, error) {
		callOrder = append(callOrder, "archive")
		if opts.ArchivePath != archivePath {
			t.Fatalf("expected archive path %q, got %q", archivePath, opts.ArchivePath)
		}
		return &localxcode.ArchiveResult{
			ArchivePath:   archivePath,
			BundleID:      "com.example.demo",
			Version:       "1.2.3",
			BuildNumber:   "42",
			Scheme:        "Demo",
			Configuration: "Release",
		}, nil
	}
	generatePublishExportOptionsFn = func(_ context.Context, opts localxcode.ExportOptionsGenerateOptions) (*localxcode.ExportOptionsGenerateResult, error) {
		callOrder = append(callOrder, "generate")
		if opts.ArchivePath != archivePath {
			t.Fatalf("expected generator archive path %q, got %q", archivePath, opts.ArchivePath)
		}
		if opts.OutputPath == deterministicPath {
			t.Fatalf("implicit generation must use a unique path, got deterministic path %q", opts.OutputPath)
		}
		if !strings.HasPrefix(opts.OutputPath, strings.TrimSuffix(deterministicPath, ".plist")+"-") || !strings.HasSuffix(opts.OutputPath, ".plist") {
			t.Fatalf("expected unique archive-adjacent output path, got %q", opts.OutputPath)
		}
		if opts.Destination != "export" {
			t.Fatalf("expected generator destination export, got %q", opts.Destination)
		}
		if opts.SigningStyle != "automatic" {
			t.Fatalf("expected automatic signing, got %q", opts.SigningStyle)
		}
		if opts.Overwrite {
			t.Fatal("implicit generation must not enable overwrite")
		}
		generatedPath = opts.OutputPath
		return &localxcode.ExportOptionsGenerateResult{Path: generatedPath}, nil
	}
	runPublishExportFn = func(_ context.Context, opts localxcode.ExportOptions) (*localxcode.ExportResult, error) {
		callOrder = append(callOrder, "export")
		if opts.ExportOptions != generatedPath {
			t.Fatalf("expected generated export options path %q, got %q", generatedPath, opts.ExportOptions)
		}
		return &localxcode.ExportResult{
			ArchivePath: archivePath,
			IPAPath:     ipaPath,
			BundleID:    "com.example.demo",
			Version:     "1.2.3",
			BuildNumber: "42",
		}, nil
	}
	validatePublishIPAPathFn = func(string) (os.FileInfo, error) {
		return newPublishTestFileInfo(t)
	}
	uploadBuildAndWaitForIDFn = func(_ context.Context, _ *asc.Client, _ string, _ string, _ os.FileInfo, version, buildNumber string, _ asc.Platform, _ time.Duration, _ time.Duration, _ bool) (*publishUploadResult, error) {
		callOrder = append(callOrder, "upload")
		return &publishUploadResult{Version: version, BuildNumber: buildNumber}, nil
	}

	result, err := runPublishLocalBuild(
		context.Background(),
		nil,
		"app-123",
		"IOS",
		"1.2.3",
		"42",
		5*time.Second,
		time.Minute,
		false,
		publishLocalBuildConfig{
			WorkspacePath:     "Demo.xcworkspace",
			Scheme:            "Demo",
			Configuration:     "Release",
			ArchivePath:       archivePath,
			IPAPath:           ipaPath,
			ExportOptionsPath: "",
		},
	)
	if err != nil {
		t.Fatalf("runPublishLocalBuild() error: %v", err)
	}
	if !reflect.DeepEqual(callOrder, []string{"archive", "generate", "export", "upload"}) {
		t.Fatalf("expected archive, generation, export, upload order; got %v", callOrder)
	}
	if result.Export == nil || result.Export.ExportOptionsPath != generatedPath {
		t.Fatalf("expected generated export options in result, got %#v", result.Export)
	}
	preserved, err := os.ReadFile(deterministicPath)
	if err != nil {
		t.Fatalf("read deterministic export options: %v", err)
	}
	if string(preserved) != string(deterministicContents) {
		t.Fatalf("implicit generation overwrote deterministic export options: %q", preserved)
	}
}

func publishExportOptionsTestFlagValues(t *testing.T, explicit string) *publishLocalBuildFlagValues {
	t.Helper()

	fs := flag.NewFlagSet("publish export-options test", flag.ContinueOnError)
	values := bindPublishLocalBuildFlags(fs)
	if err := fs.Parse([]string{
		"--workspace", "Demo.xcworkspace",
		"--scheme", "Demo",
		"--export-options", explicit,
	}); err != nil {
		t.Fatalf("parse local-build flags: %v", err)
	}
	return values
}
