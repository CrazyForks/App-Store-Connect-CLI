package xcode

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	localxcode "github.com/rudrankriyam/App-Store-Connect-CLI/internal/xcode"
)

func TestXcodeBuildPassesTypedAndRawOptionsAndPrintsJSON(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })

	var gotOpts localxcode.BuildOptions
	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		gotOpts = opts
		exitStatus := 0
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
			ExitStatus:        &exitStatus,
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
	if !payload.Success || payload.DurationMS != 1250 || !payload.NoCodeSigning || payload.ExitStatus == nil || *payload.ExitStatus != 0 {
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
		{name: "conditional signing override", args: []string{"--project", "Demo.xcodeproj", "--scheme", "Demo", "--no-code-signing", "--xcodebuild-flag=CODE_SIGNING_ALLOWED[sdk=iphoneos*]=YES"}, want: `cannot override asc-managed argument "CODE_SIGNING_ALLOWED"`},
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
		_, _ = io.WriteString(opts.LogWriter, "compile failed\n")
		exitStatus := 65
		return &localxcode.BuildResult{
			ProjectPath:     opts.ProjectPath,
			Scheme:          opts.Scheme,
			DerivedDataPath: "/tmp/derived",
			NoCodeSigning:   false,
			Success:         false,
			DurationMS:      400,
			ExitStatus:      &exitStatus,
		}, errors.New("xcodebuild build failed: compile failed")
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
	if runErr == nil || !strings.Contains(runErr.Error(), "xcodebuild build failed: compile failed") {
		t.Fatalf("Exec() error = %v, want wrapped build failure", runErr)
	}
	var reportedErr shared.ReportedError
	if !errors.As(runErr, &reportedErr) {
		t.Fatalf("Exec() error = %T %v, want ReportedError", runErr, runErr)
	}
	if got := strings.Count(stderr, "compile failed"); got != 1 {
		t.Fatalf("stderr = %q, compile diagnostic count = %d, want 1", stderr, got)
	}
	if !strings.Contains(stderr, "Error: xcode build failed with exit status 65") {
		t.Fatalf("stderr = %q, want concise final build error", stderr)
	}
	var payload localxcode.BuildResult
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout=%s", err, stdout)
	}
	if payload.Success || payload.ExitStatus == nil || *payload.ExitStatus != 65 {
		t.Fatalf("unexpected failure payload: %+v", payload)
	}
}

func TestXcodeBuildPrintsPreflightFailureReason(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })

	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		return &localxcode.BuildResult{
			ProjectPath:     opts.ProjectPath,
			Scheme:          opts.Scheme,
			DerivedDataPath: "/tmp/derived",
			Success:         false,
			DurationMS:      1,
		}, errors.New("xcodebuild not usable: xcodebuild version failed: exit status 72")
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
	var reportedErr shared.ReportedError
	if !errors.As(runErr, &reportedErr) {
		t.Fatalf("Exec() error = %T %v, want ReportedError", runErr, runErr)
	}
	if got := strings.Count(stderr, "xcodebuild not usable"); got != 1 {
		t.Fatalf("stderr = %q, preflight reason count = %d, want 1", stderr, got)
	}
	if strings.Contains(stderr, "xcode build failed with exit status 72") {
		t.Fatalf("stderr = %q, version-probe status must not be reported as build status", stderr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nstdout=%s", err, stdout)
	}
	if _, exists := payload["exit_status"]; exists {
		t.Fatalf("unexpected exit_status for preflight failure: %s", stdout)
	}
}

func TestXcodeBuildRendersTableAndMarkdown(t *testing.T) {
	originalRunBuild := runBuild
	t.Cleanup(func() { runBuild = originalRunBuild })
	runBuild = func(_ context.Context, opts localxcode.BuildOptions) (*localxcode.BuildResult, error) {
		exitStatus := 0
		return &localxcode.BuildResult{
			WorkspacePath:   opts.WorkspacePath,
			Scheme:          opts.Scheme,
			Destination:     opts.Destination,
			DerivedDataPath: "/tmp/derived",
			NoCodeSigning:   false,
			Success:         true,
			DurationMS:      10,
			ExitStatus:      &exitStatus,
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
			for _, want := range []string{"workspace", "Demo.xcworkspace", "destination", "generic/platform=iOS", "success", "exit_status", "0"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("%s output = %q, want %q", format, stdout, want)
				}
			}
		})
	}
}
