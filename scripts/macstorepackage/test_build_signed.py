"""One-shot Store packaging orchestration guards."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import build_signed


class BuildSignedTests(unittest.TestCase):
    def test_builds_before_signing_with_fresh_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            settings = {key: "fixture" for key in build_signed.REQUIRED}
            with patch.dict(os.environ, {}, clear=True), patch("build_signed.subprocess.run") as run:
                output = build_signed.build_signed(repo, settings)
            self.assertEqual([call.args[0][-1] for call in run.call_args_list],
                             ["apple-store-preflight", "apple-store-package-local", "apple-store-package-signed"])
            environment = run.call_args_list[-1].kwargs["env"]
            self.assertEqual(environment["APPLE_STORE_APP"], str(output.parent / "local/PicFetch.app"))
            self.assertEqual(environment["APPLE_STORE_SIGNED_OUTPUT_DIR"], str(output))
            self.assertEqual(environment["APPLE_STORE_TESTFLIGHT"], "1")
            self.assertFalse(output.exists())
            self.assertTrue(all(call.kwargs["check"] for call in run.call_args_list))

    def test_stops_after_failed_build(self):
        with tempfile.TemporaryDirectory() as directory:
            settings = {key: "fixture" for key in build_signed.REQUIRED}
            failure = subprocess.CalledProcessError(1, "build")
            with patch.dict(os.environ, {}, clear=True), patch("build_signed.subprocess.run", side_effect=[None, failure]) as run:
                with self.assertRaises(subprocess.CalledProcessError):
                    build_signed.build_signed(Path(directory), settings)
            self.assertEqual(run.call_count, 2)

    def test_missing_configuration_stops_before_commands(self):
        with patch.dict(os.environ, {}, clear=True), patch("build_signed.subprocess.run") as run:
            with self.assertRaisesRegex(ValueError, "APPLE_STORE"):
                build_signed.build_signed(Path.cwd(), {})
            run.assert_not_called()

    def test_environment_overrides_local_settings(self):
        with tempfile.TemporaryDirectory() as directory:
            settings = {key: "fixture" for key in build_signed.REQUIRED}
            with patch.dict(os.environ, {"APPLE_STORE_TEAM_ID": "OVERRIDE01"}, clear=True), patch("build_signed.subprocess.run") as run:
                build_signed.build_signed(Path(directory), settings)
            self.assertEqual(run.call_args.kwargs["env"]["APPLE_STORE_TEAM_ID"], "OVERRIDE01")
