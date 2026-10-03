# Ronin's Mac App Store checklist

Updated 2026-10-01. Work through the numbered steps in order; stop after step 4
and hand the signing inputs back to Pico. Steps 5–7 can then proceed while Pico
qualifies the signed candidate. No upload or release is authorized by this list.

**Already done:** Xcode is installed and licensed; the local universal Store test
app is built; repaired native folder approval is qualified as recorded below.
Saved session/Favorite reopening, renamed-folder Favorite access and fresh
image dragging from that renamed folder also pass. All platform
and race CI jobs and CodeQL passed for `62661a1`. Qodana completed with eight
reviewed unused-function false positives and no actionable findings. You
reauthorized committing and pushing this documentation after your lunch break.

Current test app: `bin/apple-store-clipboard-2026-10-01-01b5a18/PicFetch.app` (build 483).
Its adjacent `TESTING.md` and `manifest.json` record the build. Keep this app for
local E2E; Pico will make the distribution candidate in a separate directory.

## 1. Confirm the Apple developer team

- [ ] Sign in to [Apple Developer](https://developer.apple.com/account/) and
  confirm the intended paid Developer Program team is active.
- [ ] Have the Account Holder review any pending agreements in
  [App Store Connect](https://appstoreconnect.apple.com/). Confirm Account Holder
  or Admin access for the certificate/profile work below.
- [ ] Give Pico the selected **10-character Team ID** and whether the app will
  be published by you personally or an organization. Passwords and recovery
  codes are not needed.

Apple limits distribution-certificate creation to the Account Holder or Admin;
see its [certificate overview](https://developer.apple.com/help/account/create-certificates/certificates-overview).
Pending agreements can prevent creating an app record; see
[adding an app](https://developer.apple.com/help/app-store-connect/create-an-app-record/add-a-new-app).

## 2. Register or confirm both explicit identifiers

In Certificates, Identifiers & Profiles, register these explicit App IDs on the
chosen team, or confirm existing matching entries belong to it:

| Purpose | Bundle identifier | Suggested description |
|---|---|---|
| PicFetch app | `io.github.frathe.picfetch` | PicFetch |
| Embedded worker | `io.github.frathe.picfetch.worker` | PicFetch Worker |

- [ ] Both explicit IDs are available on the correct team.

Use Apple's [App ID registration guide](https://developer.apple.com/help/account/identifiers/register-an-app-id/).
If either identifier is unavailable, tell Pico before choosing another: changing
the app identity affects saved settings and sandbox data. No extra portal
capabilities are requested by the current packaging route.

## 3. Put both distribution signing identities in this Mac's Keychain

Both distribution identities are verified on 2026-10-01. Ronin's Keychain
screenshot shows each certificate with its matching private key. A read-only
check outside the sandbox confirms valid `Apple Distribution` and
`3rd Party Mac Developer Installer` identities for Florian Rathe, team
`57TZ845755`. The earlier sandboxed check returned zero identities; that was
not evidence that certificates were missing. No replacements are needed.

The latest clipboard test app passes `codesign --verify --deep --strict` and is
universal arm64/x86_64, but its actual signature is ad-hoc with no TeamIdentifier.
It is not upload-ready. TestFlight-mode preflight sees Xcode 27.0 and reports
three missing supplied inputs at that invocation: Team ID, app profile and
worker profile. The certificate check now establishes the signing identities'
Team ID above; the two profile paths remain to be supplied and validated. That is
an input check, not proof profiles do not exist elsewhere. No certificate was
created, exported or revoked, and no signing or upload occurred during this check.

- [x] Make an **Apple Distribution** application signing identity available.
- [x] Make a **Mac Installer Distribution** identity available. Keychain normally
  displays this as `3rd Party Mac Developer Installer: ...`.
- [x] Confirm each certificate has its matching private key in Keychain Access.

If the team already has suitable identities, import them securely from their
owner instead of unnecessarily replacing or revoking them. For new certificates,
create the request on the Mac that will retain the private key using Apple's
[CSR guide](https://developer.apple.com/help/account/certificates/create-a-certificate-signing-request/),
then choose the relevant certificate type in the developer portal. Apple's
[certificate overview](https://developer.apple.com/help/account/create-certificates/certificates-overview)
distinguishes the app and installer roles. Pico only needs Keychain access to
sign, not private keys pasted into chat or committed to the repository.

## 4. Download two distribution provisioning profiles

Our TestFlight packaging route requires a separate profile for the app and its
XPC worker. For **each** identifier from step 2, create a **Mac App Store Connect**
profile using the application distribution certificate from step 3. Download
both files. Follow Apple's
[profile guide](https://developer.apple.com/help/account/provisioning-profiles/create-an-app-store-provisioning-profile/).

- [x] App profile downloaded for `io.github.frathe.picfetch`.
- [x] Worker profile downloaded for `io.github.frathe.picfetch.worker`.
- [x] Both absolute file paths supplied; both profiles match the installed
  application distribution identity and team `57TZ845755`.

Verified on 2026-10-01 by decoding the supplied profiles outside the sandbox:
both target macOS, carry their respective exact application identifiers, and
expire on 2027-09-30. Neither lists provisioned devices or enables
`get-task-allow`. Their embedded certificate fingerprints match the verified
Apple Distribution identity. Files remain outside the repository in
`/Users/ronin/Documents/CSR Apple Developers/`, named
`PicFetch_Mac_App_Store.provisionprofile` and
`PicFetch_Worker_Mac_App_Store.provisionprofile`.
TestFlight-mode `make apple-store-preflight` passes outside the sandbox with
these inputs. This verifies preparation inputs, not signed-package acceptance;
no signing or upload occurred during this check.

**Handoff:** Pico can now identify the certificate fingerprints, run preflight,
run `make apple-store-package-signed`, and check nested signatures, profiles,
installer trust and extracted payloads. The exact invocation is in the
[packaging README](README.md). You do not need to assemble that command yourself.
Keychain may ask you to approve signing. This work does not upload the app.

## 4a. Enable CI signing with your approval gate

**Permissions correction on 2026-10-01:** Transporter Verify rejected the first
candidate with error 90255. Extracting its installer reproduced two ONNX dylibs
with mode 0600. Distribution staging now normalizes directories/executables to
0755 and other files to 0644 before signing, without changing the input app.
Validation rejects non-owner-inaccessible directories/files in both the app
and extracted installer. Both regression tests failed before the fix; the
packaging suite now passes (33 tests, 10 optional native tests skipped).
GoLand file inspections reported no findings for both changed Python files.
The corrected signed candidate is
`bin/apple-store-signed-2026-10-01-01b5a18-permissions/`: native packaging,
signatures, profile checks, extracted-payload comparison and readability checks
pass. Ronin reported successful Transporter re-verification on 2026-10-01
after being directed to this corrected candidate. His follow-up screenshot
confirms Transporter's green VERIFIED status for PicFetch 1.1.11 (483) on
2026-10-01 at 15:57, with Deliver still available. The detailed verification
log has not been inspected. Do not deliver the old
candidate. No delivery, installation or App Review submission was performed.
Verification does not resolve the pending privacy declaration or independently
establish the outstanding SDK privacy qualification.

**Local signing completed on 2026-10-01:** With Ronin's authorization to
continue, `make apple-store-package-signed` produced
`bin/apple-store-signed-2026-10-01-01b5a18/` containing the signed app,
`PicFetch.pkg` and provenance manifest. Input was the qualified clipboard
test app recorded above; it was not modified. The command exited successfully
after checking both profiles, nested signatures/entitlements, all-architecture
certificate/team requirements, installer trust and exact extracted-payload
bytes/executable permissions. `productbuild` emitted four `write: Permission
denied` diagnostics but completed; subsequent signature and payload checks
passed. No installation, upload or submission occurred. Apple-side validation,
TestFlight acceptance and remaining privacy/SDK/submission gates remain open.
This local run does not establish hosted-CI signing qualification.

- [ ] After the workflow reaches `main`, review the **apple-store-signing**
  GitHub environment: required reviewer **frathe**, only the `main` branch,
  administrator bypass disabled. Pico configured and checked these settings.
- [ ] Export password-protected application and installer `.p12` files from
  Keychain Access, including each matching private key, outside this repository.
- [ ] Add the six environment secrets and three environment variables listed
  in [CI signing setup](README.md#ci-candidate-signing-and-manual-approval).
  This is the hosted-CI alternative to leaving all keys only on your Mac.
- [ ] Start **Apple Store candidate** on `main`, review its build summary, and
  approve the signing job. Pico can inspect the resulting candidate and logs.

No App Store Connect API key is needed for this signing-only workflow. Upload
credentials and release automation are later work. The first real approved CI
signing run is still unverified; current tests use synthetic credential boundaries.

## 5. Create the App Store Connect app record

- [ ] In App Store Connect, use Apps -> + -> New App, platform **macOS**.
- [ ] Select `io.github.frathe.picfetch`, choose the primary language, and
  confirm **PicFetch** as the available name.
- [ ] Choose your internal SKU; `picfetch-macos` is a suggested value.
- [ ] Tell Pico that the record exists and provide its Apple ID/app link.

Use Apple's [new-app guide](https://developer.apple.com/help/app-store-connect/create-an-app-record/add-a-new-app).
Only the main app needs an App Store Connect listing. Registering the embedded
worker for signing does not mean creating a second store product.

## 6. Finish the remaining hands-on checks

Use disposable copies of images for writing, overwriting and Trash tests.
Record the build, macOS version, architecture, action and actual result for any
failure. The existing local app can catch regressions now; repeat acceptance
checks on the eventual TestFlight-delivered build.

**Folder-approval blocker resolved in build 483 (2026-10-01):** The earlier
build 477 failure left the permission panel open after approval. The repaired
native panel and moved-source handling are documented in the
[regression record](../../finished_refactorings/2026-10-01-folder-access-regression.md).
That exact package passed controlled native Open With, cancellation, grant reuse
and full quit/relaunch checks. Today Ronin additionally confirmed Finder dragging
from a newly created, unapproved folder: the correct access request appeared,
approval completed, and Left/Right browsed siblings around the dropped image.
Ronin then fully quit and reopened the same client, dropped another image from
that newly approved folder, and confirmed no new access prompt and working
Left/Right navigation. Saved session/Favorite reopening also passed Ronin's
checks below, including Favorite reopening after a folder rename without a new
access prompt. Ronin also confirmed fresh image dragging from that renamed
folder preserves sibling browsing without another prompt. The broader manual
acceptance suite has not passed.

- [x] Drop one image onto the window and open one through Finder's Open With:
  after folder approval, Left/Right reaches its siblings. Canceling a new folder
  request still leaves the selected image usable.
- [x] Reopen a saved Favorite after quitting and relaunching: on 2026-10-01
  Ronin confirmed its images open and Left/Right navigation works in build 483.
- [x] Check Restore Last Session after quitting and relaunching. On 2026-10-01
  Ronin retested and confirmed the expected behavior in build 483, resolving the
  earlier missing-offer observation. No application defect was established.
- [x] Grant a disposable folder, save a Favorite, quit, rename the folder
  in Finder, relaunch, and reopen the Favorite. On 2026-10-01 Ronin confirmed
  images open and browse without a new access prompt in build 483. This verifies
  a rename, not a move to a different parent or volume.
- [x] Drop one image from the renamed folder: on 2026-10-01 Ronin confirmed
  Left/Right sibling browsing without another access request in build 483.
- [x] Check external-volume responsiveness while disconnected and recovery
  after reconnection. Functional recovery passed; error presentation is deferred.
  Connected-drive baseline on 2026-10-01: Ronin placed test images on an external
  drive, dropped one into build 483, browsed Left/Right and saved a Favorite.
  After quit/eject/disconnect/relaunch, opening that Favorite kept the app
  responsive but produced many missing-file error toasts (Ronin reported
  NSCocoaErrorDomain code 4). Ronin explicitly accepted and deferred this
  usability issue on 2026-10-01: it is not a release blocker. He reports acceptable
  responsiveness and no problematic memory use; this was not instrumented.
  The follow-up is in todos.md's Deferred section. Ronin reconnected the drive with
  PicFetch still running, reopened the Favorite and confirmed images and
  Left/Right navigation recovered.
- [x] Exercise Save/export to a chosen destination, overwrite a disposable copy,
  Trash, clipboard, Reveal in Finder and wallpaper. Check the actual output/effect.
  PNG export to a new filename passed on 2026-10-01: Ronin reopened the output
  from Finder and confirmed the correct image. The previously unapproved
  destination folder requested separate sibling-browsing access when opening
  the image; this is expected. Export overwrite also passed: Ronin exported a
  different image over that disposable PNG, confirmed replacement, and reopened
  it from Finder to verify the replacement image. Trash also passed: PicFetch's
  action removed that disposable PNG from its folder and Ronin verified it in
  Finder's Trash. Reveal in Finder passed: Finder selected the correct image
  file in its folder. Historical clipboard failure on 2026-10-01: Ronin's screenshot shows
  `could not copy the image: open /var/folders/.../T/picfetch_clip_1885754130.png: operation not permitted`.
  The denied path is PicFetch's temporary clipboard PNG, not the source image.
  This earlier failure occurred before the Preview paste check. The backend
  now writes PNG data directly to AppKit; current acceptance is recorded below. Save Changes passed: Ronin
  rotated a disposable image, saved it, and reopened it in Preview to confirm
  the rotation persisted. Set as Wallpaper also passed: Ronin confirmed the
  chosen test image appeared on the desktop. Clipboard also passes in the
  refreshed 01b5a18 app, including Preview paste after PicFetch exits.
- [x] Qualify Copy Image in the sandboxed Mac app: source 01b5a18, 1.1.11/build
  483, macOS 27.0.1 (26A434), Apple Silicon. Native menu Copy Image and Cmd+C open the correct 16x12
  color pattern and distinct 8x6 pink PNG in Preview, including after PicFetch
  fully exits before either copied image is read. Preview's inspector confirms
  both dimensions. Ronin approved the test folder and Preview inspection.
  Current app: `bin/apple-store-clipboard-2026-10-01-01b5a18/PicFetch.app`.
  See [native evidence](../../finished_refactorings/2026-10-01-store-clipboard-image.md).
- [x] Open HEIC in the GUI: on 2026-10-01 Ronin confirmed correct display and
  navigation to another image and back without errors in build 483.
- [ ] Exercise Similarity Explorer/visual search, cancel/reopen, and maps in the
  GUI. Confirm ordinary browsing still works when model setup is declined.
  Explorer opening passed on 2026-10-01: Ronin reported the map opens, and the
  current screen showed grouped images with 441 ready, 0 failed, 441 reused.
  Ronin also confirmed opening a group, viewing one image and returning to
  Explorer works normally. This establishes cached-map opening and navigation,
  not fresh inference. Ronin reported
  accepting a model download the previous day. Visual search passed: Ronin
  confirmed Find more like this produces results that open successfully.
  Explorer close/reopen also passed: Ronin returned to the viewer, reopened
  Explorer and confirmed the map returns and remains responsive. Leaving active
  analysis and reopening also passed: Ronin started an unanalyzed folder,
  chose Back to Viewer during analysis, confirmed responsive browsing and
  reopened Explorer normally. Model-setup decline remains unchecked because
  the model is already installed; qualify separately in a clean setup without
  resetting Ronin's current installation. Fresh analysis passed: Ronin loaded a previously unanalyzed folder
  of about 900 images, observed analysis progress and confirmed it completed
  with a usable map. Ronin confirmed automatic map updates were disabled;
  waiting for completion before the first render was expected behavior.
  Location Map display passed: Ronin loaded photos
  with GPS data and confirmed map tiles load and markers appear at expected
  locations. Map-to-image navigation also passed: Ronin tested a group via the
  grid and a single image opening directly, and confirmed return to the map
  works for both routes.
- [ ] Arrange access to physical Intel hardware and the declared minimum systems:
  macOS **13.4 Intel** and **14.0 Apple Silicon**. Tell Pico what machines/testers
  are available so a specific qualification run can be prepared.
  On 2026-10-01 Ronin confirmed no physical Intel Mac is available. Physical
  Intel qualification remains an unverified coverage gap; Rosetta and CI do
  not replace it. Minimum-OS GUI qualification also remains open.

Single-file browsing expands into the folder. Multiple selected/dropped files
intentionally stay within that selection. Native folder consent now passes the
checks above; saved-collection and moved-folder checks remain separate.
Rosetta tests and current Intel CI do not establish minimum-OS GUI support.

## 7. Review the listing and owner decisions

**Pricing update, 2026-10-01:** Saved the approved free-price schedule in App Store
Connect and read back Current Price with zero-price tiers. Availability remains
174 available and one excluded (France). Content Rights is still unanswered;
the live form asks whether third-party content is present and necessary rights
are held. No rights declaration was saved. Existing live subtitle is
"a small, fast image viewer", primary category Photo & Video, secondary category
Graphics & Design; these differ from parts of the local draft and were not changed.

Use the [submission draft](submission-draft.md) and
[privacy/network inventory](privacy-dependency-audit.md) as prepared inputs.

- [x] Confirm description, category, primary language, price and territories.
  Primary category Photography and primary language English (US) approved by Ronin
  on 2026-10-01; Ronin also approved a free download and availability in all
  territories Apple permits for this app. Ronin approved the description in the
  submission draft on the same date.
  These are recorded decisions, not changes applied in App Store Connect.
- [ ] Supply legal/copyright owner, review contact and any required seller/trader
  details; complete the current age-rating questionnaire.
  Copyright owner confirmed as Florian Rathe on 2026-10-01. Read-only Chrome
  inspection shows the review contact name entered and populated phone/email
  fields; the browser masks their values. Ronin confirmed both the review phone
  number and email are correct on 2026-10-01; review contact verification is complete.
  Review notes saved and read back on 2026-10-01, with Save disabled afterward;
  the exact text is retained in the submission draft. Notes cover local-file access,
  reviewer test steps, optional model setup and maps, and Store updates. Recheck
  against the final signed artifact before submission. No app review was submitted.
  App Information shows an existing non-trader declaration;
  Ronin confirmed on 2026-10-01 that PicFetch is a private, noncommercial
  open-source project, with no plans to monetize it or promote paid services.
  This supports retaining his existing non-trader self-declaration under Apple's
  guidance; no account setting was changed and no independent legal determination
  was made. Reassess if the project's commercial or professional purpose changes.
- [x] Complete the age-rating questionnaire. Saved on 2026-10-01 after Ronin
  confirmed no app-supplied adult, violent, frightening, drug-related or profane
  material, including examples/artwork. Feature answers reflect the local viewer:
  no sharing/social/chat/ads/unrestricted browser, parental controls or age checks;
  no medical/wellness or gambling/contest features. No higher-age override or
  Made for Kids designation. Readback shows 4+ in 172 countries or regions with
  regional exceptions, and global 4+ with regional exceptions on older systems.
- [ ] Choose the public support and privacy URLs and contact details. Pico will
  check that their final content is accessible without signing in.
  Ronin approved the GitHub issue tracker and repository privacy policy on
  2026-10-01. Both approved URLs in the submission draft returned HTTP 200 to
  unauthenticated checks with the expected content. Contact details remain open.
  At Ronin's explicit request, changed the version form's Support URL from
  GitHub Discussions to GitHub Issues and saved on 2026-10-01. Readback shows
  the approved issue-tracker URL and Save disabled. Saved the approved repository
  Privacy Policy URL on App Privacy on 2026-10-01 and verified the page readback.
  Ronin confirmed no additional analytics, crash reporting or developer-run
  collection and approved proceeding with third-party disclosure on 2026-10-01.
  Saved and read back an unpublished candidate list: Precise Location, Coarse
  Location, Product Interaction and Other Diagnostic Data. These are provisional,
  not a completed privacy declaration. Location Map permits tile zoom 19, which
  motivates reviewing precise map-area disclosure rather than assuming coarse
  only; this does not establish transmission of the user's current GPS position.
  Provider policies document IP/request logging, but exact category applicability,
  purposes and identity linkage still need validation. Per-category setup remains
  incomplete, Publish is disabled, and no privacy label has been published.
- [ ] Review the privacy and encryption/export answers against the technical
  inventory. Local image analysis alone does not settle all network disclosures.
- [ ] Select or approve nonprivate, licensed/synthetic example images and review
  screenshots captured from the final viewer. Pico can prepare the screenshots
  when desktop access is available; generated mockups are not the deliverable.

## 8. Approve TestFlight upload once technical gates are resolved

**Map fix follow-up, 2026-10-01:** The working tree now has HTTP-aware EXIF tile
caching, an owned renderer that does not bypass expiry through Fyne-X's global
cache, and direct OSM license links on both map surfaces. Focused race tests,
renderer expiry/lifecycle checks, vet and native build pass; GoLand found no new
actionable issues. The full native-amd64 Docker gate cannot run on this ARM daemon.
The already delivered build 483 does not contain these changes. A new signed
candidate and native/TestFlight map retest are still needed. No rights declaration
was saved and no replacement build was uploaded. See the map-cache plan.

**Processing/export update, 2026-10-01:** The uploaded build appeared in TestFlight.
Saved the standard-encryption-outside/in-addition-to-Apple-OS answer and No for
France after Ronin explicitly postponed France. TestFlight readback shows Ready
to Submit, with the Missing Compliance warning cleared. Ronin clicked the final
availability confirmation; readback shows 174 countries or regions available on
release and France Not Available. Future territories remain enabled. This revises
the earlier all-territories decision, not the free-price decision (live pricing
still needs setup). No App Review submission occurred.

Apple Developer Support confirmed receipt of the authorized encryption question,
case **102982575995**. Asked about a Go TLS/public-download exemption, exact French
documentation if required, and separate TestFlight restrictions for France.
Keep France postponed pending clarification; email reply remains outstanding.
The separate privacy declaration remains pending. Next: internal TestFlight
installation and smoke tests, plus remaining listing/submission requirements.

**Delivery update, 2026-10-01:** Ronin explicitly chose to deliver the verified,
permissions-corrected candidate himself in Transporter. His screenshot confirms
PicFetch 1.1.11 (483) was delivered at 16:00 and is processing. This establishes
upload success, not processing completion, TestFlight readiness, SDK privacy
qualification, App Review submission or public release. The privacy declaration
and remaining submission requirements stay open. Earlier no-upload statements
describe the state at their respective checks, before this authorized delivery.

- [ ] Review Pico's signed-candidate report and authorize an upload to App Store
  Connect/TestFlight, including who should be invited to test.
- [ ] Install the delivered build and confirm the final E2E results.
  Internal TestFlight setup completed on 2026-10-01: created Internal Release
  Testing with automatic future-build distribution disabled, added 1.1.11 (483),
  and invited Florian Rathe's existing Account Holder account. Readback confirms
  one build Ready to Test and one tester Invited. No external testers or public
  links were added. Ronin installed the delivered TestFlight build and reported
  successful launch, comparison and Location Map checks on 2026-10-01. A captured
  macOS warning asked permission to access data from previously opened PicFetch
  versions; launch subsequently succeeded. Ronin also confirmed Copy Image into
  Preview's New from Clipboard displays the correct image in the delivered build.
  Ronin confirmed a previously unauthorized folder prompted for access and,
  after approval, Left/Right browsed sibling images in the delivered build.
  Ronin then fully quit, reopened through TestFlight and confirmed sibling
  browsing in the same folder without another permission prompt. The focused
  delivered-build smoke checks pass; this is not a complete E2E rerun.
  No App Review submission occurred.

Before asking for this approval, Pico must finish the Protobuf/ONNX privacy and
SDK qualification, verify the actual signed installer, and report any remaining
platform gaps. CI passing does not settle those gates. These are technical tasks
owned by Pico; you are not being asked to invent dependency declarations.

## 9. Decide on App Review and release

- [ ] Approve the final listing, screenshots, review notes and questionnaires.
- [ ] Separately authorize submission for App Review.
- [x] Choose release timing and authorize release after approval.
  On 2026-10-01 Ronin explicitly approved automatic release after Apple approval.
  The version form already has automatic release selected; no website change
  was needed. This authorizes release timing, not an upload or App Review
  submission now; the technical gates and separate submission approval remain.

Creating certificates, signing a candidate, uploading a beta, submitting for
review and releasing are distinct steps. None has been silently selected by
this checklist.
