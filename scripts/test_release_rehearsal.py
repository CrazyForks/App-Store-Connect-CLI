#!/usr/bin/env python3

import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("release_rehearsal.py")
SPEC = importlib.util.spec_from_file_location("release_rehearsal", MODULE_PATH)
release_rehearsal = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release_rehearsal)


class ReleaseRehearsalTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        self.git("init")
        self.git("config", "user.name", "Test User")
        self.git("config", "user.email", "test@example.com")
        self.commit("initial release")
        self.git("tag", "1.2.3")

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    def git(self, *args: str) -> str:
        result = subprocess.run(
            ["git", *args],
            cwd=self.root,
            check=True,
            capture_output=True,
            text=True,
        )
        return result.stdout.strip()

    def commit(self, message: str) -> None:
        marker = self.root / "history.txt"
        previous = marker.read_text() if marker.exists() else ""
        marker.write_text(f"{previous}{message}\n")
        self.git("add", "history.txt")
        self.git("commit", "-m", message)

    def create_artifacts(self, version: str) -> Path:
        release_dir = self.root / "release"
        release_dir.mkdir()
        for name in release_rehearsal.expected_artifact_names(version):
            (release_dir / name).write_bytes(f"binary:{name}".encode())
        return release_dir

    def test_generates_notes_and_checksums_for_exact_commit(self) -> None:
        self.commit("add upload reconciliation")
        release_dir = self.create_artifacts("1.2.4")
        head = self.git("rev-parse", "HEAD")

        result = release_rehearsal.rehearse(
            root=self.root,
            version="1.2.4",
            expected_sha=head,
            release_dir=release_dir,
        )

        self.assertEqual(result.tested_sha, head)
        self.assertEqual(result.previous_tag, "1.2.3")
        notes = result.notes_path.read_text()
        self.assertIn("# Release 1.2.4", notes)
        self.assertIn(f"Tested commit: `{head}`", notes)
        self.assertIn("- add upload reconciliation", notes)
        checksums = result.checksums_path.read_text().splitlines()
        self.assertEqual(len(checksums), 5)
        self.assertTrue(all("  asc_1.2.4_" in line for line in checksums))

    def test_rejects_invalid_version(self) -> None:
        with self.assertRaisesRegex(release_rehearsal.RehearsalError, "x.y.z"):
            release_rehearsal.rehearse(
                root=self.root,
                version="v1.2.4",
                expected_sha=self.git("rev-parse", "HEAD"),
                release_dir=self.root / "release",
            )

    def test_rejects_mismatched_commit(self) -> None:
        previous = self.git("rev-parse", "HEAD")
        self.commit("candidate change")
        self.create_artifacts("1.2.4")

        with self.assertRaisesRegex(release_rehearsal.RehearsalError, "does not match"):
            release_rehearsal.rehearse(
                root=self.root,
                version="1.2.4",
                expected_sha=previous,
                release_dir=self.root / "release",
            )

    def test_rejects_empty_changelog(self) -> None:
        self.create_artifacts("1.2.4")

        with self.assertRaisesRegex(release_rehearsal.RehearsalError, "no commits"):
            release_rehearsal.rehearse(
                root=self.root,
                version="1.2.4",
                expected_sha=self.git("rev-parse", "HEAD"),
                release_dir=self.root / "release",
            )

    def test_rejects_missing_artifact(self) -> None:
        self.commit("candidate change")
        release_dir = self.create_artifacts("1.2.4")
        (release_dir / "asc_1.2.4_linux_arm64").unlink()

        with self.assertRaisesRegex(release_rehearsal.RehearsalError, "missing release artifact"):
            release_rehearsal.rehearse(
                root=self.root,
                version="1.2.4",
                expected_sha=self.git("rev-parse", "HEAD"),
                release_dir=release_dir,
            )


if __name__ == "__main__":
    unittest.main()
