"""Prepare a distribution-signed Store candidate without uploading or installing it.

Profile plist checks are diagnostics. Apple's profile/SDK/submission acceptance
remains a separate gate; this tool does not claim to authenticate a CMS profile.
"""
import argparse
import copy
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import ssl
import tempfile
import xml.etree.ElementTree as ET

from package import APP_ID, ENTITLEMENTS, digest, run, verify, write_plist


def select_identity(output, fingerprint, team, role):
    if not re.fullmatch(r"[A-F0-9]{40}", fingerprint) or not re.fullmatch(r"[A-Z0-9]{10}", team):
        raise ValueError("explicit SHA-1 identity fingerprint and 10-character Team ID required")
    identities = re.findall(r'^\s*\d+\) ([A-F0-9]{40}) "([^"\n]+)"\s*$', output, re.M)
    selected = [name for key, name in identities if key == fingerprint]
    prefixes = {"app": ("Apple Distribution: ", "3rd Party Mac Developer Application: "),
                "installer": ("3rd Party Mac Developer Installer: ",)}
    if (len(selected) != 1 or not selected[0].startswith(prefixes[role]) or
            not selected[0].endswith("(" + team + ")") or
            sum(name == selected[0] for _, name in identities) != 1):
        raise ValueError(f"missing, ambiguous or wrong-team {role} Store identity")
    return selected[0]


def profile_entitlements(profile, team, identifier, certificate, role, now):
    if role not in ("app", "service") or not isinstance(profile, dict):
        raise ValueError("profile requires an application or XPC service")
    claims = profile.get("Entitlements", {})
    prefix = profile.get("ApplicationIdentifierPrefix", [])
    certificates = profile.get("DeveloperCertificates", [])
    if (not isinstance(claims, dict) or profile.get("TeamIdentifier") != [team] or
            profile.get("Platform") != ["OSX"] or not isinstance(prefix, list) or
            len(prefix) != 1 or not isinstance(prefix[0], str) or
            not re.fullmatch(r"[A-Z0-9]{10}", prefix[0]) or
            claims.get("com.apple.application-identifier") != prefix[0] + "." + identifier or
            claims.get("com.apple.developer.team-identifier") != team):
        raise ValueError("profile platform, Team or explicit application identifier mismatch")
    if ("ProvisionedDevices" in profile or profile.get("ProvisionsAllDevices", False) is not False or
            claims.get("get-task-allow", False) is not False or
            claims.get("com.apple.security.get-task-allow", False) is not False):
        raise ValueError("profile is not restricted to Store distribution")
    created, expires = profile.get("CreationDate"), profile.get("ExpirationDate")
    if not isinstance(created, datetime) or not isinstance(expires, datetime):
        raise ValueError("profile lacks validity dates")
    created = created.replace(tzinfo=timezone.utc) if created.tzinfo is None else created
    expires = expires.replace(tzinfo=timezone.utc) if expires.tzinfo is None else expires
    if not created <= now < expires:
        raise ValueError("profile is expired or not yet valid")
    if not isinstance(certificates, list) or certificate not in {
            hashlib.sha1(cert).hexdigest().upper() for cert in certificates if isinstance(cert, bytes)}:
        raise ValueError("profile does not authorize the selected signing certificate")
    result = copy.deepcopy(ENTITLEMENTS[role])
    result.update({"com.apple.application-identifier": claims["com.apple.application-identifier"],
                   "com.apple.developer.team-identifier": team})
    if "beta-reports-active" in claims:
        if claims["beta-reports-active"] is not True:
            raise ValueError("unexpected beta-reports-active profile claim")
        result["beta-reports-active"] = True
    return result


def bundle_files(app):
    result = {}
    for path in sorted(app.rglob("*")):
        if path.is_symlink():
            raise ValueError("unexpected bundle symlink")
        if path.is_file():
            result[str(path.relative_to(app))] = {"sha256": digest(path),
                                                 "executable": bool(path.stat().st_mode & 0o111)}
    return result


