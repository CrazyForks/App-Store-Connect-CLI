package shared

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestDeprecatedStringFlagAliasApply(t *testing.T) {
	tests := []struct {
		name         string
		canonicalArg string
		aliasArg     string
		want         string
		wantErr      string
		wantWarning  bool
		parseAlias   bool
	}{
		{name: "canonical only", canonicalArg: "canonical", want: "canonical"},
		{name: "alias only", aliasArg: "alias", want: "alias", wantWarning: true, parseAlias: true},
		{name: "matching values", canonicalArg: "same", aliasArg: "same", want: "same", wantWarning: true, parseAlias: true},
		{name: "conflicting values", canonicalArg: "canonical", aliasArg: "alias", want: "canonical", wantErr: "--legacy conflicts with --canonical", parseAlias: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			canonical := fs.String("canonical", "", "Canonical value")
			alias := BindDeprecatedStringFlagAlias(fs, "legacy", "canonical")

			args := []string{}
			if test.canonicalArg != "" {
				args = append(args, "--canonical", test.canonicalArg)
			}
			if test.parseAlias {
				args = append(args, "--legacy", test.aliasArg)
			}
			if err := fs.Parse(args); err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			_, stderr := captureOutput(t, func() {
				err := alias.Apply(canonical)
				if test.wantErr == "" && err != nil {
					t.Fatalf("Apply() error: %v", err)
				}
				if test.wantErr != "" {
					if err == nil || !strings.Contains(err.Error(), test.wantErr) {
						t.Fatalf("Apply() error = %v, want containing %q", err, test.wantErr)
					}
					if !errors.Is(err, flag.ErrHelp) {
						t.Fatalf("Apply() error = %v, want usage error", err)
					}
				}
			})

			if *canonical != test.want {
				t.Fatalf("canonical value = %q, want %q", *canonical, test.want)
			}
			if test.wantErr != "" && !strings.Contains(stderr, "Error: "+test.wantErr) {
				t.Fatalf("stderr = %q, want usage error", stderr)
			}
			if test.wantWarning && !strings.Contains(stderr, "Warning: `--legacy` is deprecated. Use `--canonical`.") {
				t.Fatalf("stderr = %q, want deprecation warning", stderr)
			}
			if test.wantErr == "" && !test.wantWarning && stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestBindDeprecatedStringFlagAliasHidesCompatibilityFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = fs.String("canonical", "", "Canonical value")
	BindDeprecatedStringFlagAlias(fs, "legacy", "canonical")

	if fs.Lookup("legacy") == nil {
		t.Fatal("compatibility flag was not registered")
	}
	for _, visible := range VisibleHelpFlags(fs) {
		if visible.Name == "legacy" {
			t.Fatal("compatibility flag should be hidden from canonical help")
		}
	}
}
