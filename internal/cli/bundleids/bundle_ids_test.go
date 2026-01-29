package bundleids

import (
	"context"
	"flag"
	"testing"
)

func TestBundleIDsGetCommand_MissingID(t *testing.T) {
	cmd := BundleIDsGetCommand()

	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsCreateCommand_MissingIdentifier(t *testing.T) {
	cmd := BundleIDsCreateCommand()

	if err := cmd.FlagSet.Parse([]string{"--name", "Example"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --identifier is missing, got %v", err)
	}
}

func TestBundleIDsCreateCommand_MissingName(t *testing.T) {
	cmd := BundleIDsCreateCommand()

	if err := cmd.FlagSet.Parse([]string{"--identifier", "com.example.app"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --name is missing, got %v", err)
	}
}

func TestBundleIDsUpdateCommand_MissingID(t *testing.T) {
	cmd := BundleIDsUpdateCommand()

	if err := cmd.FlagSet.Parse([]string{"--name", "New Name"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsUpdateCommand_MissingName(t *testing.T) {
	cmd := BundleIDsUpdateCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --name is missing, got %v", err)
	}
}

func TestBundleIDsDeleteCommand_MissingConfirm(t *testing.T) {
	cmd := BundleIDsDeleteCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --confirm is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesListCommand_MissingBundle(t *testing.T) {
	cmd := BundleIDsCapabilitiesListCommand()

	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --bundle is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesAddCommand_MissingBundle(t *testing.T) {
	cmd := BundleIDsCapabilitiesAddCommand()

	if err := cmd.FlagSet.Parse([]string{"--capability", "ICLOUD"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --bundle is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesAddCommand_MissingCapability(t *testing.T) {
	cmd := BundleIDsCapabilitiesAddCommand()

	if err := cmd.FlagSet.Parse([]string{"--bundle", "BUNDLE_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --capability is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesRemoveCommand_MissingID(t *testing.T) {
	cmd := BundleIDsCapabilitiesRemoveCommand()

	if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --id is missing, got %v", err)
	}
}

func TestBundleIDsCapabilitiesRemoveCommand_MissingConfirm(t *testing.T) {
	cmd := BundleIDsCapabilitiesRemoveCommand()

	if err := cmd.FlagSet.Parse([]string{"--id", "CAPABILITY_ID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if err := cmd.Exec(context.Background(), []string{}); err != flag.ErrHelp {
		t.Fatalf("expected flag.ErrHelp when --confirm is missing, got %v", err)
	}
}