def verify_payload(expanded, app):
    infos = list(expanded.rglob("PackageInfo"))
    if len(infos) != 1:
        raise ValueError("installer must contain exactly one component")
    info = ET.parse(infos[0]).getroot()
    if (info.get("identifier") != APP_ID or info.get("install-location") != "/Applications" or
            info.find("scripts") is not None):
        raise ValueError("unexpected installer identity, destination or scripts")
    payload = infos[0].parent / "Payload"
    if not payload.is_dir() or {p.name for p in payload.iterdir()} != {"PicFetch.app"}:
        raise ValueError("unexpected installer payload")
    if bundle_files(payload / "PicFetch.app") != bundle_files(app):
        raise ValueError("installer payload differs from the verified signed app")


def verify_installer_report(report, name, fingerprint):
    # pkgutil must also have exited successfully; parsing adds exact leaf selection.
    leaf = re.findall(r"^\s*1\.\s+(.+?)\s*$", report, re.M)
    status = re.findall(r"^\s*Status:\s*(.+?)\s*$", report, re.M)
    accepted = ("signed by a developer certificate issued by Apple",
                "signed by a certificate trusted by macOS",
                "signed by a certificate trusted by Mac OS X")
    if leaf != [name] or len(status) != 1 or not status[0].startswith(accepted):
        raise ValueError("installer signature does not match the selected trusted identity")
    leaf_block = re.search(r"^\s*1\..*?\n(.*?)(?=^\s*2\.|\Z)", report, re.M | re.S)
    fingerprints = re.findall(r"SHA256 Fingerprint:\s*((?:[0-9A-Fa-f]{2}[ \t\r\n]*){32})",
                              leaf_block[1] if leaf_block else "")
    if len(fingerprints) != 1 or re.sub(r"\s", "", fingerprints[0]).upper() != fingerprint:
        raise ValueError("installer leaf certificate fingerprint mismatch")


def installer_certificate(name, fingerprint, env):
    certificates = run("security", "find-certificate", "-a", "-c", name, "-p", capture=True, env=env).decode()
    selected = []
    for pem in re.findall(r"-----BEGIN CERTIFICATE-----.*?-----END CERTIFICATE-----", certificates, re.S):
        data = ssl.PEM_cert_to_DER_cert(pem)
        if hashlib.sha1(data).hexdigest().upper() == fingerprint:
            selected.append(data)
    if len(selected) != 1:
        raise ValueError("installer public certificate missing or ambiguous")
    return hashlib.sha256(selected[0]).hexdigest().upper()


def code_items(app):
    contents = app / "Contents"
    service = contents / "XPCServices" / (APP_ID + ".worker.xpc")
    return ([(p, None, p.name) for p in sorted((contents / "Frameworks").iterdir())] +
            [(contents / "MacOS/picfetch-worker-client", "helper", APP_ID + ".worker-client"),
             (service / "Contents/MacOS/picfetch-image-worker", "helper", APP_ID + ".image-worker"),
             (service, "service", APP_ID + ".worker"), (app, "app", APP_ID)])


def verify_code_identities(app, team, fingerprint, env):
    for path, _, identifier in code_items(app):
        requirement = (f'=anchor apple generic and certificate leaf = H"{fingerprint}" '
                       f'and certificate leaf[subject.OU] = "{team}" and identifier "{identifier}"')
        run("codesign", "--verify", "--strict", "--all-architectures", "-R", requirement, path, env=env)


def read_profile(path, env):
    # Decode exactly the bytes later embedded, even if the source file changes.
    data = path.read_bytes()
    with tempfile.TemporaryDirectory(prefix="picfetch-profile-") as temporary:
        snapshot = Path(temporary) / "profile.provisionprofile"
        snapshot.write_bytes(data)
        decoded = run("security", "cms", "-D", "-i", snapshot, capture=True, env=env)
    return data, plistlib.loads(decoded)


