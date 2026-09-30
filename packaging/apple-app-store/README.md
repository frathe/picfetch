# Mac App Store preparation

The Apple channel is experimental. A universal ad-hoc sandbox bundle and real
HEIC/ONNX workers are locally qualified on ARM and Intel under Rosetta. No Apple
Distribution-signed submission or complete GUI sandbox qualification is claimed. Retain all PicFetch features; do not use
temporary filesystem exceptions or weaken the analysis worker's network denial
to pass a packaging check.

The [active plan](../../plans/2026-09-29-apple-app-store.md) owns implementation
and evidence. [Apple requirements](../../docs/apple-app-store-requirements-2026-09-29.md)
records current primary sources and distinguishes native macOS requirements
from mobile-platform privacy rules.

For the remaining owner actions, follow [Ronin's ordered checklist](ronin-checklist.md).

## Build policy

Use `no_emoji,nodynamic,appleappstore` for this macOS channel. Do not combine
`appleappstore` with `microsoftstore`. Apple builds capture Store-managed update
permission at launch, never download a replacement native runtime, and explain
missing bundled code as an App Store repair. This tag alone does not enable
App Sandbox or produce a submission artifact. `make package-mac` continues to
produce the direct-download distribution and must not be submitted to the Store.

`similarity.StageMacRuntime` accepts a pinned macOS architecture's upstream
archive and stages only its verified native library, LICENSE,
ThirdPartyNotices.txt and Privacy.md. The complete archive is verified in a
temporary directory before any payload enters the package. It is a pre-sign
step; the local packager then validates signed Mach-O code and outer resource seals. The same library's SHA-256
changes when signed, so checking it against the original upstream digest after
signing is incorrect.

## Native worker qualification

```sh
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer make apple-worker-test
```

This creates a temporary ad-hoc signed app with network permission and an
embedded XPC service without it. The service launches its own signed inherited
fixture executable from `Contents/MacOS/picfetch-image-worker`. Native sources
compile with warnings treated as errors. The checks exercise both modes,
transferred pipes, TCP/UDP denial, cancellation, leader exit with descendants,
and broker crash. A generated file in the fixture app container is unreadable
from the worker before bookmark resolution, readable afterward, while an
ungranted sibling remains denied. No image/model/user files are used. Fixture container IDs are
separate from PicFetch's production identifier.

Apple-tagged Go launchers use this broker. The production layout is:

- App `Contents/MacOS/picfetch-worker-client`: inherited app sandbox helper.
- App `Contents/XPCServices/io.github.frathe.picfetch.worker.xpc`: independently
  sandboxed service with no network entitlements.
- Service `Contents/MacOS/picfetch-image-worker`: signed sandbox/inherit helper.

The service owns the process group; nested HEIC decoders must not create a new
one. Cancellation terminates the broker with SIGTERM so it can wait for child
retirement. The service kills remaining descendants before reaping the leader,
and connection invalidation independently cancels its child.

This fixture establishes the process boundary on arm64. It does not establish
real ONNX/HEIC execution, security-scoped source grants, cache access, Intel
behavior, provisioning compatibility or acceptance by Apple.

## Developer inputs

Run the read-only prerequisite check:

```sh
APPLE_STORE_TEAM_ID=YOURTEAMID \
APPLE_STORE_TESTFLIGHT=1 \
APPLE_STORE_PROFILE=/absolute/path/PicFetch.provisionprofile \
APPLE_STORE_WORKER_PROFILE=/absolute/path/PicFetchWorker.provisionprofile \
make apple-store-preflight
```

It uses `/Applications/Xcode.app/Contents/Developer` without changing the system
developer-directory selection. Override `APPLE_STORE_DEVELOPER_DIR` for another
full Xcode installation. It checks for Store application and installer signing
identities with private keys. `APPLE_STORE_TESTFLIGHT=1` requires app and XPC worker profile files for this route;
without TestFlight, an app using only unrestricted macOS entitlements may omit
a profile, per [Apple TN3125](https://developer.apple.com/documentation/technotes/tn3125-inside-code-signing-provisioning-profiles).
It does not install credentials, access an Apple account, upload a package or
certify the profile or app. Supplying a profile does not establish its signature,
expiry, registered identifier or entitlement compatibility.

Required inputs:

- Active Apple Developer Program team and a registered explicit bundle ID
  `io.github.frathe.picfetch` available to that team.
- Apple Distribution / Mac App Store application signing certificate with its
  private key, plus a Mac Installer Distribution identity.
- Mac App Store Connect provisioning profile for TestFlight or restricted
  entitlements, matching the identifier and signing certificate. Inherited
  helpers have exactly sandbox/inherit entitlements, without their own profile.
- App Store Connect app record, public support/contact and privacy-policy URLs,
  localized listing, screenshots, category, rating and intended countries.
- Encryption questionnaire determination covering Go's bundled TLS/cryptography;
  do not assume the Apple-OS-only exemption applies.

## Qualification before submission

1. Preserve native open/drop/Dock access as real URL authority and retain
   security-scoped bookmarks for complete sessions and Favorites. Test moved,
   missing, protected and external-volume sources, parent-folder browsing,
   original edits, exports, deletion, clipboard and wallpaper behavior.
2. Use a public sandbox-compatible helper architecture. Verify actual TCP/UDP
   denial in image-analysis workers, current source grants, HEIC decoding,
   inference and worker exit after cancellation, app quit and parent failure.
3. Stage architecture-pinned runtimes with their upstream notices; preserve the
   app's complete dependency/license closure. Bundle models if Apple's review
   requirements or the selected first-run design call for it. Inspect every
   native dependent-library path and minimum OS version.
4. Validate nested code before the outer app signature and provisioning profile;
   finish all plist/resource edits before signing. Inspect sandbox entitlements,
   library signatures, quarantine attributes and the signed installer's payload.
5. Qualify the final Store artifact on Apple Silicon and Intel, with a fresh
   account and without developer codecs/model caches. Retain the exact artifact
   hash, build number and verification output.
6. Reconcile optional tile/download requests with published privacy disclosures
   and App Store privacy answers. Check the entire dependency closure against
   Apple's SDK manifest list; do not copy iOS API-reason requirements onto macOS.
7. Validate and upload with Apple's current tools, then test the actual
   TestFlight/Store installation and upgrade. Submission/upload needs Ronin's
   authorization separately; this preparation does not publish anything.

The local target below is explicitly an ad-hoc qualification route; distribution
real distribution signing and submission validation are still separate qualification work. A successful prerequisite check cannot
substitute for signed sandbox or submission evidence.


## Local universal sandbox bundle

Python 3.11+, full Xcode, Go and the pinned Fyne CLI are used; no developer
certificate or account is needed. Supply both reviewed ONNX archives and a fresh
output directory (an existing output is refused):

```sh
APPLE_STORE_ARM64_ARCHIVE=/absolute/path/onnxruntime-osx-arm64-1.29.0.tgz \
APPLE_STORE_AMD64_ARCHIVE=/absolute/path/onnxruntime-osx-x86_64-1.23.2.tgz \
APPLE_STORE_OUTPUT_DIR=/absolute/path/local-qualification \
make apple-store-package-local
```

The builder compiles arm64 and x86_64 app/broker/service binaries, stages both
pinned native libraries and complete upstream notices, verifies the original libraries against Microsoft's Apple-anchored signing
identity, retains Abseil's exact privacy declaration/license/provenance resource,
signs nested code before the app, and validates exact entitlements, native dependency paths, architecture,
minimum OS, copyright/category metadata and code/resource seals. Final validation
rejects quarantine attributes on the app or any nested item without changing them. ARM requires macOS 14.0 and Intel 13.4.
Deployment flags are explicit cgo cache inputs. Fyne metadata writes happen in a
temporary directory; the repository build number is unchanged. The output includes
`PicFetch.app` and a JSON manifest containing source/diff evidence and file hashes.
These ad-hoc signatures are local test signatures and cannot be submitted.

```sh
python3 scripts/macstorepackage/package.py --verify /path/local-qualification/PicFetch.app
PICFETCH_TEST_MAC_APP_BUNDLE=/path/local-qualification/PicFetch.app \
python3 -m unittest discover -s scripts/macstorepackage -v
python3 scripts/macstorepackage/qualify.py \
  --app /path/local-qualification/PicFetch.app \
  --models /path/to/verified-model-data --result /path/worker-results.json
```

The worker qualifier creates disposable thin copies and substitutes only the
main GUI executable with a Go test driver. Production broker/service/image worker
code remains. It checks HEIC 8/10-bit pixels, ONNX inference, model/app access,
private-container source/cache transfer, cache reuse, retained search and actual
TCP/UDP denial. Models are verified by production checks; they are only added to
the temporary fixture. Intel results on this host are Rosetta evidence, not
physical Intel or minimum-OS testing. Generated private fixture directories are
removed on normal exit. The original local app is never changed.

Explorer and visual search capture URI authority before leaving UI. Production
clients transfer fresh implicit bookmarks while original source scopes are active,
keep acquisitions through worker return, and release receiving scopes when the
worker exits. Exact source files are transferred; no source parent is granted.
The outer app grant permits full signature validation, and internal model/cache
locations carry separate grants. Missing sources/cache grants keep existing
per-item/cache failure behavior. This does not yet qualify stale-grant recovery,
window drops or every OS file handoff.

## Submission preparation records

[Submission draft](submission-draft.md) contains listing copy, review steps,
screenshot plans and the human-input checklist. [Dependency/privacy audit](privacy-dependency-audit.md)
records the remaining Protobuf/ONNX privacy coverage and submission validation
gap. Exact Abseil manifests are retained separately and verified during packaging,
and original Microsoft runtime identities are checked before local re-signing.
These input checks do not clear all Apple submission requirements.

The worker qualifier also supports `--deny-source`. It compiles only its test
parent with a Go overlay omitting source grants, keeps the packaged production
broker/service/helper unchanged, and requires the private source to fail with an
OS permission error while an allowed bundle source still completes inference.
Run it with the same `--app`, `--models` and a separate `--result` output as the
positive qualification. The overlay and diagnostic driver are never shipped.

## Single-image folder navigation

Opening a single image in the Store build offers a native folder permission panel
when sibling discovery needs a wider grant. This shared path covers the Open
dialog, window drops and Open With/Dock delivery. Confirm **Allow Folder Access**
for the containing folder to enable Left/Right browsing. The selected image stays
on screen initially; siblings retain the confirmed directory bookmark. The folder
approval is also saved in app preferences and reused for fresh single-image opens,
including after quitting and relaunching. Saved bookmarks are resolved and checked
before use; moved/stale bookmarks are refreshed. An unavailable or unusable grant
falls back to the permission panel. Earlier test builds did not keep this separate
approval history, so approve each folder once in the new build.
Cancel opens only the selected image. Existing directory grants, folders,
multiple-file selections, saved-collection replay and merge additions do not
request this extra permission. A replaced or cancelled opening discards late
permission results. Native presentation is tracked without blocking shutdown.


## Distribution-signed candidate packaging

`make apple-store-package-signed` consumes a qualified local app and creates a
fresh output directory containing a signed app, `PicFetch.pkg` and provenance
manifest. It does not upload or install anything and does not modify the input
app. This tooling is implemented; a real Store-signed run still needs credentials
and remains unverified. Continue using `apple-store-package-local` for direct E2E
runs: Store distribution signatures are intended for Apple's distribution path.

```sh
APPLE_STORE_APP=/absolute/path/local-output/PicFetch.app \
APPLE_STORE_SIGNED_OUTPUT_DIR=/absolute/path/fresh-store-candidate \
APPLE_STORE_TEAM_ID=YOURTEAMID \
APPLE_STORE_APP_IDENTITY=APPLICATION_CERTIFICATE_SHA1 \
APPLE_STORE_INSTALLER_IDENTITY=INSTALLER_CERTIFICATE_SHA1 \
APPLE_STORE_TESTFLIGHT=1 \
APPLE_STORE_PROFILE=/absolute/path/PicFetch.provisionprofile \
APPLE_STORE_WORKER_PROFILE=/absolute/path/PicFetchWorker.provisionprofile \
make apple-store-package-signed
```

Identity selectors are full uppercase SHA-1 certificate fingerprints from
Keychain's valid identity inventory. The application identity must be Apple
Distribution or 3rd Party Mac Developer Application; the installer identity must
be 3rd Party Mac Developer Installer, both for the chosen Team. Private keys stay
in Keychain; only public certificate bytes are read for fingerprint comparison.
No private-key export or Apple-account login is performed.

This route requires explicit app (`io.github.frathe.picfetch`) and XPC worker
(`io.github.frathe.picfetch.worker`) profiles when TestFlight is requested.
Without TestFlight, these bundles' unrestricted sandbox claims may omit profiles.
Inherited helpers retain only sandbox/inherit and have no profiles. Supplied
profiles are copied as immutable bytes, decoded with `security cms`, and checked
for macOS, exact identifier/prefix, Team, validity, distribution scope and the
selected certificate. These checks are diagnostics; profile plist structure is
not a stable API or authoritative proof of CMS/DER acceptance. Apple's validation
remains required. See [TN3125](https://developer.apple.com/documentation/technotes/tn3125-inside-code-signing-provisioning-profiles)
and [manual distribution signing](https://developer.apple.com/documentation/xcode/creating-distribution-signed-code-for-the-mac/).

Code is signed inside-out and checked against the Apple anchor, selected leaf
certificate, Team and code identifier. The installer uses `productbuild
--component` targeting `/Applications`. `pkgutil` checks its signature and the selected leaf certificate's SHA-256 fingerprint, then
expands it without installation; the sole app payload must match the verified
signed app in bytes and executable permissions. A missing or mismatched input
qualification manifest, existing output directory or failed validation stops
publication. Portable policy tests run in CI; the native installer roundtrip
uses an explicitly unsigned disposable installer and does not prove distribution
signing. The remaining SDK/privacy, real certificate/profile, TestFlight and
App Store submission gates remain open.

## CI candidate signing and manual approval

`.github/workflows/apple-store.yml` prepares signed candidates on demand. It runs
only for `main` in `frathe/picfetch`, pins both checkouts to the dispatch revision,
and requires the existing full CI gate. A credential-free hosted Mac builds the
universal app from pinned runtime archives and runs native packaging guards.
A separate hosted Mac downloads that same-run artifact and signs it only after
the **apple-store-signing** environment is approved. No Apple upload, TestFlight
invitation, App Review submission or automatic release is part of this workflow.
The workflow must reach the default branch before GitHub exposes manual dispatch.

The environment was configured on 2026-09-30 with **frathe** as required reviewer,
a single `main` branch policy, and administrator bypass disabled. Self-review is
allowed so Ronin can approve a run he started, matching the existing Microsoft
signing workflow. Recheck these settings before adding credentials; merely
naming an environment in YAML does not create its approval rules. GitHub documents
[environment protection](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments).

In repository Settings -> Environments -> **apple-store-signing**, add these
**environment secrets** (not repository-wide secrets):

| Secret | Value |
|---|---|
| `APPLE_STORE_APP_P12_BASE64` | Base64 of a password-protected `.p12` containing the Apple Distribution certificate and its private key |
| `APPLE_STORE_APP_P12_PASSWORD` | Password for that application `.p12` |
| `APPLE_STORE_INSTALLER_P12_BASE64` | Base64 of a password-protected `.p12` containing the Mac Installer Distribution certificate and its private key |
| `APPLE_STORE_INSTALLER_P12_PASSWORD` | Password for that installer `.p12` |
| `APPLE_STORE_PROFILE_BASE64` | Base64 of the app's Mac App Store Connect `.provisionprofile` |
| `APPLE_STORE_WORKER_PROFILE_BASE64` | Base64 of the worker's Mac App Store Connect `.provisionprofile` |

Add these **environment variables** in the same environment:

| Variable | Value |
|---|---|
| `APPLE_STORE_TEAM_ID` | The 10-character Apple developer Team ID |
| `APPLE_STORE_APP_IDENTITY` | Full uppercase 40-character SHA-1 of the application certificate |
| `APPLE_STORE_INSTALLER_IDENTITY` | Full uppercase 40-character SHA-1 of the installer certificate |

Export only the intended certificate/private-key pair for each `.p12` from
Keychain Access. For hosted CI, those private keys must be available on the
runner; the local-only signing route keeps them on your Mac instead. GitHub's
[Apple signing guide](https://docs.github.com/en/actions/how-tos/deploy/deploy-to-third-party-platforms/sign-xcode-applications)
describes password-protected certificate export and Base64 secrets. Base64 is an
encoding, not encryption. Keep exports outside the repository and enter passwords
through the secret UI. For example, upload one encoded file without printing it:

```sh
base64 -i /absolute/private/path/application.p12 | \
  gh secret set APPLE_STORE_APP_P12_BASE64 --repo frathe/picfetch --env apple-store-signing
```

The wrapper generates its temporary Keychain password; no extra Keychain-password
secret is needed. It imports only during the signing step, restricts key access
to Apple signing tools, uses only that Keychain for identity lookup, and removes
raw exports before calling the packager. It strips encoded credentials/passwords
from the packager environment. Success, failure and handled cancellation restore
the old search list and remove temporary credentials; `always()` retries cleanup.
Hosted runner destruction covers an uncatchable kill. No private-key files or
credential directory is uploaded or cached.

After merging and configuring the inputs, select Actions -> **Apple Store
candidate** -> Run workflow -> **main**. Review the build summary's source SHA
and artifact hash, then approve the signing environment. Download
`apple-store-signed-<sha>-<attempt>` for `PicFetch.pkg` and its provenance manifest.
Signing refuses an artifact whose recorded source differs from the dispatch SHA
or whose source was dirty. Installer payload checks then verify the actual files.

Credential-free policy/lifecycle tests and workflow linting are available now.
The first approved hosted run with real identities/profiles remains required to
qualify noninteractive Keychain access, profile acceptance and distribution trust.
The SDK/privacy, minimum-OS GUI and Apple submission gates remain unchanged.
