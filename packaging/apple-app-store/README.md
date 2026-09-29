# Mac App Store preparation

The Apple channel is experimental. No submission package or signed sandbox
qualification of the complete app has been completed. The native worker
fixture is qualified on this Apple Silicon host. Retain all PicFetch features; do not use
temporary filesystem exceptions or weaken the analysis worker's network denial
to pass a packaging check.

The [active plan](../../plans/2026-09-29-apple-app-store.md) owns implementation
and evidence. [Apple requirements](../../docs/apple-app-store-requirements-2026-09-29.md)
records current primary sources and distinguishes native macOS requirements
from mobile-platform privacy rules.

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
step; signed Mach-O validation is separate work. The same library's SHA-256
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
APPLE_STORE_PROFILE=/absolute/path/PicFetch.provisionprofile \
make apple-store-preflight
```

It uses `/Applications/Xcode.app/Contents/Developer` without changing the system
developer-directory selection. Override `APPLE_STORE_DEVELOPER_DIR` for another
full Xcode installation. It checks for Store application and installer signing
identities with private keys and confirms that a profile file was supplied.
It does not install credentials, access an Apple account, upload a package or
certify the profile or app. Supplying a profile does not establish its signature,
expiry, registered identifier or entitlement compatibility.

Required inputs:

- Active Apple Developer Program team and a registered explicit bundle ID
  `io.github.frathe.picfetch` available to that team.
- Apple Distribution / Mac App Store application signing certificate with its
  private key, plus a Mac Installer Distribution identity.
- Mac App Store Connect provisioning profile for that identifier and signing
  certificate; exact helpers and their profiles depend on the sandbox design.
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

No `package-mac-store` target is advertised until these runtime and signing
contracts are implemented and exercised. A successful prerequisite check cannot
substitute for signed sandbox or submission evidence.
