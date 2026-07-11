#!/usr/bin/env python3
"""Protect CI runner, build, and artifact ownership contracts."""

from pathlib import Path

import ci_change_scope


ROOT = Path(__file__).resolve().parent.parent
PR_WORKFLOW = ROOT / ".github/workflows/pr-checks.yml"
MAIN_WORKFLOW = ROOT / ".github/workflows/main-branch.yml"
RELEASE_WORKFLOW = ROOT / ".github/workflows/release.yml"
STUDIO_WORKFLOW = ROOT / ".github/workflows/studio-checks.yml"
WEBSITE_WORKFLOW = ROOT / ".github/workflows/website-checks.yml"


def job_block(workflow: str, job: str) -> str:
    marker = f"  {job}:\n"
    start = workflow.find(marker)
    if start < 0:
        raise AssertionError(f"missing job {job!r}")

    end = len(workflow)
    for line_start in range(start + len(marker), len(workflow)):
        if line_start > 0 and workflow[line_start - 1] != "\n":
            continue
        line_end = workflow.find("\n", line_start)
        if line_end < 0:
            line_end = len(workflow)
        line = workflow[line_start:line_end]
        if line.startswith("  ") and not line.startswith("    ") and line.endswith(":"):
            end = line_start
            break
    return workflow[start:end]


def assert_optimized_workflow(path: Path, test_job: str) -> None:
    workflow = path.read_text()

    assert "actions/upload-artifact" not in workflow, f"{path}: development artifacts must not be uploaded"
    changes = job_block(workflow, "changes")
    assert "scope: ${{ steps.scope.outputs.scope }}" in changes
    assert "python3 scripts/ci_change_scope.py" in changes
    assert "wall|docs|website|studio|telemetry|full" in changes
    assert "invalid CI scope" in changes

    assert "runs-on: ubuntu-latest" in job_block(workflow, "wall-only-check")
    assert "runs-on: ubuntu-latest" in job_block(workflow, "format-and-lint")
    tests = job_block(workflow, test_job)
    assert "runs-on: ubuntu-latest" in tests
    assert "needs.changes.outputs.scope == 'full'" in tests

    build_platforms = job_block(workflow, "build-platforms")
    assert "needs.changes.outputs.scope == 'full'" in build_platforms
    for runner in ("macos-latest", "ubuntu-latest", "windows-latest"):
        assert f"runner: {runner}" in build_platforms, f"{path}: missing native build runner {runner}"

    ordinary_build = job_block(workflow, "ordinary-build")
    assert "needs.changes.outputs.scope == 'telemetry'" in ordinary_build
    assert "go build ./..." in ordinary_build

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

    website = WEBSITE_WORKFLOW.read_text()
    assert "runs-on: ubuntu-latest" in job_block(website, "website")
    assert "make check-website-docs" in website
    for path in ci_change_scope.WEBSITE_FILES:
        assert f"'{path}'" in website, f"website workflow does not own {path}"
    for prefix in ci_change_scope.WEBSITE_PREFIXES:
        assert f"'{prefix}**'" in website, f"website workflow does not own {prefix}"
    assert "'*.mdx'" in website

    studio = STUDIO_WORKFLOW.read_text()
    assert "runs-on: ubuntu-latest" in job_block(studio, "studio")
    assert f"'{ci_change_scope.STUDIO_PREFIX}**'" in studio

    release = RELEASE_WORKFLOW.read_text()
    assert "actions/upload-artifact" in release, "release workflow must retain official artifact publication"

    print("CI workflow contracts passed")


if __name__ == "__main__":
    main()
