"""Portable signing-policy guards; native installer tests are opt-in."""
from argparse import Namespace
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import shutil
import ssl
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import distribution
from package import APP_ID, run


class DistributionPolicy(unittest.TestCase):
    def setUp(self):
        self.now = datetime(2026, 9, 30, tzinfo=timezone.utc)
        self.team = "ABCDEFGHIJ"
        self.certificate = b"synthetic certificate"
        self.fingerprint = hashlib.sha1(self.certificate).hexdigest().upper()
        self.profile = {
            "TeamIdentifier": [self.team], "ApplicationIdentifierPrefix": ["LEGACYPFX1"],
            "Platform": ["OSX"], "CreationDate": self.now - timedelta(days=1),
            "ExpirationDate": self.now + timedelta(days=30),
            "DeveloperCertificates": [self.certificate],
            "Entitlements": {"com.apple.application-identifier": "LEGACYPFX1." + APP_ID,
                             "com.apple.developer.team-identifier": self.team},
        }

    def check(self, profile):
        return distribution.profile_entitlements(profile, self.team, APP_ID,
                                                  self.fingerprint, "app", self.now)

    def test_rejects_wrong_profile_authority(self):
        changes = [
            {"TeamIdentifier": ["OTHERTEAM1"]}, {"Platform": ["iOS"]},
            {"ExpirationDate": self.now}, {"CreationDate": self.now + timedelta(days=1)},
            {"DeveloperCertificates": [b"other"]}, {"ProvisionedDevices": []},
            {"ProvisionsAllDevices": True}, {"ApplicationIdentifierPrefix": ["WRONGPREFX"]},
            {"Entitlements": {"com.apple.application-identifier": "LEGACYPFX1.*"}},
            {"Entitlements": self.profile["Entitlements"] | {"com.apple.security.get-task-allow": True}},
            {"Entitlements": self.profile["Entitlements"] | {"com.apple.developer.team-identifier": "OTHERTEAM1"}},
        ]
        for change in changes:
            with self.subTest(change=change), self.assertRaises(ValueError):
                self.check(self.profile | change)

    def test_exact_claims_preserve_legacy_prefix(self):
        claims = self.check(self.profile)
        self.assertEqual(claims, distribution.ENTITLEMENTS["app"] | self.profile["Entitlements"])
        self.assertNotIn("get-task-allow", claims)
        self.assertNotIn("com.apple.security.network.server", claims)
        self.assertEqual(self.profile["Entitlements"]["com.apple.application-identifier"], "LEGACYPFX1." + APP_ID)

    def test_profile_does_not_broaden_capabilities_or_apply_to_helpers(self):
        profile = self.profile | {"Entitlements": self.profile["Entitlements"] | {
            "beta-reports-active": True, "com.apple.security.network.server": True}}
        claims = self.check(profile)
        self.assertIs(claims["beta-reports-active"], True)
        self.assertNotIn("com.apple.security.network.server", claims)
        with self.assertRaises(ValueError):
            distribution.profile_entitlements(self.profile, self.team, APP_ID,
                                              self.fingerprint, "helper", self.now)
        worker = dict(self.profile)
        worker["Entitlements"] = self.profile["Entitlements"] | {"com.apple.application-identifier": "LEGACYPFX1." + APP_ID + ".worker"}
        claims = distribution.profile_entitlements(worker, self.team, APP_ID + ".worker",
                                                   self.fingerprint, "service", self.now)
        self.assertNotIn("com.apple.security.network.client", claims)
        self.assertEqual(claims["com.apple.application-identifier"], "LEGACYPFX1." + APP_ID + ".worker")

    def test_identity_requires_role_team_and_unique_fingerprint(self):
        name = "Apple Distribution: Fixture (ABCDEFGHIJ)"
        output = f'  1) {self.fingerprint} "{name}"\n  1 valid identities found\n'
        self.assertEqual(distribution.select_identity(output, self.fingerprint, self.team, "app"), name)
        for text, fingerprint, team, role in [
            (output, self.fingerprint, self.team, "installer"),
            (output, self.fingerprint, "OTHERTEAM1", "app"),
            (output, "-", self.team, "app"),
            (output, "0" * 40, self.team, "app"),
            (output + output, self.fingerprint, self.team, "app"),
            (output.replace("Apple Distribution:", "Developer ID Application:"), self.fingerprint, self.team, "app"),
        ]:
            with self.subTest(text=text, role=role), self.assertRaises(ValueError):
                distribution.select_identity(text, fingerprint, team, role)


