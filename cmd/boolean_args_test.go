package cmd

import (
	"flag"
	"reflect"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestNormalizeSpacedBooleanFlags(t *testing.T) {
	root := &ffcli.Command{
		Name:    "asc",
		FlagSet: flag.NewFlagSet("asc", flag.ContinueOnError),
	}
	commandFlags := flag.NewFlagSet("import", flag.ContinueOnError)
	commandFlags.Bool("confirm", false, "")
	commandFlags.Bool("dry-run", false, "")
	commandFlags.Bool("continue-on-error", true, "")
	commandFlags.String("input", "", "")
	importCommand := &ffcli.Command{
		Name:    "import",
		FlagSet: commandFlags,
	}
	stringSiblingFlags := flag.NewFlagSet("push", flag.ContinueOnError)
	stringSiblingFlags.String("continue-on-error", "", "")
	root.Subcommands = []*ffcli.Command{
		importCommand,
		{Name: "push", FlagSet: stringSiblingFlags},
	}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "explicit false before another flag",
			args: []string{"import", "--confirm", "false", "--dry-run"},
			want: []string{"import", "--confirm=false", "--dry-run"},
		},
		{
			name: "all boolean flags are normalized",
			args: []string{"import", "--confirm", "--continue-on-error", "false", "--dry-run", "true"},
			want: []string{"import", "--confirm", "--continue-on-error=false", "--dry-run=true"},
		},
		{
			name: "non-boolean values are untouched",
			args: []string{"import", "--input", "false", "--confirm"},
			want: []string{"import", "--input", "false", "--confirm"},
		},
		{
			name: "equals syntax is untouched",
			args: []string{"import", "--confirm=false", "--dry-run=true"},
			want: []string{"import", "--confirm=false", "--dry-run=true"},
		},
		{
			name: "positional boolean is untouched",
			args: []string{"import", "false"},
			want: []string{"import", "false"},
		},
		{
			name: "mixed sibling flag kinds use active command",
			args: []string{"import", "--continue-on-error", "false"},
			want: []string{"import", "--continue-on-error=false"},
		},
		{
			name: "flag terminator stops normalization",
			args: []string{"import", "--", "--confirm", "false"},
			want: []string{"import", "--", "--confirm", "false"},
		},
		{
			name: "flag-looking string value is untouched",
			args: []string{"import", "--input", "--confirm", "false"},
			want: []string{"import", "--input", "--confirm", "false"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeSpacedBooleanFlags(root, test.args)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("normalizeSpacedBooleanFlags() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRunNormalizesSpacedBooleansOnActiveCommandPath(t *testing.T) {
	resetReportFlags(t)
	tests := [][]string{
		{
			"subscriptions", "offers", "introductory", "import",
			"--confirm", "--continue-on-error", "false", "--help",
		},
		{
			"migrate", "import",
			"--confirm", "false", "--dry-run", "true", "--help",
		},
	}

	for _, args := range tests {
		_, stderr := captureCommandOutput(t, func() {
			if code := Run(args, "1.0.0"); code != ExitSuccess {
				t.Fatalf("Run(%v) exit code = %d, want success", args, code)
			}
		})
		if !strings.Contains(stderr, "USAGE") {
			t.Fatalf("Run(%v) stderr = %q, want command help", args, stderr)
		}
	}
}

func TestRunNormalizesSpacedRootBooleanBeforeLazyCommandDiscovery(t *testing.T) {
	resetReportFlags(t)
	_, stderr := captureCommandOutput(t, func() {
		args := []string{"--version", "false", "builds", "list", "--help"}
		if code := Run(args, "1.0.0"); code != ExitSuccess {
			t.Fatalf("Run(%v) exit code = %d, want success", args, code)
		}
	})
	if !strings.Contains(stderr, "asc builds list") {
		t.Fatalf("stderr = %q, want builds list help", stderr)
	}
}

func TestRunVersionRootFlagNeverDispatchesTrailingCommand(t *testing.T) {
	resetReportFlags(t)
	stdout, stderr := captureCommandOutput(t, func() {
		args := []string{"--version", "true", "builds", "list"}
		if code := Run(args, "9.8.7-test"); code != ExitSuccess {
			t.Fatalf("Run(%v) exit code = %d, want success", args, code)
		}
	})
	if stdout != "9.8.7-test\n" {
		t.Fatalf("stdout = %q, want version only", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want no nested-command output", stderr)
	}
}
