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
    assert "runs-on: ubuntu-latest" in job_block(workflow, "wall-only-check")
    assert "runs-on: ubuntu-latest" in job_block(workflow, "format-and-lint")
    assert "runs-on: ubuntu-latest" in job_block(workflow, test_job)

    build_platforms = job_block(workflow, "build-platforms")
    for runner in ("macos-latest", "ubuntu-latest", "windows-latest"):
        assert f"runner: {runner}" in build_platforms, f"{path}: missing native build runner {runner}"
    assert "go test -short ./internal/screenshots" in build_platforms, f"{path}: missing Darwin-only tests"
    assert "asc_dev_macos_amd64" in build_platforms
    assert "asc_dev_macos_arm64" in build_platforms

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
