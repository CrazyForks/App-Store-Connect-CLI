package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"testing"
)

func TestTestFlightBetaLicenseAgreementsGetValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing id and app",
			args: []string{"testflight", "beta-license-agreements", "get"},
		},
		{
			name: "id and app both set",
			args: []string{"testflight", "beta-license-agreements", "get", "--id", "AGREEMENT_ID", "--app", "APP_ID"},
		},
		{
			name: "app with include",
			args: []string{"testflight", "beta-license-agreements", "get", "--app", "APP_ID", "--include", "app"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, _ := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
		})
	}
}

func TestTestFlightBetaLicenseAgreementsUpdateValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing id",
			args: []string{"testflight", "beta-license-agreements", "update", "--agreement-text", "Updated"},
		},
		{
			name: "missing agreement text",
			args: []string{"testflight", "beta-license-agreements", "update", "--id", "AGREEMENT_ID"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, _ := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
		})
	}
}

func TestTestFlightBetaLicenseAgreementsListLimitValidation(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "beta-license-agreements", "list", "--limit", "500"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
}
