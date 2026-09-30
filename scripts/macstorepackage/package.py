"""Build and validate an ad-hoc, universal Apple Store qualification app.

This route has no distribution signing or upload mode. Apple account credentials
are unnecessary; the result cannot be submitted as an App Store release.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import subprocess
import tempfile
import tomllib


ENTITLEMENTS = {
    "app": {"com.apple.security.app-sandbox": True,
            "com.apple.security.network.client": True,
            "com.apple.security.files.user-selected.read-write": True,
            "com.apple.security.files.bookmarks.app-scope": True},
    "service": {"com.apple.security.app-sandbox": True},
    "helper": {"com.apple.security.app-sandbox": True,
               "com.apple.security.inherit": True},
}


APP_ID = "io.github.frathe.picfetch"
TARGETS = {"arm64": ("arm64", "14.0"), "amd64": ("x86_64", "13.4")}
TAGS = "no_emoji,nodynamic,appleappstore"


def check_dependencies(paths):
    for path in paths.splitlines():
        if not path.startswith(("/System/Library/", "/usr/lib/")):
            raise ValueError(f"non-system native dependency: {path}")


def check_entitlements(role, actual):
    if actual != ENTITLEMENTS[role]:
        raise ValueError(f"unexpected {role} entitlements: {actual}")


def check_macho(actual_arches, expected_arches, load_commands, minimum):
    if set(actual_arches.split()) != expected_arches:
        raise ValueError(f"unexpected architectures: {actual_arches}")
    versions = re.findall(r"minos (\d+\.\d+(?:\.\d+)?)", load_commands)
    if not versions:
        versions = re.findall(r"version (\d+\.\d+(?:\.\d+)?)", load_commands)
    normalize = lambda v: tuple((list(map(int, v.split("."))) + [0, 0])[:3])
    if not versions or any(normalize(v) > normalize(minimum) for v in versions):
        raise ValueError(f"native minimum OS exceeds {minimum}: {versions}")


def run(*args, cwd=None, env=None, capture=False):
    print("+", " ".join(map(str, args)), flush=True)
    result = subprocess.run(list(map(str, args)), cwd=cwd, env=env, check=True,
                            stdout=subprocess.PIPE if capture else None)
    return result.stdout if capture else None


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def write_plist(path, data):
    path.write_bytes(plistlib.dumps(data, sort_keys=True))


def native_check(path, architectures, minima, env):
    actual = run("xcrun", "lipo", "-archs", path, env=env, capture=True).decode().strip()
    for arch in architectures:
        commands = run("xcrun", "otool", "-arch", arch, "-l", path, env=env, capture=True).decode()
        # Keep only the deployment command; dylib compatibility versions are unrelated.
        deployment = re.findall(r"cmd LC_(?:BUILD_VERSION|VERSION_MIN_MACOSX)\n(.*?)(?=Load command|\Z)", commands, re.S)
        check_macho(actual, architectures, "\n".join(deployment), minima[arch])
        linked = run("xcrun", "otool", "-arch", arch, "-L", path, env=env, capture=True).decode().splitlines()[1:]
        dependencies = [line.strip().split(" (", 1)[0] for line in linked]
        # otool lists a dylib's install ID first; it is not a dependency.
        if path.suffix == ".dylib":
            dependencies = dependencies[1:]
        check_dependencies("\n".join(dependencies))


def verify(app, env):
    contents = app / "Contents"
    service = contents / "XPCServices" / (APP_ID + ".worker.xpc")
    roles = {app: "app", service: "service",
             contents / "MacOS/picfetch-worker-client": "helper",
             service / "Contents/MacOS/picfetch-image-worker": "helper"}
    for path, role in roles.items():
        data = run("codesign", "-d", "--entitlements", ":-", path, capture=True, env=env)
        check_entitlements(role, plistlib.loads(data))
        run("codesign", "--verify", "--strict", path, env=env)
    plist = plistlib.loads((contents / "Info.plist").read_bytes())
    if plist.get("CFBundleIdentifier") != APP_ID or plist.get("CFBundleExecutable") != "PicFetch":
        raise ValueError("unexpected app identity")
    if plist.get("LSMinimumSystemVersionByArchitecture") != {a: m for a, m in TARGETS.values()}:
        raise ValueError("unexpected per-architecture deployment minimum")
    expected_code = {contents / "MacOS/PicFetch", contents / "MacOS/picfetch-worker-client",
                     service / "Contents/MacOS/worker", service / "Contents/MacOS/picfetch-image-worker"}
    for path in expected_code:
        native_check(path, {"arm64", "x86_64"}, {a: m for a, m in TARGETS.values()}, env)
    libraries = {"libonnxruntime.1.29.0.dylib": ("arm64", "14.0"),
                 "libonnxruntime.1.23.2.dylib": ("x86_64", "13.4")}
    if {p.name for p in (contents / "Frameworks").iterdir()} != set(libraries):
        raise ValueError("unexpected bundled runtime files")
    for name, (arch, minimum) in libraries.items():
        path = contents / "Frameworks" / name
        native_check(path, {arch}, {arch: minimum}, env)
        run("codesign", "--verify", "--strict", path, env=env)
        data = run("codesign", "-d", "--entitlements", ":-", path, capture=True, env=env)
        if data.strip() and plistlib.loads(data):
            raise ValueError("native libraries must not carry entitlements")
        expected_code.add(path)
    # Reject extra executable payloads or symlinks rather than silently signing them.
    for path in contents.rglob("*"):
        if path.is_symlink():
            raise ValueError(f"unexpected bundle symlink: {path}")
        if path.is_file() and path not in expected_code:
            with path.open("rb") as source:
                magic = source.read(4)
            if os.access(path, os.X_OK) or magic in (b"\xcf\xfa\xed\xfe", b"\xce\xfa\xed\xfe", b"\xfe\xed\xfa\xcf", b"\xfe\xed\xfa\xce", b"\xca\xfe\xba\xbe", b"\xca\xfe\xba\xbf"):
                raise ValueError(f"unexpected executable payload: {path}")
    run("codesign", "--verify", "--deep", "--strict", app, env=env)


def build_environment(env, arch, minimum):
    # cgo hashes these flags into its cache key. MACOSX_DEPLOYMENT_TARGET alone
    # can reuse objects compiled for a newer SDK deployment target.
    flags = "-O2 -g -mmacosx-version-min=" + minimum
    return env | {"GOOS": "darwin", "GOARCH": arch, "CGO_ENABLED": "1",
                  "MACOSX_DEPLOYMENT_TARGET": minimum, "CC": "clang", "CXX": "clang++",
                  "CGO_CFLAGS": flags, "CGO_CXXFLAGS": flags, "CGO_LDFLAGS": flags}


def build(args):
    repo = Path(__file__).resolve().parents[2]
    env = os.environ | {"DEVELOPER_DIR": os.environ.get("APPLE_STORE_DEVELOPER_DIR", "/Applications/Xcode.app/Contents/Developer")}
    if args.verify:
        verify(Path(args.verify).resolve(), env)
        return
    out = Path(args.out).absolute()
    if out.exists() or out.is_symlink():
        raise ValueError("output already exists; choose a fresh directory")
    archives = {"arm64": Path(args.arm64_archive).resolve(), "amd64": Path(args.amd64_archive).resolve()}
    if any(not p.is_file() for p in archives.values()):
        raise ValueError("both pinned ONNX archives are required")
    fyne = repo / ".tools/fyne-v1.7.2/fyne"
    if not fyne.is_file():
        raise ValueError("run make install-fyne first")
    details = tomllib.loads((repo / "FyneApp.toml").read_text())["Details"]
    if details["ID"] != APP_ID or details["Name"] != "PicFetch":
        raise ValueError("unexpected source app identity")
    run("xcodebuild", "-version", env=env)
    out.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".picfetch-store-", dir=out.parent) as temporary:
        stage = Path(temporary)
        runtime = stage / "runtime"
        # Verify all downloaded code before spending time compiling the application.
        for arch, archive in archives.items():
            run("go", "run", "-tags", TAGS, "./scripts/macstorestage", "-arch", arch,
                "-archive", archive, "-out", runtime / arch, cwd=repo, env=env)
        for arch, (native, minimum) in TARGETS.items():
            archdir = stage / arch
            archdir.mkdir()
            buildenv = build_environment(env, arch, minimum)
            run("go", "build", "-tags", TAGS, "-trimpath", "-ldflags=-s -w", "-o", archdir / "PicFetch", ".", cwd=repo, env=buildenv)
            for source, name in (("client.m", "picfetch-worker-client"), ("service.m", "worker")):
                run("xcrun", "clang", "-arch", native, "-mmacosx-version-min=" + minimum,
                    "-fobjc-arc", "-Wall", "-Wextra", "-Werror", "-framework", "Foundation",
                    repo / "internal/macworker/native" / source, "-o", archdir / name, env=env)
        for name in ("PicFetch", "picfetch-worker-client", "worker"):
            run("xcrun", "lipo", "-create", stage / "arm64" / name, stage / "amd64" / name,
                "-output", stage / name, env=env)
        # Fyne validates a source directory even with an explicit executable.
        # A staging-only stub keeps its metadata/build-number writes off the repo.
        (stage / "main.go").write_text("package main\n\nfunc main() {}\n")
        run(fyne, "package", "--os", "darwin", "--executable", stage / "PicFetch",
            "--name", "PicFetch", "--appID", APP_ID, "--appVersion", details["Version"],
            "--appBuild", details["Build"], "--icon", repo / details["Icon"], "--release", cwd=stage, env=env)
        app = stage / "PicFetch.app"
        contents = app / "Contents"
        service = contents / "XPCServices" / (APP_ID + ".worker.xpc")
        (service / "Contents/MacOS").mkdir(parents=True)
        shutil.copy2(stage / "PicFetch", service / "Contents/MacOS/picfetch-image-worker")
        shutil.copy2(stage / "worker", service / "Contents/MacOS/worker")
        shutil.copy2(stage / "picfetch-worker-client", contents / "MacOS/picfetch-worker-client")
        write_plist(service / "Contents/Info.plist", {
            "CFBundleIdentifier": APP_ID + ".worker", "CFBundleExecutable": "worker",
            "CFBundlePackageType": "XPC!", "CFBundleVersion": str(details["Build"]),
            "XPCService": {"ServiceType": "Application"}})
        run("go", "run", "-tags", TAGS, "./scripts/plistdoctypes", contents / "Info.plist", cwd=repo, env=env)
        info = plistlib.loads((contents / "Info.plist").read_bytes())
        info.update({"LSMinimumSystemVersion": "13.4",
                     "LSMinimumSystemVersionByArchitecture": {a: m for a, m in TARGETS.values()},
                     "LSApplicationCategoryType": "public.app-category.photography"})
        write_plist(contents / "Info.plist", info)
        (contents / "Frameworks").mkdir()
        for arch in TARGETS:
            extracted = next((runtime / arch).iterdir())
            library = next((extracted / "lib").glob("*.dylib"))
            shutil.copy2(library, contents / "Frameworks" / library.name)
            notices = contents / "Resources" / extracted.name
            notices.mkdir()
            for name in ("LICENSE", "ThirdPartyNotices.txt", "Privacy.md"):
                shutil.copyfile(extracted / name, notices / name)
        for name in ("LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"):
            shutil.copyfile(repo / name, contents / "Resources" / name)
        for role, entitlements in ENTITLEMENTS.items():
            write_plist(stage / (role + ".plist"), entitlements)
        for library in (contents / "Frameworks").iterdir():
            run("codesign", "--force", "--sign", "-", library, env=env)
        for path, role in ((contents / "MacOS/picfetch-worker-client", "helper"),
                           (service / "Contents/MacOS/picfetch-image-worker", "helper"),
                           (service, "service"), (app, "app")):
            run("codesign", "--force", "--sign", "-", "--entitlements", stage / (role + ".plist"), path, env=env)
        verify(app, env)
        manifest = {"purpose": "ad-hoc local qualification; not for submission",
                    "version": details["Version"], "build": details["Build"],
                    "source": run("git", "rev-parse", "HEAD", cwd=repo, capture=True).decode().strip(),
                    "archives": {arch: digest(path) for arch, path in archives.items()},
                    "source_diff_sha256": hashlib.sha256(run("git", "diff", "--binary", "HEAD", cwd=repo, capture=True)).hexdigest(),
                    "dirty": bool(run("git", "status", "--porcelain", cwd=repo, capture=True)),
                    "untracked_sources": {name: digest(repo / name) for name in run("git", "ls-files", "--others", "--exclude-standard", cwd=repo, capture=True).decode().splitlines() if (repo / name).is_file()},
                    "files": {str(p.relative_to(app)): digest(p)
                              for p in sorted(app.rglob("*")) if p.is_file()}}
        # Reserve only after validation; never replace an existing output directory.
        out.mkdir()
        shutil.move(app, out / app.name)
        (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(f"Qualified local bundle: {out / 'PicFetch.app'}", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--arm64-archive", default=os.environ.get("APPLE_STORE_ARM64_ARCHIVE", ""))
    parser.add_argument("--amd64-archive", default=os.environ.get("APPLE_STORE_AMD64_ARCHIVE", ""))
    parser.add_argument("--out", default=os.environ.get("APPLE_STORE_OUTPUT_DIR", ""))
    parser.add_argument("--verify", help="validate an existing local app without modifying it")
    options = parser.parse_args()
    if not options.verify and not all((options.arm64_archive, options.amd64_archive, options.out)):
        parser.error("both runtime archives and a fresh --out are required")
    build(options)