def sign_candidate(args):
    if not re.fullmatch(r"[A-Z0-9]{10}", args.team):
        raise ValueError("10-character developer Team ID required")
    if args.testflight and not (args.profile and args.worker_profile):
        raise ValueError("TestFlight route requires explicit app and XPC worker profiles")
    app, out = Path(args.app).resolve(), Path(args.out).absolute()
    if out.exists() or out.is_symlink() or out.is_relative_to(app):
        raise ValueError("choose a fresh output directory outside the input app")
    env = os.environ | {"DEVELOPER_DIR": os.environ.get("APPLE_STORE_DEVELOPER_DIR", "/Applications/Xcode.app/Contents/Developer"),
                        "LC_ALL": "C", "LANG": "C"}
    # Inspect public identity metadata only; private keys remain in the Keychain.
    app_identities = run("security", "find-identity", "-v", "-p", "codesigning", capture=True, env=env).decode()
    identities = run("security", "find-identity", "-v", capture=True, env=env).decode()
    select_identity(app_identities, args.app_identity, args.team, "app")
    installer_name = select_identity(identities, args.installer_identity, args.team, "installer")
    installer_fingerprint = installer_certificate(installer_name, args.installer_identity, env)
    verify(app, env)
    source_manifest = json.loads((app.parent / "manifest.json").read_text())
    original = bundle_files(app)
    if {name: value["sha256"] for name, value in original.items()} != source_manifest.get("files"):
        raise ValueError("input app does not match its local qualification manifest")
    entitlements = copy.deepcopy(ENTITLEMENTS)
    profiles = {}
    for role, value in (("app", args.profile), ("service", args.worker_profile)):
        if value:
            path = Path(value).resolve()
            data, profile = read_profile(path, env)
            entitlements[role] = profile_entitlements(profile, args.team,
                                                      APP_ID if role == "app" else APP_ID + ".worker",
                                                      args.app_identity, role, datetime.now(timezone.utc))
            profiles[role] = data
    out.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".picfetch-distribution-", dir=out.parent) as temporary:
        stage = Path(temporary)
        candidate = stage / "PicFetch.app"
        run("ditto", app, candidate, env=env)
        if bundle_files(candidate) != original:
            raise ValueError("staged app differs from qualified input")
        for role, data in profiles.items():
            bundle = candidate if role == "app" else candidate / "Contents/XPCServices" / (APP_ID + ".worker.xpc")
            (bundle / "Contents/embedded.provisionprofile").write_bytes(data)
        for role, values in entitlements.items():
            write_plist(stage / (role + ".plist"), values)
        for path, role, identifier in code_items(candidate):
            claims = ["--entitlements", stage / (role + ".plist")] if role else []
            run("codesign", "--force", "--sign", args.app_identity, "--timestamp", "--identifier", identifier,
                *claims, path, env=env)
        verify(candidate, env, expected_entitlements=entitlements)
        verify_code_identities(candidate, args.team, args.app_identity, env)
        installer = stage / "PicFetch.pkg"
        run("productbuild", "--component", candidate, "/Applications", "--sign", installer_name,
            "--timestamp", installer, env=env)
        report = run("pkgutil", "--check-signature", installer, capture=True, env=env).decode()
        verify_installer_report(report, installer_name, installer_fingerprint)
        expanded = stage / "expanded"
        run("pkgutil", "--expand-full", installer, expanded, env=env)
        verify_payload(expanded, candidate)
        manifest = {"purpose": "distribution-signed candidate; Apple submission validation still required",
                    "team": args.team, "app_identity": args.app_identity,
                    "installer_identity": args.installer_identity, "installer_certificate_sha256": installer_fingerprint,
                    "testflight": args.testflight,
                    "profiles": {role: hashlib.sha256(data).hexdigest() for role, data in profiles.items()},
                    "input": source_manifest, "files": bundle_files(candidate), "installer_sha256": digest(installer)}
        if bundle_files(app) != original:
            raise ValueError("input app changed during packaging")
        out.mkdir()
        shutil.move(candidate, out / candidate.name)
        shutil.move(installer, out / installer.name)
        (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(f"Distribution candidate: {out}; not uploaded or installed", flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for flag, variable in (("app", "APPLE_STORE_APP"), ("out", "APPLE_STORE_SIGNED_OUTPUT_DIR"),
                           ("team", "APPLE_STORE_TEAM_ID"), ("app-identity", "APPLE_STORE_APP_IDENTITY"),
                           ("installer-identity", "APPLE_STORE_INSTALLER_IDENTITY"),
                           ("profile", "APPLE_STORE_PROFILE"), ("worker-profile", "APPLE_STORE_WORKER_PROFILE")):
        parser.add_argument("--" + flag, default=os.environ.get(variable, ""))
    parser.add_argument("--testflight", action="store_true", default=os.environ.get("APPLE_STORE_TESTFLIGHT") == "1")
    options = parser.parse_args()
    if not all((options.app, options.out, options.team, options.app_identity, options.installer_identity)):
        parser.error("input app, fresh output, Team and both identity SHA-1 fingerprints are required")
    sign_candidate(options)
