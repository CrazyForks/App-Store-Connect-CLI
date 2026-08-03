#!/usr/bin/env python3
"""Protect CI runner, build, and artifact ownership contracts."""

import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
PR_WORKFLOW = ROOT / ".github/workflows/pr-checks.yml"
MAIN_WORKFLOW = ROOT / ".github/workflows/main-branch.yml"
RELEASE_WORKFLOW = ROOT / ".github/workflows/release.yml"
RELEASE_REHEARSAL_WORKFLOW = ROOT / ".github/workflows/release-rehearsal.yml"
WEBSITE_WORKFLOW = ROOT / ".github/workflows/website-checks.yml"


def job_block(workflow: str, job: str) -> str:
    marker = f"  {job}:\n"
    start = workflow.find(marker)
    if start < 0:
        raise AssertionError(f"missing job {job!r}")

    end = len(workflow)
    offset = start + len(marker)
    for line in workflow[offset:].splitlines(keepends=True):
        content = line.rstrip("\r\n")
        if content.startswith("  ") and not content.startswith("    ") and content.endswith(":"):
            end = offset
            break
        offset += len(line)
    return workflow[start:end]


def assert_optimized_workflow(path: Path, test_job: str) -> None:
    workflow = path.read_text()

    assert "actions/upload-artifact" not in workflow, f"{path}: development artifacts must not be uploaded"
    changes = job_block(workflow, "changes")
    assert "scope: ${{ steps.scope.outputs.scope }}" in changes
    assert "website_affected: ${{ steps.scope.outputs.website_affected }}" in changes
    assert "python3 scripts/ci_change_scope.py --github-output" in changes
    assert "git diff --name-only --no-renames" in changes
    for guarded_path in (
        ".github/workflows/*",
        "scripts/ci_change_scope.py",
        "scripts/test_ci_change_scope.py",
        "scripts/test_ci_workflows.py",
    ):
        assert guarded_path in changes
    assert changes.index("force_full=false") < changes.index(
        "python3 scripts/ci_change_scope.py --github-output"
    )
    assert 'if [ "$force_full" = true ]; then' in changes
    assert "website_affected=true" in changes
    assert "wall|docs|website|telemetry|full" in changes
    assert "invalid CI scope" in changes

    assert "runs-on: ubuntu-latest" in job_block(workflow, "wall-only-check")
    assert "runs-on: ubuntu-latest" in job_block(workflow, "format-and-lint")
    quality = job_block(workflow, "quality-checks")
    assert "runs-on: ubuntu-latest" in quality
    assert "python3 scripts/test_ci_change_scope.py" in quality
    assert "contains(fromJSON('[\"telemetry\", \"full\"]'), needs.changes.outputs.scope)" in quality
    website = job_block(workflow, "website-checks")
    assert "uses: ./.github/workflows/website-checks.yml" in website
    assert "needs.changes.outputs.website_affected == 'true'" in website
    tests = job_block(workflow, test_job)
    assert "runs-on: ubuntu-latest" in tests
    assert "needs.changes.outputs.scope == 'full'" in tests

    build_platforms = job_block(workflow, "build-platforms")
    assert "needs.changes.outputs.scope == 'full'" in build_platforms
    for runner in ("macos-latest", "ubuntu-latest", "windows-latest"):
        assert f"runner: {runner}" in build_platforms, f"{path}: missing native build runner {runner}"
    assert "go test -short ./internal/screenshots" in build_platforms, f"{path}: missing Darwin-only tests"
    assert "asc_dev_macos_amd64" in build_platforms
    assert "asc_dev_macos_arm64" in build_platforms

    ordinary_build = job_block(workflow, "ordinary-build")
    assert "needs.changes.outputs.scope == 'telemetry'" in ordinary_build
    assert "runner: ubuntu-latest" in ordinary_build
    assert "runner: macos-latest" in ordinary_build
    assert "run: go build ." in ordinary_build
    assert "go build ./..." not in ordinary_build

    build = job_block(workflow, "build")
    assert "needs: [changes, build-platforms, ordinary-build]" in build
    assert "if: always()" in build
    assert "needs.build-platforms.result" in build
    assert "needs.ordinary-build.result" in build


def main() -> None:
    assert_optimized_workflow(PR_WORKFLOW, "unit-test-shards")
    assert_optimized_workflow(MAIN_WORKFLOW, "test-shards")

    pr = PR_WORKFLOW.read_text()
    for required_job in ("format-and-lint", "unit-tests", "build"):
        assert "if: always()" in job_block(pr, required_job), f"required job {required_job} must always resolve"
    quality_gate = job_block(pr, "format-and-lint")
    assert "needs: [changes, wall-only-check, quality-checks, website-checks]" in quality_gate
    assert "needs.website-checks.result" in quality_gate

    website = WEBSITE_WORKFLOW.read_text()
    assert "workflow_call:" in website
    assert "runs-on: ubuntu-latest" in job_block(website, "website")
    assert "make check-website-docs" in website

    main = MAIN_WORKFLOW.read_text()
    assert "git diff-tree --no-commit-id --name-only --no-renames -r" in main
    main_windows = job_block(main, "telemetry-windows-tests")
    assert "[\"telemetry\", \"full\"]" in main_windows
    assert "needs.telemetry-windows-tests.result" in job_block(main, "test")

    release = RELEASE_WORKFLOW.read_text()
    assert "actions/upload-artifact" in release, "release workflow must retain official artifact publication"

    rehearsal = RELEASE_REHEARSAL_WORKFLOW.read_text()
    assert "workflow_dispatch:" in rehearsal
    assert "version:" in rehearsal
    assert "ref:" in rehearsal
    assert "contents: read" in rehearsal
    assert "GIT_TERMINAL_PROMPT: 0" in rehearsal
    assert "GH_PROMPT_DISABLED: 1" in rehearsal
    assert "ref: ${{ inputs.ref }}" in rehearsal
    assert "fetch-depth: 0" in rehearsal
    assert "persist-credentials: false" in rehearsal
    assert "go-version-file: go.mod" in rehearsal
    assert "go-version:" not in rehearsal
    assert 'TESTED_SHA="$(git rev-parse HEAD)"' in rehearsal
    assert 'make release-rehearsal VERSION="${VERSION}" EXPECTED_SHA="${TESTED_SHA}"' in rehearsal
    assert "GITHUB_STEP_SUMMARY" in rehearsal
    for forbidden in (
        "contents: write",
        "secrets.",
        "apple-actions/import-codesign-certs",
        "actions/upload-artifact",
        "gh release create",
        "gh release upload",
        "git push",
        "git tag",
    ):
        assert forbidden not in rehearsal, f"release rehearsal must not contain {forbidden!r}"

    subprocess.run([sys.executable, str(ROOT / "scripts/test_release_rehearsal.py")], check=True)

    print("CI workflow contracts passed")


if __name__ == "__main__":
    main()
