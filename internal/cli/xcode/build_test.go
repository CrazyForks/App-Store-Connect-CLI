package xcode

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	localxcode "github.com/rudrankriyam/App-Store-Connect-CLI/internal/xcode"
)

func TestXcodeBuildPassesTypedAndRawOptionsAndPrintsJSON(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })

	var gotOpts localxcode.BuildOptions
	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		gotOpts = opts
		return &localxcode.BuildResult{
			ProjectPath:       opts.ProjectPath,
			Scheme:            opts.Scheme,
			Configuration:     opts.Configuration,
			Destination:       opts.Destination,
			DerivedDataPath:   opts.DerivedDataPath,
			BuildProductsPath: "/tmp/Derived Data/Build/Products",
			Clean:             opts.Clean,
			NoCodeSigning:     opts.NoCodeSigning,
			Success:           true,
			DurationMS:        1250,
		}, nil
	}

	cmd := XcodeBuildCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--project", "Demo App.xcodeproj",
		"--scheme", "Demo App",
		"--configuration", "Debug",
		"--destination", "platform=iOS Simulator,name=iPhone 17 Pro Max,OS=27.0",
		"--derived-data-path", "/tmp/Derived Data",
		"--clean",
		"--no-code-signing",
		"--xcodebuild-flag=-quiet",
		"--xcodebuild-flag=OTHER_SWIFT_FLAGS=-D ASC_BUILD",
		"--output", "json",
	}); err != nil {
		t.Fatalf("FlagSet.Parse() error = %v", err)
	}

	var runErr error
	stdout, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if runErr != nil {
		t.Fatalf("Exec() error = %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if gotOpts.ProjectPath != "Demo App.xcodeproj" || gotOpts.WorkspacePath != "" || gotOpts.Scheme != "Demo App" {
		t.Fatalf("unexpected selector options: %+v", gotOpts)
	}
	if !gotOpts.Clean || !gotOpts.NoCodeSigning {
		t.Fatalf("expected clean and no-code-signing: %+v", gotOpts)
	}
	wantRaw := []string{"-quiet", "OTHER_SWIFT_FLAGS=-D ASC_BUILD"}
	if len(gotOpts.XcodebuildArgs) != len(wantRaw) || gotOpts.XcodebuildArgs[0] != wantRaw[0] || gotOpts.XcodebuildArgs[1] != wantRaw[1] {
		t.Fatalf("XcodebuildArgs = %#v, want %#v", gotOpts.XcodebuildArgs, wantRaw)
	}
	var payload localxcode.BuildResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout=%s", err, stdout)
	}
	if !payload.Success || payload.DurationMS != 1250 || !payload.NoCodeSigning {
		t.Fatalf("unexpected JSON payload: %+v", payload)
	}
}

func TestXcodeBuildValidationErrorsAreUsageErrors(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		positional []string
		want       string
	}{
		{name: "missing selector", args: []string{"--scheme", "Demo"}, want: "exactly one of --workspace or --project"},
		{name: "both selectors", args: []string{"--project", "Demo.xcodeproj", "--workspace", "Demo.xcworkspace", "--scheme", "Demo"}, want: "exactly one of --workspace or --project"},
		{name: "missing scheme", args: []string{"--project", "Demo.xcodeproj"}, want: "--scheme is required"},
		{name: "bad project suffix", args: []string{"--project", "Demo.txt", "--scheme", "Demo"}, want: "--project must end with .xcodeproj"},
		{name: "reserved raw flag", args: []string{"--project", "Demo.xcodeproj", "--scheme", "Demo", "--xcodebuild-flag=-derivedDataPath"}, want: "cannot override asc-managed argument"},
		{name: "positional", args: []string{"--project", "Demo.xcodeproj", "--scheme", "Demo"}, positional: []string{"build"}, want: "does not accept positional arguments"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := XcodeBuildCommand()
			cmd.FlagSet.SetOutput(io.Discard)
			if err := cmd.FlagSet.Parse(test.args); err != nil {
				t.Fatalf("FlagSet.Parse() error = %v", err)
			}
			var runErr error
			stdout, stderr := captureCommandOutput(t, func() error {
				runErr = cmd.Exec(context.Background(), test.positional)
				return runErr
			})
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("Exec() error = %v, want usage error", runErr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, test.want) {
				t.Fatalf("stderr = %q, want %q", stderr, test.want)
			}
		})
	}
}

func TestXcodeBuildRejectsInvalidOutputBeforeStartingBuild(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })
	runBuild = func(context.Context, localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		t.Fatal("runBuild must not be called for invalid output")
		return nil, nil
	}

	cmd := XcodeBuildCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{"--project", "Demo.xcodeproj", "--scheme", "Demo", "--output", "yaml"}); err != nil {
		t.Fatalf("FlagSet.Parse() error = %v", err)
	}
	var runErr error
	stdout, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("Exec() error = %v, want usage error", runErr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unsupported format: yaml") {
		t.Fatalf("stderr = %q, want unsupported format error", stderr)
	}
}

func TestXcodeBuildPrintsStructuredFailureBeforeReturningError(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })

	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		return &localxcode.BuildResult{
			ProjectPath:     opts.ProjectPath,
			Scheme:          opts.Scheme,
			DerivedDataPath: "/tmp/derived",
			NoCodeSigning:   false,
			Success:         false,
			DurationMS:      400,
			ExitStatus:      65,
		}, errors.New("compile failed")
	}

	cmd := XcodeBuildCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{"--project", "Demo.xcodeproj", "--scheme", "Demo", "--output", "json"}); err != nil {
		t.Fatalf("FlagSet.Parse() error = %v", err)
	}
	var runErr error
	stdout, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "xcode build: compile failed") {
		t.Fatalf("Exec() error = %v, want wrapped build failure", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want subprocess logger to own diagnostics", stderr)
	}
	var payload localxcode.BuildResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout=%s", err, stdout)
	}
	if payload.Success || payload.ExitStatus != 65 {
		t.Fatalf("unexpected failure payload: %+v", payload)
	}
}

func TestXcodeBuildRendersTableAndMarkdown(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })
	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		return &localxcode.BuildResult{
			WorkspacePath:   opts.WorkspacePath,
			Scheme:          opts.Scheme,
			Destination:     opts.Destination,
			DerivedDataPath: "/tmp/derived",
			NoCodeSigning:   false,
			Success:         true,
			DurationMS:      10,
		}, nil
	}

	for _, format := range []string{"table", "markdown"} {
		t.Run(format, func(t *testing.T) {
			cmd := XcodeBuildCommand()
			cmd.FlagSet.SetOutput(io.Discard)
			if err := cmd.FlagSet.Parse([]string{
				"--workspace", "Demo.xcworkspace", "--scheme", "Demo",
				"--destination", "generic/platform=iOS", "--output", format,
			}); err != nil {
				t.Fatalf("FlagSet.Parse() error = %v", err)
			}
			var runErr error
			stdout, _ := captureCommandOutput(t, func() error {
				runErr = cmd.Exec(context.Background(), nil)
				return runErr
			})
			if runErr != nil {
				t.Fatalf("Exec() error = %v", runErr)
			}
			for _, want := range []string{"workspace", "Demo.xcworkspace", "destination", "generic/platform=iOS", "success"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("%s output = %q, want %q", format, stdout, want)
				}
			}
		})
	}
}
