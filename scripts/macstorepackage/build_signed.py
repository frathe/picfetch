"""Build both Mac architectures and sign a fresh Store candidate without uploading."""
import json
import os
from pathlib import Path
import subprocess
import tempfile


REQUIRED = (
    "APPLE_STORE_TEAM_ID", "APPLE_STORE_APP_IDENTITY", "APPLE_STORE_INSTALLER_IDENTITY",
    "APPLE_STORE_PROFILE", "APPLE_STORE_WORKER_PROFILE",
    "APPLE_STORE_ARM64_ARCHIVE", "APPLE_STORE_AMD64_ARCHIVE",
)


def build_signed(repo, settings):
    allowed = (*REQUIRED, "APPLE_STORE_DEVELOPER_DIR")
    if not isinstance(settings, dict) or any(
            key not in allowed or not isinstance(value, str) for key, value in settings.items()):
        raise ValueError("local signing configuration must contain only documented string settings")
    environment = settings | dict(os.environ)
    missing = [key for key in REQUIRED if not environment.get(key)]
    if missing:
        raise ValueError("configure local signing inputs: " + ", ".join(missing))
    environment["APPLE_STORE_TESTFLIGHT"] = "1"
    subprocess.run(["make", "apple-store-preflight"], cwd=repo, env=environment, check=True)
    (repo / "bin").mkdir(exist_ok=True)
    root = Path(tempfile.mkdtemp(prefix="apple-store-candidate-", dir=repo / "bin"))
    environment.update({
        "APPLE_STORE_OUTPUT_DIR": str(root / "local"),
        "APPLE_STORE_APP": str(root / "local/PicFetch.app"),
        "APPLE_STORE_SIGNED_OUTPUT_DIR": str(root / "signed"),
    })
    subprocess.run(["make", "apple-store-package-local"], cwd=repo, env=environment, check=True)
    subprocess.run(["make", "apple-store-package-signed"], cwd=repo, env=environment, check=True)
    print(f"Store candidate: {root / 'signed/PicFetch.pkg'}; not uploaded or submitted", flush=True)
    return root / "signed"


if __name__ == "__main__":
    repository = Path(__file__).resolve().parents[2]
    config = repository / ".scratch/apple-store-signing.json"
    build_signed(repository, json.loads(config.read_text()) if config.exists() else {})
