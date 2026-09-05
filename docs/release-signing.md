# Windows release signing

PicFetch signs the Windows release executables automatically after the tag
release build passes its test gate. The signing key remains in Certum
SimplySign's cloud service; it is never stored in this repository or uploaded
as a PFX file.

## One-time GitHub setup

Create a GitHub Actions environment named release-signing in the PicFetch
repository. Require a maintainer's approval before deployments to that
environment. This is the human release approval: after it, the signing job can
use the protected secrets below.

Add these environment secrets to release-signing, not ordinary repository
secrets:

| Secret | Value |
|---|---|
| CERTUM_USERNAME | The SimplySign account username, normally its e-mail address. |
| CERTUM_OTP_URI | The complete otpauth:// TOTP URI for the SimplySign account. |
| CERTUM_CERT_THUMBPRINT | The code-signing certificate's 40-character SHA-1 thumbprint, without spaces. |

CERTUM_OTP_URI is highly sensitive: it enables unattended generation of the
second-factor code. Anyone who can change the release workflow and obtain this
secret could cause a trusted signature to be made. Keep environment approval
enabled, limit environment access, protect the release branch, and review all
changes to .github/workflows/release.yml.

The certificate must expose a pinless virtual card in SimplySign. Certum's
additional interactive card-PIN prompt cannot be answered safely by this
unattended workflow. Confirm this account setting before the first test tag.

Certum does not currently document an official headless SimplySign API. The
workflow therefore pins the third-party setup action to a reviewed commit
rather than a mutable tag. It also downloads Certum SimplySign Desktop 9.4.4.92
from Certum's own server and checks its Windows Authenticode signature before
the action installs it.

## Release flow

1. A v* tag runs the normal reusable CI test gate.
2. The Linux cross-build produces the two unsigned Windows ZIP artifacts.
3. The sign-windows job waits for the protected release-signing environment,
   downloads those artifacts, and authenticates SimplySign.
4. SignTool signs each picfetch.exe with SHA-256 and Certum's RFC-3161
   timestamp service, then verifies the embedded signature.
5. The job uploads new signed Windows ZIP artifacts.
6. The final release job publishes macOS, Linux, and only the signed Windows
   ZIPs. It does not download the unsigned Windows artifacts.

The timestamp keeps a valid signature trustworthy after the certificate later
expires, provided it was signed while the certificate was valid.

## Recover an unpublished release without moving its tag

If packaging failed after a version tag was pushed, land the reviewed build
fix and recovery workflow on `main`, then dispatch the Release workflow with
the existing tag:

```sh
gh workflow run release.yml --repo frathe/picfetch --ref main -f release-tag=v1.0.1
```

Recovery accepts an existing stable version tag and refuses to overwrite a
GitHub release, including a draft. It resolves the tag to a commit once and
uses that commit for every CI job, application build, and release notes.
The cross-build reads only the Makefile from the dispatched workflow revision,
so its corrected packaging commands can build the original application source.
The tag is never moved, and previous run artifacts are retained.

Both macOS and cross-platform artifacts are rebuilt in the new run. The same
protected `release-signing` environment, SignTool verification, and publication
dependencies still apply. Recovery runs from `main`, and the environment's
required reviewer must approve the signing job as usual.

After the run succeeds, verify all six platform archives in the v1.0.1 release
and confirm the tag's commit is unchanged. The WinGet workflow automatically
follows tag-push releases only; dispatch its existing manual recovery after
the GitHub release is published:

```sh
gh workflow run winget.yml --repo frathe/picfetch --ref main -f release-tag=v1.0.1
```

## First release

Before relying on an automatic public release, run the workflow on a test tag
and inspect both Windows ZIPs after download:

~~~powershell
signtool verify /pa /all /v /tw .\picfetch.exe
~~~

Windows should report a successful signature chain and timestamp. If
authentication or certificate discovery fails, the signing job fails before
the GitHub release job can run; it cannot fall back to publishing unsigned
Windows executables.
