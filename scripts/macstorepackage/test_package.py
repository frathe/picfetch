"""Policy guards complement the actual signed bundle qualification."""
import importlib.util
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