class InstallerPolicy(unittest.TestCase):
    def test_installer_report_requires_selected_trusted_leaf(self):
        name = "3rd Party Mac Developer Installer: Fixture (ABCDEFGHIJ)"
        report = f"Package: PicFetch.pkg\n Status: signed by a certificate trusted by macOS\n Certificate Chain:\n  1. {name}\n    SHA256 Fingerprint: {'12' * 32}\n  2. Apple Root CA\n"
        distribution.verify_installer_report(report, name, "12" * 32)
        for changed in (report.replace(name, "Developer ID Installer: Fixture (ABCDEFGHIJ)"),
                        report.replace("12" * 32, "34" * 32),
                        report.replace("trusted by macOS", "not trusted by macOS"),
                        report.replace("signed by a certificate trusted by macOS", "no signature")):
            with self.assertRaises(ValueError):
                distribution.verify_installer_report(changed, name, "12" * 32)

    def test_payload_requires_exact_app_and_install_location(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            app = root / "PicFetch.app"
            app.mkdir()
            (app / "fixture").write_bytes(b"signed-content")
            expanded = root / "expanded"
            component = expanded / "component.pkg"
            payload = component / "Payload"
            payload.mkdir(parents=True)
            shutil.copytree(app, payload / app.name)
            info = component / "PackageInfo"
            original = f'<pkg-info identifier="{APP_ID}" install-location="/Applications"/>'
            info.write_text(original)
            distribution.verify_payload(expanded, app)
            for xml in (original.replace("/Applications", "/Library"),
                        original.replace(APP_ID, "other.app"),
                        original.replace('/>', '><scripts/></pkg-info>')):
                info.write_text(xml)
                with self.assertRaises(ValueError):
                    distribution.verify_payload(expanded, app)
            info.write_text(original)
            (payload / app.name / "fixture").write_bytes(b"tampered")
            with self.assertRaisesRegex(ValueError, "differs"):
                distribution.verify_payload(expanded, app)
            (payload / app.name / "fixture").write_bytes(b"signed-content")
            (payload / "extra").write_text("extra payload")
            with self.assertRaisesRegex(ValueError, "unexpected installer payload"):
                distribution.verify_payload(expanded, app)


@unittest.skipUnless(os.environ.get("PICFETCH_TEST_MAC_APP_BUNDLE"), "requires a built signed app")
class NativeInstallerGuards(unittest.TestCase):
    def test_ad_hoc_app_cannot_pass_distribution_identity_check(self):
        app = Path(os.environ["PICFETCH_TEST_MAC_APP_BUNDLE"]).resolve()
        with self.assertRaises(subprocess.CalledProcessError):
            distribution.verify_code_identities(app, "ABCDEFGHIJ", "A" * 40, os.environ.copy())

    def test_productbuild_roundtrip_preserves_signed_app(self):
        # Deliberately unsigned installer: this qualifies payload assembly only.
        app = Path(os.environ["PICFETCH_TEST_MAC_APP_BUNDLE"]).resolve()
        before = distribution.bundle_files(app)
        with tempfile.TemporaryDirectory(prefix="picfetch-installer-guard-") as temporary:
            root = Path(temporary)
            installer = root / "PicFetch.pkg"
            run("productbuild", "--component", app, "/Applications", installer)
            expanded = root / "expanded"
            run("pkgutil", "--expand-full", installer, expanded)
            distribution.verify_payload(expanded, app)
            extracted = next(expanded.rglob("Payload")) / "PicFetch.app"
            run("codesign", "--verify", "--deep", "--strict", extracted)
        self.assertEqual(distribution.bundle_files(app), before)


class DistributionAssembly(unittest.TestCase):
    def test_signs_inside_out_and_publishes_verified_payload_only(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "input" / "PicFetch.app"
            for relative in ("Contents/MacOS/PicFetch", "Contents/Frameworks/runtime.dylib",
                             "Contents/MacOS/picfetch-worker-client",
                             "Contents/XPCServices/" + APP_ID + ".worker.xpc/Contents/MacOS/worker",
                             "Contents/XPCServices/" + APP_ID + ".worker.xpc/Contents/MacOS/picfetch-image-worker"):
                path = source / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("fixture")
            original = distribution.bundle_files(source)
            (source.parent / "manifest.json").write_text(json.dumps({"files": {name: value["sha256"] for name, value in original.items()}}))
            app_key = "A" * 40
            installer_certificate = b"fixture installer certificate"
            installer_key = hashlib.sha1(installer_certificate).hexdigest().upper()
            installer_sha256 = hashlib.sha256(installer_certificate).hexdigest().upper()
            installer_name = "3rd Party Mac Developer Installer: Fixture (ABCDEFGHIJ)"
            listing = f' 1) {app_key} "Apple Distribution: Fixture (ABCDEFGHIJ)"\n 2) {installer_key} "{installer_name}"\n'
            commands = []
            candidate = None

            def fake_run(*args, **_):
                nonlocal candidate
                args = list(map(str, args))
                commands.append(args)
                if args[:2] == ["security", "find-identity"]:
                    return listing.encode()
                if args[:2] == ["security", "find-certificate"]:
                    return ssl.DER_cert_to_PEM_cert(installer_certificate).encode()
                if args[0] == "ditto":
                    shutil.copytree(args[1], args[2])
                    candidate = Path(args[2])
                elif args[0] == "productbuild":
                    self.assertEqual(args[1:4], ["--component", str(candidate), "/Applications"])
                    self.assertEqual(args[4:7], ["--sign", installer_name, "--timestamp"])
                    Path(args[-1]).write_bytes(b"package")
                elif args[:2] == ["pkgutil", "--check-signature"]:
                    return f"Status: signed by a certificate trusted by macOS\n 1. {installer_name}\n SHA256 Fingerprint: {installer_sha256}\n".encode()
                elif args[:2] == ["pkgutil", "--expand-full"]:
                    component = Path(args[-1]) / "component.pkg"
                    payload = component / "Payload"
                    payload.mkdir(parents=True)
                    shutil.copytree(candidate, payload / "PicFetch.app")
                    (component / "PackageInfo").write_text(f'<pkg-info identifier="{APP_ID}" install-location="/Applications"/>')
                return None

            args = Namespace(app=str(source), out=str(root / "output"), team="ABCDEFGHIJ",
                             app_identity=app_key, installer_identity=installer_key,
                             profile="", worker_profile="", testflight=False)
            with patch.object(distribution, "run", side_effect=fake_run), patch.object(distribution, "verify") as verify:
                distribution.sign_candidate(args)
                self.assertEqual(verify.call_count, 2)
            signed = [cmd for cmd in commands if cmd[:2] == ["codesign", "--force"]]
            self.assertEqual([cmd[-1].split("/")[-1] for cmd in signed],
                             ["runtime.dylib", "picfetch-worker-client", "picfetch-image-worker", APP_ID + ".worker.xpc", "PicFetch.app"])
            self.assertTrue(all(cmd[2:4] == ["--sign", app_key] for cmd in signed))
            validations = [cmd for cmd in commands if cmd[:2] == ["codesign", "--verify"]]
            self.assertEqual(len(validations), 5)
            self.assertTrue(all('anchor apple generic' in cmd[-2] and app_key in cmd[-2] for cmd in validations))
            self.assertEqual(distribution.bundle_files(source), original)
            self.assertTrue((Path(args.out) / "PicFetch.pkg").is_file())
            self.assertTrue((Path(args.out) / "manifest.json").is_file())

    def test_profile_snapshot_is_the_same_data_decoded_and_embedded(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "profile"
            source.write_bytes(b"original CMS")

            def decode(*args, **_):
                snapshot = Path(args[-1])
                self.assertEqual(snapshot.read_bytes(), b"original CMS")
                source.write_bytes(b"changed after snapshot")
                return distribution.plistlib.dumps({"fixture": True})

            with patch.object(distribution, "run", side_effect=decode):
                data, profile = distribution.read_profile(source, {})
            self.assertEqual(data, b"original CMS")
            self.assertEqual(profile, {"fixture": True})

    def test_testflight_requires_both_bundle_profiles_before_side_effects(self):
        for app_profile, worker_profile in (("", ""), ("app", ""), ("", "worker")):
            args = Namespace(team="ABCDEFGHIJ", testflight=True, profile=app_profile, worker_profile=worker_profile)
            with patch.object(distribution, "run") as run_tool, self.assertRaisesRegex(ValueError, "app and XPC"):
                distribution.sign_candidate(args)
            run_tool.assert_not_called()
