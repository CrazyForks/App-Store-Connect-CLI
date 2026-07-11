#!/usr/bin/env python3
"""Protect CI runner, build, and artifact ownership contracts."""

from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
PR_WORKFLOW = ROOT / ".github/workflows/pr-checks.yml"
MAIN_WORKFLOW = ROOT / ".github/workflows/main-branch.yml"
RELEASE_WORKFLOW = ROOT / ".github/workflows/release.yml"


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
    assert "runs-on: ubuntu-latest" in job_block(workflow, "wall-only-check")
    assert "runs-on: ubuntu-latest" in job_block(workflow, "format-and-lint")
    assert "runs-on: ubuntu-latest" in job_block(workflow, test_job)

    build_platforms = job_block(workflow, "build-platforms")
    for runner in ("macos-latest", "ubuntu-latest", "windows-latest"):
        assert f"runner: {runner}" in build_platforms, f"{path}: missing native build runner {runner}"
    assert "go test -short ./internal/screenshots" in build_platforms, f"{path}: missing Darwin-only tests"

    build = job_block(workflow, "build")
    assert "needs: [changes, build-platforms]" in build
    assert "needs.build-platforms.result" in build


def main() -> None:
    assert_optimized_workflow(PR_WORKFLOW, "unit-test-shards")
    assert_optimized_workflow(MAIN_WORKFLOW, "test-shards")

    release = RELEASE_WORKFLOW.read_text()
    assert "actions/upload-artifact" in release, "release workflow must retain official artifact publication"

    print("CI workflow contracts passed")


if __name__ == "__main__":
    main()
