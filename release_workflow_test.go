package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseWorkflowExportsHomebrewChecksumsBeforeFormulaGeneration(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	exportAMD64 := "export SHA256_AMD64="
	exportArm64 := "export SHA256="
	pythonStep := "python3 - <<'PY'"

	exportAMD64Index := strings.Index(workflow, exportAMD64)
	if exportAMD64Index == -1 {
		t.Fatalf("release workflow missing %q", exportAMD64)
	}

	exportArm64Index := strings.Index(workflow, exportArm64)
	if exportArm64Index == -1 {
		t.Fatalf("release workflow missing %q", exportArm64)
	}

	pythonIndex := strings.Index(workflow, pythonStep)
	if pythonIndex == -1 {
		t.Fatalf("release workflow missing %q", pythonStep)
	}

	if exportAMD64Index > pythonIndex {
		t.Fatalf("%q must appear before %q", exportAMD64, pythonStep)
	}
	if exportArm64Index > pythonIndex {
		t.Fatalf("%q must appear before %q", exportArm64, pythonStep)
	}
}

func TestReleaseWorkflowPreservesRubyBinInterpolationInFormulaTest(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	want := `shell_output("#{{bin}}/asc --help")`
	if !strings.Contains(workflow, want) {
		t.Fatalf("release workflow missing escaped Ruby interpolation %q", want)
	}

	unwanted := `shell_output("#{bin}/asc --help")`
	if strings.Contains(workflow, unwanted) {
		t.Fatalf("release workflow still contains unescaped Ruby interpolation %q", unwanted)
	}
}

func TestReleaseWorkflowKeepsHistoricalGuardrailsInline(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	for _, want := range []string{
		`python3 scripts/test_check_docs.py`,
		`make format-check`,
		`make check-docs`,
		`make check-wall-of-apps`,
		`make lint`,
		`ASC_BYPASS_KEYCHAIN=1 make test`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing historical guardrail %q", want)
		}
	}
	if strings.Contains(workflow, "make release-guardrails") {
		t.Fatal("release workflow cannot call a target absent from historical tags")
	}
}

func TestReleaseRehearsalGuardrailsIncludeDocsValidatorSelfTest(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	makefile := string(data)
	start := strings.Index(makefile, "release-guardrails:\n")
	if start == -1 {
		t.Fatal("Makefile missing release-guardrails target")
	}
	end := strings.Index(makefile[start:], "\n# Show help")
	if end == -1 {
		t.Fatal("could not find end of release-guardrails target")
	}
	guardrails := makefile[start : start+end]
	if !strings.Contains(guardrails, "python3 scripts/test_check_docs.py") {
		t.Fatal("release-guardrails must run the docs-validator self-test")
	}
}

func TestReleaseWorkflowBuildsStrippedTrimmedBinaries(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	buildLines := 0
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "go build") {
			continue
		}
		buildLines++
		if !strings.Contains(line, "-trimpath") {
			t.Errorf("release build line missing -trimpath: %s", strings.TrimSpace(line))
		}
		if !strings.Contains(line, "-s -w") {
			t.Errorf("release build line missing -s -w: %s", strings.TrimSpace(line))
		}
	}
	if buildLines == 0 {
		t.Fatal("release workflow contains no go build lines")
	}
}

func TestReleaseWorkflowDoesNotInterpolateDispatchInputIntoShell(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	if strings.Contains(workflow, `TAG="${{ github.event_name == 'workflow_dispatch'`) {
		t.Fatal("release workflow interpolates dispatch input directly into shell")
	}
	for _, want := range []string{
		`RELEASE_TAG: ${{ github.event_name == 'workflow_dispatch' && inputs.version || github.ref_name }}`,
		`TAG="${RELEASE_TAG}"`,
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing safe dispatch handling %q", want)
		}
	}
}

func TestReleaseWorkflowNotarizesMacOSBinariesBeforePublishing(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	for _, want := range []string{
		`ASC_KEY_ID: ${{ secrets.ASC_KEY_ID }}`,
		`ASC_ISSUER_ID: ${{ secrets.ASC_ISSUER_ID }}`,
		`ASC_PRIVATE_KEY_B64: ${{ secrets.ASC_PRIVATE_KEY_B64 }}`,
		`ASC_BYPASS_KEYCHAIN: "1"`,
		`auth status --validate --output json`,
		`notarization submit`,
		`--wait`,
		`--timeout 1h`,
		`notarization log --id`,
		`spctl --assess --type execute`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing notarization contract %q", want)
		}
	}

	notarizeIndex := strings.Index(workflow, "- name: Notarize macOS binaries")
	checksumIndex := strings.Index(workflow, "- name: Create checksums")
	publishIndex := strings.Index(workflow, "- name: Create or update GitHub Release")
	if notarizeIndex == -1 || checksumIndex == -1 || publishIndex == -1 {
		t.Fatalf("release workflow must contain notarization, checksum, and publish steps")
	}
	if notarizeIndex >= checksumIndex || checksumIndex >= publishIndex {
		t.Fatalf("release workflow must notarize before checksums and publish: notarize=%d checksum=%d publish=%d", notarizeIndex, checksumIndex, publishIndex)
	}
}

func TestReleaseWorkflowCanRepairExistingNotarizationWithoutReplacingAssets(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	if !strings.Contains(workflow, "notarize_existing:") {
		t.Fatal("release workflow missing notarize_existing dispatch input")
	}

	start := strings.Index(workflow, "\n  repair-notarization:\n")
	end := strings.Index(workflow, "\n  release:\n")
	if start == -1 || end == -1 || start >= end {
		t.Fatal("release workflow must define repair-notarization before release")
	}
	repairJob := workflow[start:end]

	for _, want := range []string{
		`gh release download "${VERSION}"`,
		`shasum -a 256 -c`,
		`codesign --verify --deep --strict --verbose=2`,
		`is signed by an unexpected Developer ID`,
		`notarization submit`,
		`spctl --assess --type execute`,
	} {
		if !strings.Contains(repairJob, want) {
			t.Errorf("repair-notarization job missing %q", want)
		}
	}

	for _, unwanted := range []string{
		`gh release upload`,
		`gh release create`,
		`codesign --force`,
		`--clobber`,
	} {
		if strings.Contains(repairJob, unwanted) {
			t.Errorf("repair-notarization job must not replace release assets; found %q", unwanted)
		}
	}
}
