package wallgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func TestGenerateWritesReadmeSnippet(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Zulu App",
    "link": "https://example.com/zulu",
    "creator": "Zulu Creator",
    "platform": ["iOS"]
  },
  {
    "app": "Alpha App",
    "link": "https://example.com/alpha",
    "creator": "Alpha Creator",
    "platform": ["macOS", "iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo

Before.
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
After.
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	alphaRow := "| Alpha App | [Open](https://example.com/alpha) | Alpha Creator | macOS, iOS |"
	zuluRow := "| Zulu App | [Open](https://example.com/zulu) | Zulu Creator | iOS |"

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)
	if strings.Contains(readme, "Old content.") {
		t.Fatalf("expected README snippet to be replaced, got:\n%s", readme)
	}
	if !strings.Contains(readme, "| App | Link | Creator | Platform |") {
		t.Fatalf("expected markdown table header in README, got:\n%s", readme)
	}
	if !strings.Contains(readme, alphaRow) || !strings.Contains(readme, zuluRow) {
		t.Fatalf("expected README to include generated rows, got:\n%s", readme)
	}
	alphaIdx := strings.Index(readme, alphaRow)
	zuluIdx := strings.Index(readme, zuluRow)
	if alphaIdx > zuluIdx {
		t.Fatalf("expected deterministic app sorting in README, got:\n%s", readme)
	}
	if _, err := os.Stat(filepath.Join(tmpRepo, "docs", "generated", "app-wall.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated wall file, stat error: %v", err)
	}
}

func TestGenerateFailsWhenCreatorMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "No Creator App",
    "link": "https://example.com/no-creator",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail for missing creator")
	}
	if !strings.Contains(err.Error(), "'creator' is required") {
		t.Fatalf("expected missing creator error, got %v", err)
	}
}

func TestGenerateAcceptsCustomPlatformLabels(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Platform App",
    "link": "https://example.com/platform",
    "creator": "Platform Creator",
    "platform": ["Android", "WATCH_OS", "watchOS", "TV_OS", "tvos", "IOS", "ios", "Vision OS", "visionos"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	generatedContentBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	generatedContent := string(generatedContentBytes)
	row := "| Platform App | [Open](https://example.com/platform) | Platform Creator | Android, watchOS, tvOS, iOS, visionOS |"
	if !strings.Contains(generatedContent, row) {
		t.Fatalf("expected row with custom and normalized platform labels, got:\n%s", generatedContent)
	}
}

func TestGenerateFailsWhenPlatformEntryEmpty(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Bad Platform App",
    "link": "https://example.com/bad-platform",
    "creator": "Platform Creator",
    "platform": ["iOS", "   "]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail for empty platform value")
	}
	if !strings.Contains(err.Error(), "'platform' entries must be non-empty strings") {
		t.Fatalf("expected empty platform entry error, got %v", err)
	}
}

func TestGenerateFailsWhenReadmeMarkersMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Marker App",
    "link": "https://example.com/marker",
    "creator": "Marker Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), "# Demo\nNo wall markers here.\n")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when README markers are missing")
	}
	if !strings.Contains(err.Error(), "README markers not found") {
		t.Fatalf("expected README marker error, got %v", err)
	}
}

func TestGenerateFailsWhenReadmeMarkersOutOfOrder(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Marker App",
    "link": "https://example.com/marker",
    "creator": "Marker Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:END -->
Old content.
<!-- WALL-OF-APPS:START -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when README markers are out of order")
	}
	if !strings.Contains(err.Error(), "README markers not found") {
		t.Fatalf("expected README marker error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source file is missing")
	}
	if !strings.Contains(err.Error(), "missing source file") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileEmpty(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "   \n")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source file is empty")
	}
	if !strings.Contains(err.Error(), "source file is empty") {
		t.Fatalf("expected empty source error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileInvalidJSON(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "{invalid")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source JSON is invalid")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileHasNoEntries(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "[]")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source has no entries")
	}
	if !strings.Contains(err.Error(), "source file has no entries") {
		t.Fatalf("expected no entries error, got %v", err)
	}
}

func TestGenerateValidatesLinkURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		link string
	}{
		{name: "missing scheme", link: "example.com/app"},
		{name: "unsupported scheme", link: "ftp://example.com/app"},
		{name: "missing host", link: "https:///path-only"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpRepo := t.TempDir()

			writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "URL App",
    "link": "`+tc.link+`",
    "creator": "URL Creator",
    "platform": ["iOS"]
  }
]`)
			writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

			_, err := Generate(tmpRepo)
			if err == nil {
				t.Fatal("expected generate to fail for invalid link")
			}
			if !strings.Contains(err.Error(), "'link' must be a valid http/https URL") {
				t.Fatalf("expected link validation error, got %v", err)
			}
		})
	}
}

func TestGenerateSortsByLinkWhenAppNamesMatch(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Same App",
    "link": "https://b.example.com",
    "creator": "Creator B",
    "platform": ["iOS"]
  },
  {
    "app": "Same App",
    "link": "https://a.example.com",
    "creator": "Creator A",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	rowA := "| Same App | [Open](https://a.example.com) | Creator A | iOS |"
	rowB := "| Same App | [Open](https://b.example.com) | Creator B | iOS |"
	if !strings.Contains(readme, rowA) || !strings.Contains(readme, rowB) {
		t.Fatalf("expected both rows in README, got:\n%s", readme)
	}
	if strings.Index(readme, rowA) > strings.Index(readme, rowB) {
		t.Fatalf("expected same-name apps to be sorted by link, got:\n%s", readme)
	}
}

func TestGenerateEscapesMarkdownCells(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Pipe|App\nName",
    "link": "https://example.com/pipe",
    "creator": "Creator|Team\nOne",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	expectedRow := "| Pipe\\|App Name | [Open](https://example.com/pipe) | Creator\\|Team One | iOS |"
	if !strings.Contains(readme, expectedRow) {
		t.Fatalf("expected escaped markdown row, got:\n%s", readme)
	}
}

func TestGenerateDedupesUnknownPlatformsCaseInsensitive(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Web App",
    "link": "https://example.com/web",
    "creator": "Web Creator",
    "platform": ["Web", "web", "WEB", "iOS", "IOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	expectedRow := "| Web App | [Open](https://example.com/web) | Web Creator | Web, iOS |"
	if !strings.Contains(readme, expectedRow) {
		t.Fatalf("expected deduped platform row, got:\n%s", readme)
	}
}

func TestGeneratePreservesReadmeContentOutsideMarkers(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Preserve App",
    "link": "https://example.com/preserve",
    "creator": "Preserve Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
Top section.
<!-- WALL-OF-APPS:START -->
Replace me.
<!-- WALL-OF-APPS:END -->
Bottom section.
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	if !strings.Contains(readme, "# Demo\nTop section.\n<!-- WALL-OF-APPS:START -->") {
		t.Fatalf("expected top section preserved, got:\n%s", readme)
	}
	if !strings.Contains(readme, "<!-- WALL-OF-APPS:END -->\nBottom section.\n") {
		t.Fatalf("expected bottom section preserved, got:\n%s", readme)
	}
}
