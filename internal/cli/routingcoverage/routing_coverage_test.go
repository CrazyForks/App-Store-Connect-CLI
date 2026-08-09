package routingcoverage

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/rootfs"
)

const validRoutingCoverageGeoJSON = `{"type":"MultiPolygon","coordinates":[[[[77.5,12.9],[77.7,12.9],[77.7,13.1],[77.5,12.9]]]]}`

func TestPrepareRoutingCoverageFileValidatesGeoJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "malformed JSON", content: `{"type":`, want: "invalid JSON"},
		{name: "wrong geometry type", content: `{"type":"Polygon","coordinates":[]}`, want: "MultiPolygon"},
		{name: "no polygons", content: `{"type":"MultiPolygon","coordinates":[]}`, want: "at least one Polygon"},
		{name: "short ring", content: `{"type":"MultiPolygon","coordinates":[[[[77.5,12.9],[77.7,12.9],[77.5,12.9]]]]}`, want: "at least four coordinate points"},
		{name: "open ring", content: `{"type":"MultiPolygon","coordinates":[[[[77.5,12.9],[77.7,12.9],[77.7,13.1],[77.6,12.8]]]]}`, want: "start and end coordinate points"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "coverage.geojson")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			_, err := PrepareRoutingCoverageFile(path)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("PrepareRoutingCoverageFile() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPrepareRoutingCoverageFileRejectsSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "coverage.geojson"), []byte(validRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}
	t.Chdir(root)

	_, err := PrepareRoutingCoverageFile(filepath.Join("linked", "coverage.geojson"))
	if !errors.Is(err, rootfs.ErrSymlink) {
		t.Fatalf("PrepareRoutingCoverageFile() error = %v, want rootfs.ErrSymlink", err)
	}
}

func TestPreparedRoutingCoverageFileRechecksRootedParent(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "coverage.geojson"), []byte(validRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Chdir(root)
	prepared, err := PrepareRoutingCoverageFile(filepath.Join("source", "coverage.geojson"))
	if err != nil {
		t.Fatalf("PrepareRoutingCoverageFile() error: %v", err)
	}

	if err := os.Rename(sourceDir, filepath.Join(root, "original")); err != nil {
		t.Fatalf("move source directory: %v", err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "coverage.geojson"), []byte(validRoutingCoverageGeoJSON), 0o600); err != nil {
		t.Fatalf("write replacement fixture: %v", err)
	}
	if err := os.Symlink(outside, sourceDir); err != nil {
		t.Fatalf("replace source with symlink: %v", err)
	}

	file, err := prepared.openSource()
	if file != nil {
		_ = file.Close()
	}
	if !errors.Is(err, rootfs.ErrSymlink) {
		t.Fatalf("openSource() error = %v, want rootfs.ErrSymlink", err)
	}
}

func TestRoutingCoverageCommandShape(t *testing.T) {
	cmd := RoutingCoverageCommand()
	if cmd == nil {
		t.Fatal("expected routing-coverage command")
		return
	}
	if cmd.Name != "routing-coverage" {
		t.Fatalf("unexpected command name: %q", cmd.Name)
	}
	if len(cmd.Subcommands) != 4 {
		t.Fatalf("expected 4 subcommands, got %d", len(cmd.Subcommands))
	}
}

func TestRoutingCoverageGetCommand_MissingVersionID(t *testing.T) {
	cmd := RoutingCoverageGetCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestRoutingCoverageInfoCommand_MissingID(t *testing.T) {
	cmd := RoutingCoverageInfoCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestRoutingCoverageCreateCommand_MissingRequiredFlags(t *testing.T) {
	t.Run("missing version-id", func(t *testing.T) {
		cmd := RoutingCoverageCreateCommand()
		if err := cmd.FlagSet.Parse([]string{"--file", "coverage.geojson"}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		cmd := RoutingCoverageCreateCommand()
		if err := cmd.FlagSet.Parse([]string{"--version-id", "VERSION_ID"}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})
}

func TestRoutingCoverageDeleteCommandValidation(t *testing.T) {
	t.Run("missing id", func(t *testing.T) {
		cmd := RoutingCoverageDeleteCommand()
		if err := cmd.FlagSet.Parse([]string{"--confirm"}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})

	t.Run("missing confirm", func(t *testing.T) {
		cmd := RoutingCoverageDeleteCommand()
		if err := cmd.FlagSet.Parse([]string{"--id", "COVERAGE_ID"}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected flag.ErrHelp, got %v", err)
		}
	})
}

func TestCommandWrapper(t *testing.T) {
	if got := RoutingCoverageCommand(); got == nil {
		t.Fatal("expected Command wrapper to return command")
	}
}
