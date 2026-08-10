package cmdtest

import (
	"strings"
	"testing"
)

func TestReviewDeprecatedItemSurfacesAreRemoved(t *testing.T) {
	root := RootCommand("4.0.0")

	for _, path := range [][]string{
		{"review", "items", "view"},
		{"review", "items-get"},
	} {
		if command := findSubcommand(root, path...); command != nil {
			t.Fatalf("deprecated command %q is still registered", strings.Join(path, " "))
		}
	}

	for _, path := range [][]string{
		{"review", "items", "update"},
		{"review", "items-update"},
	} {
		command := findSubcommand(root, path...)
		if command == nil {
			t.Fatalf("supported command %q is not registered", strings.Join(path, " "))
		}
		if command.FlagSet.Lookup("state") != nil {
			t.Fatalf("deprecated --state flag is still registered on %q", strings.Join(path, " "))
		}
		for _, flagName := range []string{"resolved", "removed", "clear-resolved", "clear-removed"} {
			if command.FlagSet.Lookup(flagName) == nil {
				t.Fatalf("supported --%s flag is not registered on %q", flagName, strings.Join(path, " "))
			}
		}
	}
}
