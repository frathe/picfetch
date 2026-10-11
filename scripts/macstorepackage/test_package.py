"""Policy guards complement the actual signed bundle qualification."""
import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("package", Path(__file__).with_name("package.py"))
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class PackagingPolicy(unittest.TestCase):
    def test_built_executable_preserves_runtime_metadata_outside_checkout(self):
        repo = Path(__file__).resolve().parents[2]
        metadata = package.tomllib.loads((repo / "FyneApp.toml").read_text())
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            source, stage = root / "source", root / "stage"
            source.mkdir()
            stage.mkdir()
            for name in ("go.mod", "go.sum"):
                shutil.copyfile(repo / name, source / name)
            icon = source / metadata["Details"]["Icon"]
            icon.parent.mkdir(parents=True)
            shutil.copyfile(repo / metadata["Details"]["Icon"], icon)
            (source / "main.go").write_text('''package main
import (
    "encoding/json"
    "os"
    "fyne.io/fyne/v2/app"
)
func main() {
    meta := app.NewWithID("metadata-test").Metadata()
    if err := json.NewEncoder(os.Stdout).Encode(meta); err != nil { panic(err) }
}
''')
            before = {str(p.relative_to(source)): p.read_bytes()
                      for p in source.rglob("*") if p.is_file()}
            for version, build in ((metadata["Details"]["Version"], metadata["Details"]["Build"]),
                                   ("2.4.6", 789)):
                with self.subTest(version=version):
                    config = {**metadata, "Details": {**metadata["Details"],
                                                      "Version": version, "Build": build}}
                    executable = stage / "metadata-probe"
                    package.build_executable(source, stage, executable, os.environ.copy(), config, tags="ci")
                    actual = json.loads(package.run(executable, cwd=stage, capture=True))
                    for key in ("Version", "Build", "ID", "Name"):
                        self.assertEqual(actual[key], config["Details"][key])
                    self.assertIs(actual["Release"], True)
                    self.assertEqual(actual["Migrations"], metadata["Migrations"])
                    self.assertIsNotNone(actual["Icon"])
                    self.assertEqual(before, {str(p.relative_to(source)): p.read_bytes()
                                             for p in source.rglob("*") if p.is_file()})

    def test_missing_privacy_resource_is_refused(self):
        with tempfile.TemporaryDirectory() as folder:
            with self.assertRaisesRegex(ValueError, "Abseil privacy"):
                package.verify_privacy(Path(folder))

    def test_privacy_resource_preserves_upstream_bytes_and_refuses_changes(self):
        repo = Path(__file__).resolve().parents[2]
        upstream = repo / "packaging/apple-app-store/privacy/abseil"
        with tempfile.TemporaryDirectory() as folder:
            contents = Path(folder)
            package.stage_privacy(repo, contents)
            package.verify_privacy(contents)
            resources = contents / "Resources/AbseilPrivacy.bundle/Contents/Resources"
            for name in ("PrivacyInfo.xcprivacy", "LICENSE", "provenance.json"):
                self.assertEqual((resources / name).read_bytes(), (upstream / name).read_bytes())
                original = (resources / name).read_bytes()
                (resources / name).write_bytes(original + b"changed")
                with self.assertRaisesRegex(ValueError, "Abseil privacy"):
                    package.verify_privacy(contents)
                (resources / name).write_bytes(original)
            declaration = package.plistlib.loads((resources / "PrivacyInfo.xcprivacy").read_bytes())
            self.assertIs(declaration["NSPrivacyTracking"], False)
            self.assertEqual(declaration["NSPrivacyCollectedDataTypes"], [])

    def test_dependency_paths(self):
        package.check_dependencies("/System/Library/Frameworks/Foundation.framework/Versions/C/Foundation\n/usr/lib/libSystem.B.dylib")
        for path in ("/opt/homebrew/lib/libfoo.dylib", "@rpath/libfoo.dylib", "libfoo.dylib"):
            with self.subTest(path=path), self.assertRaises(ValueError):
                package.check_dependencies(path)

    def test_entitlements_are_exact(self):
        for role, expected in package.ENTITLEMENTS.items():
            package.check_entitlements(role, expected.copy())
            for key in expected:
                altered = expected.copy()
                del altered[key]
                with self.assertRaises(ValueError):
                    package.check_entitlements(role, altered)
            altered = expected | {"com.apple.security.network.server": True}
            with self.assertRaises(ValueError):
                package.check_entitlements(role, altered)
        with self.assertRaises(ValueError):
            package.check_entitlements("helper", package.ENTITLEMENTS["app"])

    def test_architecture_and_minimum(self):
        package.check_macho("arm64 x86_64", {"arm64", "x86_64"}, "minos 14.0\n tool LD\n version 27037.1\n", "14.0")
        for architectures, minimum in (("arm64", "14.0"), ("arm64 x86_64", "15.0")):
            with self.assertRaises(ValueError):
                package.check_macho(architectures, {"arm64", "x86_64"}, "minos " + minimum, "14.0")
        with self.assertRaises(ValueError):
            package.check_macho("arm64 x86_64", {"arm64", "x86_64"}, "", "14.0")


