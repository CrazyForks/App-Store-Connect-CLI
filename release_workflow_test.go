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
	assignAMD64 := "SHA256_AMD64=$(shasum"
	assignArm64 := "SHA256=$(shasum"
	exportChecksums := "export SHA256_AMD64 SHA256"
	pythonStep := "python3 - <<'PY'"

	assignAMD64Index := strings.Index(workflow, assignAMD64)
	if assignAMD64Index == -1 {
		t.Fatalf("release workflow missing %q", assignAMD64)
	}

	assignArm64Index := strings.Index(workflow, assignArm64)
	if assignArm64Index == -1 {
		t.Fatalf("release workflow missing %q", assignArm64)
	}

	exportIndex := strings.Index(workflow, exportChecksums)
	if exportIndex == -1 {
		t.Fatalf("release workflow missing %q", exportChecksums)
	}

	pythonIndex := strings.Index(workflow, pythonStep)
	if pythonIndex == -1 {
		t.Fatalf("release workflow missing %q", pythonStep)
	}

	if assignAMD64Index > exportIndex || exportIndex > pythonIndex {
		t.Fatalf("%q must be assigned and exported before %q", assignAMD64, pythonStep)
	}
	if assignArm64Index > exportIndex || exportIndex > pythonIndex {
		t.Fatalf("%q must be assigned and exported before %q", assignArm64, pythonStep)
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
	releaseStart := strings.Index(workflow, "\n  build:\n")
	if releaseStart == -1 {
		t.Fatal("release workflow missing build job")
	}
	releaseJob := workflow[releaseStart:]
	for _, want := range []string{
		`ASC_KEY_ID: ${{ secrets.ASC_KEY_ID }}`,
		`ASC_ISSUER_ID: ${{ secrets.ASC_ISSUER_ID }}`,
		`ASC_PRIVATE_KEY_B64: ${{ secrets.ASC_PRIVATE_KEY_B64 }}`,
		`ASC_PRIVATE_KEY_PATH: ${{ runner.temp }}/AuthKey.p8`,
		`ASC_BYPASS_KEYCHAIN: "1"`,
		`notarization list --limit 1 --output json`,
		`notarization submit`,
		`--wait`,
		`--timeout 1h`,
		`notarization log --id`,
		`codesign -vvvv -R="notarized" --check-notarization`,
	} {
		if !strings.Contains(releaseJob, want) {
			t.Errorf("release workflow missing notarization contract %q", want)
		}
	}
	if strings.Contains(releaseJob, `auth status --validate`) {
		t.Fatal("release workflow must validate the environment credentials with a Notary API request")
	}
	if strings.Contains(releaseJob, `spctl --assess`) {
		t.Fatal("release workflow must not assess standalone binaries with spctl")
	}

	notarizeIndex := strings.Index(workflow, "- name: Notarize macOS binaries")
	checksumIndex := strings.Index(workflow, "- name: Create checksums")
	publishIndex := strings.Index(workflow, "- name: Create or verify GitHub Release")
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
	end := strings.Index(workflow, "\n  build:\n")
	if start == -1 || end == -1 || start >= end {
		t.Fatal("release workflow must define repair-notarization before release")
	}
	repairJob := workflow[start:end]

	for _, want := range []string{
		`persist-credentials: false`,
		`gh release download "${VERSION}"`,
		`shasum -a 256 -c`,
		`codesign --verify --deep --strict --verbose=2`,
		`is signed by an unexpected Developer ID`,
		`ASC_KEY_ID: ${{ secrets.ASC_KEY_ID }}`,
		`ASC_ISSUER_ID: ${{ secrets.ASC_ISSUER_ID }}`,
		`ASC_PRIVATE_KEY_B64: ${{ secrets.ASC_PRIVATE_KEY_B64 }}`,
		`ASC_PRIVATE_KEY_PATH: ${{ runner.temp }}/AuthKey.p8`,
		`ASC_BYPASS_KEYCHAIN: "1"`,
		`notarization list --limit 1 --output json`,
		`notarization submit`,
		`codesign -vvvv -R="notarized" --check-notarization`,
	} {
		if !strings.Contains(repairJob, want) {
			t.Errorf("repair-notarization job missing %q", want)
		}
	}
	for _, unwanted := range []string{
		`auth status --validate`,
		`spctl --assess`,
	} {
		if strings.Contains(repairJob, unwanted) {
			t.Errorf("repair-notarization job contains invalid verification %q", unwanted)
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

func TestReleaseWorkflowCreatesPrivateKeyWithRestrictedModeInRunnerTemp(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	if got := strings.Count(workflow, `install -m 600 /dev/null "$RUNNER_TEMP/AuthKey.p8"`); got != 2 {
		t.Fatalf("release workflow must securely create both temporary keys, got %d sites", got)
	}
	cleanup := "- name: Remove notarization credentials\n        if: always()\n        run: rm -f \"$RUNNER_TEMP/AuthKey.p8\""
	if got := strings.Count(workflow, cleanup); got != 2 {
		t.Fatalf("release workflow must unconditionally clean up both temporary keys, got %d sites", got)
	}
	if strings.Contains(workflow, "/tmp/AuthKey.p8") {
		t.Fatal("release workflow must not store private keys in shared /tmp")
	}
}

func TestReleaseWorkflowReusesOneBuildArtifactForEveryPublisher(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	for _, want := range []string{
		"\n  build:\n",
		"\n  publish:\n",
		"\n  homebrew:\n",
		"\n  winget:\n",
		"outputs:\n      version: ${{ steps.version.outputs.version }}",
		"actions/upload-artifact@",
		"actions/download-artifact@",
		"name: release-${{ needs.build.outputs.version }}",
		`tar -cf "workflow-artifact/release-${VERSION}.tar" release`,
		`tar -xf "workflow-artifact/release-${VERSION}.tar"`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing split-job contract %q", want)
		}
	}

	if got := strings.Count(workflow, "name: release-${{ steps.version.outputs.version }}"); got != 1 {
		t.Fatalf("release workflow must upload exactly one immutable release artifact, got %d", got)
	}
	if got := strings.Count(workflow, "actions/download-artifact@"); got != 3 {
		t.Fatalf("each publisher must download the same build artifact, got %d downloads", got)
	}
}

func TestReleaseWorkflowNeverClobbersPublishedAssets(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	for _, unwanted := range []string{"gh release upload", "--clobber"} {
		if strings.Contains(workflow, unwanted) {
			t.Errorf("release workflow must not replace published assets; found %q", unwanted)
		}
	}
	for _, want := range []string{
		`gh release download "${VERSION}"`,
		`cmp -s "$asset" "$published/$name"`,
		`Published release assets differ from this run`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing published-asset verification %q", want)
		}
	}
}

func TestReleaseWorkflowUsesCommitTimestampForBuildMetadata(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	if !strings.Contains(workflow, `DATE=$(git show -s --format=%cI HEAD)`) {
		t.Fatal("release workflow must derive build metadata from the release commit")
	}
	if strings.Contains(workflow, `DATE=$(date -u`) {
		t.Fatal("release workflow must not embed the wall-clock build time")
	}
}

func TestReleaseWorkflowPushesWinGetBranchWithoutHistoryRewrite(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	for _, unwanted := range []string{"--force-with-lease", "git push --force"} {
		if strings.Contains(workflow, unwanted) {
			t.Errorf("WinGet publication must not rewrite branch history; found %q", unwanted)
		}
	}
	for _, want := range []string{
		`git merge-base --is-ancestor origin/master upstream/master`,
		`git checkout -b "${BRANCH}" origin/master`,
		`git push --set-upstream origin "${BRANCH}"`,
		`WinGet branch contains changes outside the ${VERSION} manifest directory`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing fast-forward-only WinGet contract %q", want)
		}
	}
}
