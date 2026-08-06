package bundleids

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestBundleIDsGetCommand_MissingID(t *testing.T) {
	cmd := BundleIDsGetCommand()

	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsCreateCommand_MissingIdentifier(t *testing.T) {
	cmd := BundleIDsCreateCommand()

	if err := cmd.FlagSet.Parse([]string{"--name", "Example"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --identifier is missing, got %v", err)
	}
}

func TestBundleIDsCreateCommand_MissingName(t *testing.T) {
	cmd := BundleIDsCreateCommand()

	if err := cmd.FlagSet.Parse([]string{"--identifier", "com.example.app"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --name is missing, got %v", err)
	}
}

func TestBundleIDsUpdateCommand_MissingID(t *testing.T) {
	cmd := BundleIDsUpdateCommand()

	if err := cmd.FlagSet.Parse([]string{"--name", "New Name"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsUpdateCommand_MissingName(t *testing.T) {
	cmd := BundleIDsUpdateCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --name is missing, got %v", err)
	}
}

func TestBundleIDsDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := BundleIDsDeleteCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --confirm is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesListCommand_MissingBundle(t *testing.T) {
	cmd := BundleIDsCapabilitiesListCommand()

	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --bundle is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesAddCommand_MissingBundle(t *testing.T) {
	cmd := BundleIDsCapabilitiesAddCommand()

	if err := cmd.FlagSet.Parse([]string{"--capability", "ICLOUD"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --bundle is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesAddCommand_MissingCapability(t *testing.T) {
	cmd := BundleIDsCapabilitiesAddCommand()

	if err := cmd.FlagSet.Parse([]string{"--bundle", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --capability is missing, got %v", err)
	}
}

func TestParseCapabilitySettingsRejectsUnsupportedInput(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{
			name:    "unknown setting field",
			value:   `[{"key":"ICLOUD_VERSION","unexpected":true}]`,
			wantErr: `unknown field "unexpected"`,
		},
		{
			name:    "unknown option field",
			value:   `[{"key":"ICLOUD_VERSION","options":[{"key":"XCODE_6","unexpected":true}]}]`,
			wantErr: `unknown field "unexpected"`,
		},
		{
			name:    "unsupported setting key",
			value:   `[{"key":"APP_GROUP_IDS"}]`,
			wantErr: `unsupported capability setting key "APP_GROUP_IDS"`,
		},
		{
			name:    "unsupported option key",
			value:   `[{"key":"ICLOUD_VERSION","options":[{"key":"XCODE_13","enabled":true}]}]`,
			wantErr: `unsupported capability option key "XCODE_13"`,
		},
		{
			name:    "unsupported allowed instances",
			value:   `[{"key":"ICLOUD_VERSION","allowedInstances":"MANY"}]`,
			wantErr: `unsupported allowedInstances "MANY"`,
		},
		{
			name:    "null is not an array",
			value:   `null`,
			wantErr: `must be a JSON array, got null`,
		},
		{
			name:    "null nested field",
			value:   `[{"key":"ICLOUD_VERSION","options":null}]`,
			wantErr: `settings[0].options must not be null`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCapabilitySettings(tc.value)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseCapabilitySettingsAcceptsExactOpenAPIEnumsAndFields(t *testing.T) {
	settings, err := parseCapabilitySettings(`[
		{"key":"ICLOUD_VERSION","name":"iCloud","description":"version","enabledByDefault":true,"visible":true,"allowedInstances":"SINGLE","minInstances":1,"options":[
			{"key":"XCODE_5","name":"Xcode 5","description":"legacy","enabledByDefault":false,"enabled":false,"supportsWildcard":true},
			{"key":"XCODE_6","enabled":true}
		]},
		{"key":"DATA_PROTECTION_PERMISSION_LEVEL","allowedInstances":"ENTRY","options":[
			{"key":"COMPLETE_PROTECTION"},
			{"key":"PROTECTED_UNLESS_OPEN"},
			{"key":"PROTECTED_UNTIL_FIRST_USER_AUTH"}
		]},
		{"key":"APPLE_ID_AUTH_APP_CONSENT","allowedInstances":"MULTIPLE","options":[{"key":"PRIMARY_APP_CONSENT"}]}
	]`)
	if err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	if len(settings) != 3 {
		t.Fatalf("settings count = %d, want 3", len(settings))
	}
	if settings[0].MinInstances == nil || *settings[0].MinInstances != 1 {
		t.Fatalf("minInstances = %v, want 1", settings[0].MinInstances)
	}
	if settings[0].Options[0].Enabled == nil || *settings[0].Options[0].Enabled {
		t.Fatalf("explicit enabled=false was not preserved: %+v", settings[0].Options[0].Enabled)
	}
	if settings[0].Options[0].SupportsWildcard == nil || !*settings[0].Options[0].SupportsWildcard {
		t.Fatalf("supportsWildcard=true was not preserved: %+v", settings[0].Options[0].SupportsWildcard)
	}
}

func TestBundleIDsCapabilitiesHelpUsesSupportedSettings(t *testing.T) {
	for _, cmd := range []*ffcli.Command{
		BundleIDsCapabilitiesCommand(),
		BundleIDsCapabilitiesAddCommand(),
		BundleIDsCapabilitiesUpdateCommand(),
	} {
		if strings.Contains(cmd.LongHelp, "XCODE_9") || strings.Contains(cmd.LongHelp, "XCODE_13") {
			t.Fatalf("%s help advertises an unsupported Xcode capability option: %q", cmd.Name, cmd.LongHelp)
		}
		if !strings.Contains(cmd.LongHelp, "XCODE_6") {
			t.Fatalf("%s help does not include supported XCODE_6 option: %q", cmd.Name, cmd.LongHelp)
		}
	}
}

func TestBundleIDsCapabilitiesRemoveCommand_MissingID(t *testing.T) {
	cmd := BundleIDsCapabilitiesRemoveCommand()

	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesRemoveCommand_MissingConfirm(t *testing.T) {
	cmd := BundleIDsCapabilitiesRemoveCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "CAPABILITY_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp when --confirm is missing, got %v", err)
	}
}

func TestExtractBundleIDFromNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-123/profiles?cursor=abc"
	got, err := extractBundleIDFromNextURL(next)
	if err != nil {
		t.Fatalf("extractBundleIDFromNextURL() error: %v", err)
	}
	if got != "bundle-123" {
		t.Fatalf("expected bundle-123, got %q", got)
	}
}

func TestExtractBundleIDFromNextURLRelationships(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-123/relationships/profiles?cursor=abc"
	got, err := extractBundleIDFromNextURL(next)
	if err != nil {
		t.Fatalf("extractBundleIDFromNextURL() error: %v", err)
	}
	if got != "bundle-123" {
		t.Fatalf("expected bundle-123, got %q", got)
	}
}

func TestExtractBundleIDFromNextURL_Invalid(t *testing.T) {
	_, err := extractBundleIDFromNextURL("https://api.appstoreconnect.apple.com/v1/bundleIds")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExtractBundleIDFromNextURL_RejectsMalformedHost(t *testing.T) {
	tests := []string{
		"http://localhost:80:80/v1/bundleIds/bundle-123/profiles?cursor=abc",
		"http://::1/v1/bundleIds/bundle-123/profiles?cursor=abc",
	}

	for _, next := range tests {
		t.Run(next, func(t *testing.T) {
			if _, err := extractBundleIDFromNextURL(next); err == nil {
				t.Fatalf("expected error for malformed URL %q", next)
			}
		})
	}
}
