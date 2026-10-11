"""Qualify packaged production workers in disposable per-architecture apps.

The original app is never modified. Only the main GUI executable is replaced by
a Go driver that calls the production clients and unchanged packaged broker. Models and a generated
image live in the derived app resources. The driver copies the image into its
private container and tests transfer to the worker along with a private cache. Intel execution on Apple Silicon uses Rosetta.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess
import tempfile
import zlib

from package import APP_ID, ENTITLEMENTS, TARGETS, build_environment, run, write_plist


def png():
    def chunk(kind, data):
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data))
    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", 32, 32, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress((b"\x00" + b"\x30\x80\xc0" * 32) * 32)) + chunk(b"IEND", b""))


def qualify(app, models, deny_source=False):
    repo = Path(__file__).resolve().parents[2]
    env = os.environ | {"DEVELOPER_DIR": os.environ.get("APPLE_STORE_DEVELOPER_DIR", "/Applications/Xcode.app/Contents/Developer")}
    results = []
    for native, minimum in TARGETS.values():
        with tempfile.TemporaryDirectory(prefix="picfetch-packaged-worker-", dir="/private/tmp") as temporary:
            stage = Path(temporary)
            derived = stage / "PicFetch.app"
            shutil.copytree(app, derived)
            contents = derived / "Contents"
            service = contents / "XPCServices" / (APP_ID + ".worker.xpc")
            executable = contents / "MacOS/PicFetch"
            for path in (contents / "MacOS/picfetch-worker-client", service / "Contents/MacOS/worker",
                         service / "Contents/MacOS/picfetch-image-worker"):
                thin = stage / path.name
                run("xcrun", "lipo", path, "-thin", native, "-output", thin, env=env)
                shutil.copy2(thin, path)
            executable.unlink()
            goarch = "arm64" if native == "arm64" else "amd64"
            build_flags = []
            if deny_source:
                source = repo / "internal/similarity/worker_access.go"
                original = source.read_text()
                marker = "for _, path := range paths {"
                if original.count(marker) != 1:
                    raise ValueError("source-grant qualification overlay no longer matches")
                patched = stage / "worker_access.go"
                patched.write_text(original.replace(marker, "for _, path := range paths[:0] {"))
                overlay = stage / "source-denial.json"
                overlay.write_text(json.dumps({"Replace": {str(source): str(patched)}}))
                build_flags = ["-overlay", str(overlay)]
            run_flags = ["--expect-source-denial"] if deny_source else []
            run("go", "build", *build_flags, "-tags", "no_emoji,nodynamic,appleappstore", "-o", executable,
                "./scripts/macstorequalify", cwd=repo,
                env=build_environment(env, goarch, minimum))
            resources = contents / "Resources"
            (resources / "models").mkdir()
            for name in ("vision_model.onnx", "preprocessor_config.json"):
                shutil.copyfile(models / name, resources / "models" / name)
            (resources / "fixture.png").write_bytes(png())
            for role, value in ENTITLEMENTS.items():
                write_plist(stage / (role + ".plist"), value)
            for path, role in ((contents / "MacOS/picfetch-worker-client", "helper"),
                               (service / "Contents/MacOS/picfetch-image-worker", "helper"),
                               (service, "service"), (derived, "app")):
                run("codesign", "--force", "--sign", "-", "--entitlements", stage / (role + ".plist"), path, env=env)
            run("codesign", "--verify", "--deep", "--strict", derived, env=env)
            result = subprocess.run([str(executable), *run_flags], capture_output=True, timeout=180)
            if result.returncode:
                raise ValueError(f"worker qualification failed ({result.returncode}): {result.stdout!r} {result.stderr!r}")
            evidence = json.loads(result.stdout)
            if deny_source:
                print(f"PASS {native}: production worker refused the private source without its grant", flush=True)
            else:
                print(f"PASS {native}: real HEIC pixels, ONNX inference, private source/cache grants, cache reuse, retained search, TCP/UDP denial and worker exit", flush=True)
            results.append({"architecture": native, **evidence})
    return results


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app", required=True)
    parser.add_argument("--models", required=True)
    parser.add_argument("--result", required=True)
    parser.add_argument("--deny-source", action="store_true", help="qualify refusal with source export disabled in the test parent")
    args = parser.parse_args()
    evidence = qualify(Path(args.app).resolve(), Path(args.models).resolve(), args.deny_source)
    Path(args.result).write_text(json.dumps(evidence, indent=2) + "\n")
