#!/usr/bin/env python3
"""Tests for conservative CI change-scope classification."""

import unittest

import ci_change_scope


class ChangeScopeTest(unittest.TestCase):
    def test_empty_change_list_requires_full_suite(self) -> None:
        self.assertEqual(ci_change_scope.classify([]), "full")

    def test_wall_source_is_the_only_wall_only_change(self) -> None:
        self.assertEqual(ci_change_scope.classify(["docs/wall-of-apps.json"]), "wall")
        self.assertEqual(
            ci_change_scope.classify(["docs/wall-of-apps.json", "README.md"]),
            "full",
        )

    def test_repository_documentation_uses_docs_scope(self) -> None:
        self.assertEqual(
            ci_change_scope.classify(["README.md", "docs/TESTING.md"]),
            "docs",
        )

    def test_go_source_under_documentation_requires_full_suite(self) -> None:
        for path in ("docs/embed.go", "guides/example.go"):
            with self.subTest(path=path):
                self.assertEqual(ci_change_scope.classify([path]), "full")

    def test_openapi_snapshot_requires_schema_drift_tests(self) -> None:
        self.assertEqual(
            ci_change_scope.classify(["docs/openapi/latest.json"]),
            "full",
        )

    def test_mintlify_content_uses_website_scope(self) -> None:
        self.assertEqual(
            ci_change_scope.classify(["docs.json", "guides/getting-started.mdx"]),
            "website",
        )

    def test_telemetry_scope_requires_only_telemetry_files(self) -> None:
        self.assertEqual(
            ci_change_scope.classify(["internal/telemetry/client.go"]),
            "telemetry",
        )
        self.assertEqual(
            ci_change_scope.classify(
                ["internal/telemetry/client.go", "commands/telemetry.mdx"]
            ),
            "full",
        )

    def test_studio_scope_requires_only_studio_files(self) -> None:
        self.assertEqual(ci_change_scope.classify(["apps/studio/main.go"]), "studio")
        self.assertEqual(
            ci_change_scope.classify(
                ["apps/studio/main.go", "guides/studio.mdx"]
            ),
            "full",
        )

    def test_mixed_specialized_changes_require_full_suite(self) -> None:
        self.assertEqual(
            ci_change_scope.classify(
                ["apps/studio/main.go", "internal/telemetry/client.go"]
            ),
            "full",
        )

    def test_general_go_and_ci_changes_require_full_suite(self) -> None:
        for path in ("main.go", "internal/asc/client.go", ".github/workflows/pr-checks.yml"):
            with self.subTest(path=path):
                self.assertEqual(ci_change_scope.classify([path]), "full")

    def test_dedicated_workflow_impact_matches_owned_paths(self) -> None:
        self.assertTrue(ci_change_scope.affects_website(["guides/testflight.mdx"]))
        self.assertFalse(ci_change_scope.affects_website(["docs/TESTING.md"]))
        self.assertTrue(ci_change_scope.affects_studio(["apps/studio/main.go"]))
        self.assertTrue(ci_change_scope.affects_studio(["internal/asc/client.go"]))
        self.assertTrue(ci_change_scope.affects_studio(["go.mod"]))
        self.assertFalse(ci_change_scope.affects_studio(["internal/telemetry/client.go"]))


if __name__ == "__main__":
    unittest.main()