@unittest.skipUnless(os.environ.get("PICFETCH_TEST_MAC_APP_BUNDLE"), "requires a built signed app")
class SignedArtifactGuards(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="picfetch-package-guard-")
        self.addCleanup(self.directory.cleanup)
        self.app = Path(self.directory.name) / "PicFetch.app"
        shutil.copytree(os.environ["PICFETCH_TEST_MAC_APP_BUNDLE"], self.app)
        self.env = os.environ.copy()

    def sign(self, path, role):
        entitlement = Path(self.directory.name) / "entitlements.plist"
        package.write_plist(entitlement, package.ENTITLEMENTS[role])
        package.run("codesign", "--force", "--sign", "-", "--entitlements", entitlement, path)

    def test_ad_hoc_runtime_cannot_claim_upstream_identity(self):
        runtime = self.app / "Contents/Frameworks/libonnxruntime.1.29.0.dylib"
        package.run("codesign", "--verify", "--strict", runtime)
        with self.assertRaises(subprocess.CalledProcessError):
            package.verify_upstream_library(runtime, self.env)

    def test_resigned_missing_submission_metadata_is_refused(self):
        path = self.app / "Contents/Info.plist"
        original = package.plistlib.loads(path.read_bytes())
        for key in ("NSHumanReadableCopyright", "LSApplicationCategoryType"):
            for value in (None, "unexpected"):
                with self.subTest(key=key, value=value):
                    data = original.copy()
                    data["NSHumanReadableCopyright"] = "Copyright (c) 2026 Florian Rathe"
                    data["LSApplicationCategoryType"] = "public.app-category.photography"
                    if value is None:
                        del data[key]
                    else:
                        data[key] = value
                    package.write_plist(path, data)
                    self.sign(self.app, "app")
                    package.run("codesign", "--verify", "--deep", "--strict", self.app)
                    with self.assertRaisesRegex(ValueError, "submission metadata"):
                        package.verify(self.app, self.env)

    def test_quarantined_bundle_is_refused_without_mutation(self):
        metadata = self.app / "Contents/Info.plist"
        data = package.plistlib.loads(metadata.read_bytes())
        data["NSHumanReadableCopyright"] = "Copyright (c) 2026 Florian Rathe"
        package.write_plist(metadata, data)
        self.sign(self.app, "app")
        for path in (self.app, self.app / "Contents/Resources",
                     self.app / "Contents/Resources/LICENSE"):
            with self.subTest(path=path):
                attribute = "0081;00000000;PicFetchTest;"
                package.run("xattr", "-w", "com.apple.quarantine", attribute, path)
                try:
                    with self.assertRaisesRegex(ValueError, "quarantine"):
                        package.verify(self.app, self.env)
                    self.assertEqual(package.run("xattr", "-p", "com.apple.quarantine", path, capture=True).decode().strip(), attribute)
                finally:
                    package.run("xattr", "-d", "com.apple.quarantine", path)

    def test_resigned_privacy_change_is_refused(self):
        package.stage_privacy(Path(__file__).resolve().parents[2], self.app / "Contents")
        resource = self.app / "Contents/Resources/AbseilPrivacy.bundle/Contents/Resources/PrivacyInfo.xcprivacy"
        declaration = package.plistlib.loads(resource.read_bytes())
        declaration["NSPrivacyTracking"] = True
        package.write_plist(resource, declaration)
        self.sign(self.app, "app")
        package.run("codesign", "--verify", "--deep", "--strict", self.app)
        with self.assertRaisesRegex(ValueError, "Abseil privacy"):
            package.verify(self.app, self.env)

    def test_edited_resource_breaks_seal(self):
        with (self.app / "Contents/Resources/LICENSE").open("a") as target:
            target.write("tampered")
        with self.assertRaises(subprocess.CalledProcessError):
            package.verify(self.app, self.env)

    def test_resigned_network_service_is_refused(self):
        service = self.app / "Contents/XPCServices" / (package.APP_ID + ".worker.xpc")
        self.sign(service, "app")
        self.sign(self.app, "app")
        package.run("codesign", "--verify", "--deep", "--strict", self.app)
        with self.assertRaisesRegex(ValueError, "service entitlements"):
            package.verify(self.app, self.env)

    def test_hidden_native_payload_is_refused(self):
        hidden = self.app / "Contents/Resources/hidden-data"
        shutil.copyfile(self.app / "Contents/MacOS/PicFetch", hidden)
        hidden.chmod(0o644)
        self.sign(self.app, "app")
        with self.assertRaisesRegex(ValueError, "unexpected executable payload"):
            package.verify(self.app, self.env)

    def test_resigned_wrong_minimum_is_refused(self):
        path = self.app / "Contents/Info.plist"
        data = package.plistlib.loads(path.read_bytes())
        data["LSMinimumSystemVersionByArchitecture"]["arm64"] = "10.0"
        package.write_plist(path, data)
        self.sign(self.app, "app")
        with self.assertRaisesRegex(ValueError, "deployment minimum"):
            package.verify(self.app, self.env)


if __name__ == "__main__":
    unittest.main()
