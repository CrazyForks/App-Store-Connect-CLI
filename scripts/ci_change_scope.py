#!/usr/bin/env python3
"""Classify changed paths for conservative CI test selection."""

from __future__ import annotations

import sys
from collections.abc import Iterable


WALL_SOURCE = "docs/wall-of-apps.json"
WEBSITE_FILES = {".mintignore", "docs.json"}
WEBSITE_PREFIXES = (
    ".mintlify/",
    "commands/",
    "concepts/",
    "configuration/",
    "guides/",
    "resources/",
)
TELEMETRY_PREFIXES = ("internal/telemetry/", "internal/cli/telemetry/")
STUDIO_PREFIX = "apps/studio/"


def path_kind(path: str) -> str:
    if path == WALL_SOURCE:
        return "wall"
    if path in WEBSITE_FILES or path.startswith(WEBSITE_PREFIXES):
        return "website"
    if "/" not in path and path.endswith(".mdx"):
        return "website"
    if path.startswith(TELEMETRY_PREFIXES):
        return "telemetry"
    if path.startswith(STUDIO_PREFIX):
        return "studio"
    if path.startswith("docs/"):
        return "docs"
    if "/" not in path and path.endswith(".md"):
        return "docs"
    return "full"


def classify(paths: Iterable[str]) -> str:
    normalized = [path.strip() for path in paths if path.strip()]
    if not normalized:
        return "full"
    if normalized == [WALL_SOURCE]:
        return "wall"

    kinds = {path_kind(path) for path in normalized}
    if "wall" in kinds or "full" in kinds:
        return "full"

    if kinds == {"telemetry"}:
        return "telemetry"
    if kinds == {"studio"}:
        return "studio"
    if kinds == {"website"}:
        return "website"
    if kinds <= {"docs", "website"}:
        return "docs"
    return "full"


def main() -> None:
    print(classify(sys.stdin))


if __name__ == "__main__":
    main()
