package asc

import (
	"strings"
	"testing"
)

func TestPrintTable_PassTypeIDs(t *testing.T) {
	resp := &PassTypeIDsResponse{
		Data: []Resource[PassTypeIDAttributes]{
			{
				ID: "p1",
				Attributes: PassTypeIDAttributes{
					Name:       "Example",
					Identifier: "pass.com.example",
				},
			},
		},
	}

	output := captureStdout(t, func() error {
		return PrintTable(resp)
	})

	if !strings.Contains(output, "Identifier") || !strings.Contains(output, "Name") {
		t.Fatalf("expected header in output, got: %s", output)
	}
	if !strings.Contains(output, "pass.com.example") {
		t.Fatalf("expected identifier in output, got: %s", output)
	}
}

func TestPrintMarkdown_PassTypeIDs(t *testing.T) {
	resp := &PassTypeIDsResponse{
		Data: []Resource[PassTypeIDAttributes]{
			{
				ID: "p1",
				Attributes: PassTypeIDAttributes{
					Name:       "Example",
					Identifier: "pass.com.example",
				},
			},
		},
	}

	output := captureStdout(t, func() error {
		return PrintMarkdown(resp)
	})

	if !strings.Contains(output, "| ID | Name | Identifier |") {
		t.Fatalf("expected markdown header, got: %s", output)
	}
	if !strings.Contains(output, "pass.com.example") {
		t.Fatalf("expected identifier in output, got: %s", output)
	}
}

func TestPrintTable_PassTypeIDDeleteResult(t *testing.T) {
	result := &PassTypeIDDeleteResult{ID: "p1", Deleted: true}

	output := captureStdout(t, func() error {
		return PrintTable(result)
	})

	if !strings.Contains(output, "Deleted") {
		t.Fatalf("expected header in output, got: %s", output)
	}
	if !strings.Contains(output, "p1") {
		t.Fatalf("expected ID in output, got: %s", output)
	}
}

func TestPrintMarkdown_PassTypeIDDeleteResult(t *testing.T) {
	result := &PassTypeIDDeleteResult{ID: "p1", Deleted: true}

	output := captureStdout(t, func() error {
		return PrintMarkdown(result)
	})

	if !strings.Contains(output, "| ID | Deleted |") {
		t.Fatalf("expected markdown header, got: %s", output)
	}
	if !strings.Contains(output, "p1") {
		t.Fatalf("expected ID in output, got: %s", output)
	}
}
