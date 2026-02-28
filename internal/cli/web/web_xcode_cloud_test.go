package web

import (
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
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
