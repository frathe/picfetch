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

Add these non-secret environment variables to the same protected environment:

| Variable | Value |
|---|---|
| SIMPLYSIGN_INSTALLER_SHA256 | The reviewed 64-character SHA-256 digest of `SimplySignDesktop-9.4.4.92-64-bit-en.msi`. |
| SIMPLYSIGN_SIGNER_THUMBPRINT | The reviewed 40-character SHA-1 thumbprint of that installer's Authenticode signing certificate. |

Obtain both values from a separately authenticated copy of the installer, not
from the download performed by the release workflow. Update them only after
reviewing a deliberate SimplySign upgrade.

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
from Certum's own server, checks its pinned SHA-256 digest and Authenticode
signer certificate, and leaves the verified `SimplySignDesktop.msi` in the
workspace. The pinned action skips downloading when that file exists and
installs it directly. Its fallback URL uses the reserved `.invalid` domain so
a missing local installer fails instead of fetching another copy.

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
