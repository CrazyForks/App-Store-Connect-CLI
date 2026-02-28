package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

func TestValidateDateFlagValidDates(t *testing.T) {
	tests := []string{"2026-01-01", "2025-12-31", "2000-06-15"}
	for _, d := range tests {
		if err := validateDateFlag("--start", d); err != nil {
			t.Fatalf("validateDateFlag(%q) unexpected error: %v", d, err)
		}
	}
}

func TestValidateDateFlagRejectsEmpty(t *testing.T) {
	err := validateDateFlag("--start", "")
	if err == nil {
		t.Fatal("expected error for empty date")
	}
	if !strings.Contains(err.Error(), "--start is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDateFlagRejectsInvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"wrong format", "01-01-2026"},
		{"not a date", "foobar"},
		{"month-day only", "01-01"},
		{"slash separator", "2026/01/01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateFlag("--end", tt.value)
			if err == nil {
				t.Fatalf("expected error for %q", tt.value)
			}
			if !strings.Contains(err.Error(), "must be YYYY-MM-DD") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWebXcodeCloudCommandHierarchy(t *testing.T) {
	cmd := WebXcodeCloudCommand()
	if cmd.Name != "xcode-cloud" {
		t.Fatalf("expected command name %q, got %q", "xcode-cloud", cmd.Name)
	}
	if len(cmd.Subcommands) != 2 {
		t.Fatalf("expected 2 subcommands (usage, products), got %d", len(cmd.Subcommands))
	}

	names := map[string]bool{}
	for _, sub := range cmd.Subcommands {
		names[sub.Name] = true
	}
	if !names["usage"] {
		t.Fatal("expected 'usage' subcommand")
	}
	if !names["products"] {
		t.Fatal("expected 'products' subcommand")
	}
}

func TestWebXcodeCloudUsageSubcommands(t *testing.T) {
	cmd := WebXcodeCloudCommand()
	usageCmd := findSub(cmd, "usage")
	if usageCmd == nil {
		t.Fatal("could not find 'usage' subcommand")
	}
	if len(usageCmd.Subcommands) != 3 {
		t.Fatalf("expected 3 usage subcommands, got %d", len(usageCmd.Subcommands))
	}
	usageNames := map[string]bool{}
	for _, sub := range usageCmd.Subcommands {
		usageNames[sub.Name] = true
	}
	for _, expected := range []string{"summary", "months", "days"} {
		if !usageNames[expected] {
			t.Fatalf("expected %q usage subcommand", expected)
		}
	}
}

func TestWebXcodeCloudSubcommandsResolveSessionWithinTimeoutContext(t *testing.T) {
	origResolveSession := resolveSessionFn
	t.Cleanup(func() {
		resolveSessionFn = origResolveSession
	})

	resolveErr := errors.New("stop before network call")
	tests := []struct {
		name  string
		build func() *ffcli.Command
		args  []string
	}{
		{
			name:  "usage summary",
			build: webXcodeCloudUsageSummaryCommand,
			args:  []string{"--apple-id", "user@example.com"},
		},
		{
			name:  "usage months",
			build: webXcodeCloudUsageMonthsCommand,
			args:  []string{"--apple-id", "user@example.com"},
		},
		{
			name:  "usage days",
			build: webXcodeCloudUsageDaysCommand,
			args:  []string{"--apple-id", "user@example.com", "--product-id", "product-123"},
		},
		{
			name:  "products",
			build: webXcodeCloudProductsCommand,
			args:  []string{"--apple-id", "user@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hadDeadline := false
			resolveSessionFn = func(
				ctx context.Context,
				appleID, password, twoFactorCode string,
				usePasswordStdin bool,
			) (*webcore.AuthSession, string, error) {
				_, hadDeadline = ctx.Deadline()
				return nil, "", resolveErr
			}

			cmd := tt.build()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}

			err := cmd.Exec(context.Background(), nil)
			if !errors.Is(err, resolveErr) {
				t.Fatalf("expected resolveSession error %v, got %v", resolveErr, err)
			}
			if !hadDeadline {
				t.Fatal("expected resolveSession to receive a timeout context")
			}
		})
	}
}

func TestWebXcodeCloudUsageSummaryOutputTableUsesHumanRenderer(t *testing.T) {
	origResolveSession := resolveSessionFn
	t.Cleanup(func() {
		resolveSessionFn = origResolveSession
	})

	resolveSessionFn = func(
		ctx context.Context,
		appleID, password, twoFactorCode string,
		usePasswordStdin bool,
	) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			PublicProviderID: "team-uuid",
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					body := `{"plan":{"name":"Plan","reset_date":"2026-03-27","reset_date_time":"2026-03-27T07:26:10Z","available":1500,"used":0,"total":1500},"links":{"manage":"https://developer.apple.com/xcode-cloud/"}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(body)),
						Request:    req,
					}, nil
				}),
			},
		}, "cache", nil
	}

	cmd := webXcodeCloudUsageSummaryCommand()
	if err := cmd.FlagSet.Parse([]string{"--apple-id", "user@example.com", "--output", "table"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, `"plan"`) {
		t.Fatalf("expected table output, got json: %q", stdout)
	}
	for _, token := range []string{"Plan", "Available", "1500"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("expected table output to include %q, got %q", token, stdout)
		}
	}
}

func TestWebXcodeCloudUsageDaysFlagSet(t *testing.T) {
	cmd := WebXcodeCloudCommand()
	daysCmd := findSub(findSub(cmd, "usage"), "days")
	if daysCmd == nil {
		t.Fatal("could not find 'usage days' subcommand")
	}

	fs := daysCmd.FlagSet
	if fs == nil {
		t.Fatal("expected flag set on days command")
	}

	for _, name := range []string{"product-id", "start", "end"} {
		if fs.Lookup(name) == nil {
			t.Fatalf("expected --%s flag", name)
		}
	}
}

func TestWebXcodeCloudUsageMonthsFlagSet(t *testing.T) {
	cmd := WebXcodeCloudCommand()
	monthsCmd := findSub(findSub(cmd, "usage"), "months")
	if monthsCmd == nil {
		t.Fatal("could not find 'usage months' subcommand")
	}

	fs := monthsCmd.FlagSet
	for _, name := range []string{"start-month", "start-year", "end-month", "end-year"} {
		if fs.Lookup(name) == nil {
			t.Fatalf("expected --%s flag", name)
		}
	}
}

func TestWebXcodeCloudAllCommandsHaveUsageFunc(t *testing.T) {
	cmd := WebXcodeCloudCommand()
	if cmd.UsageFunc == nil {
		t.Fatal("expected UsageFunc on xcode-cloud command")
	}
	for _, sub := range cmd.Subcommands {
		if sub.UsageFunc == nil {
			t.Fatalf("expected UsageFunc on %q subcommand", sub.Name)
		}
		for _, subsub := range sub.Subcommands {
			if subsub.UsageFunc == nil {
				t.Fatalf("expected UsageFunc on %q subcommand", subsub.Name)
			}
		}
	}
}

func findSub(cmd *ffcli.Command, name string) *ffcli.Command {
	if cmd == nil {
		return nil
	}
	for _, sub := range cmd.Subcommands {
		if sub.Name == name {
			return sub
		}
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = wOut.Close()
		_ = wErr.Close()
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()

	stdout := <-outC
	stderr := <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdout, stderr
}
