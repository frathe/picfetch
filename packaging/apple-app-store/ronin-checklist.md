# Ronin's Mac App Store checklist

Updated 2026-09-30. Work through the numbered steps in order; stop after step 4
and hand the signing inputs back to Pico. Steps 5–7 can then proceed while Pico
qualifies the signed candidate. No upload or release is authorized by this list.

**Already done:** Xcode is installed and licensed; the local universal Store test
app is built; you confirmed folder approvals survive quit/relaunch. All platform
and race CI jobs and CodeQL passed for `62661a1`. Qodana completed with eight
reviewed unused-function false positives and no actionable findings. You
reauthorized committing and pushing this documentation after your lunch break.

Current test app: `bin/apple-store-e2e-persistent-folders/PicFetch.app`.
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

The read-only Keychain check on 2026-09-30 found **zero valid Store application
identities and zero valid Store installer identities with private keys**.

- [ ] Make an **Apple Distribution** application signing identity available.
- [ ] Make a **Mac Installer Distribution** identity available. Keychain normally
  displays this as `3rd Party Mac Developer Installer: ...`.
- [ ] Confirm each certificate has its matching private key in Keychain Access.

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

- [ ] App profile downloaded for `io.github.frathe.picfetch`.
- [ ] Worker profile downloaded for `io.github.frathe.picfetch.worker`.
- [ ] Tell Pico the two absolute file paths and confirm the intended Team ID.

**Handoff:** Pico can now identify the certificate fingerprints, run preflight,
run `make apple-store-package-signed`, and check nested signatures, profiles,
installer trust and extracted payloads. The exact invocation is in the
[packaging README](README.md). You do not need to assemble that command yourself.
Keychain may ask you to approve signing. This work does not upload the app.

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

- [ ] Drop one image onto the window and open one through Finder's Open With:
  after folder approval, Left/Right reaches its siblings. Canceling a new folder
  request still leaves the selected image usable.
- [ ] Check a saved session and a saved Favorite after quitting and relaunching.
- [ ] Grant a disposable folder, save a Favorite, quit, move/rename the folder
  in Finder, relaunch, and reopen the Favorite. Report whether images resolve
  and whether another grant is requested; then try a single image from the new
  location. Automated bookmark tests pass; this native behavior is still open.
- [ ] If available, repeat with an external volume disconnected, then
  reconnected. The app should remain responsive and recover after access returns.
- [ ] Exercise Save/export to a chosen destination, overwrite a disposable copy,
  Trash, clipboard, Reveal in Finder and wallpaper. Check the actual output/effect.
- [ ] Exercise HEIC, Similarity Explorer/visual search, cancel/reopen, and maps
  in the GUI. Confirm ordinary browsing still works when model setup is declined.
- [ ] Arrange access to physical Intel hardware and the declared minimum systems:
  macOS **13.4 Intel** and **14.0 Apple Silicon**. Tell Pico what machines/testers
  are available so a specific qualification run can be prepared.

Single-file browsing expands into the folder. Multiple selected/dropped files
intentionally stay within that selection. You already confirmed this UX and
persistent folder approvals; neither is an outstanding redesign request.
Rosetta tests and current Intel CI do not establish minimum-OS GUI support.

## 7. Review the listing and owner decisions

Use the [submission draft](submission-draft.md) and
[privacy/network inventory](privacy-dependency-audit.md) as prepared inputs.

- [ ] Confirm description, category, primary language, price and territories.
- [ ] Supply legal/copyright owner, review contact and any required seller/trader
  details; complete the current age-rating questionnaire.
- [ ] Choose the public support and privacy URLs and contact details. Pico will
  check that their final content is accessible without signing in.
- [ ] Review the privacy and encryption/export answers against the technical
  inventory. Local image analysis alone does not settle all network disclosures.
- [ ] Select or approve nonprivate, licensed/synthetic example images and review
  screenshots captured from the final viewer. Pico can prepare the screenshots
  when desktop access is available; generated mockups are not the deliverable.

## 8. Approve TestFlight upload once technical gates are resolved

- [ ] Review Pico's signed-candidate report and authorize an upload to App Store
  Connect/TestFlight, including who should be invited to test.
- [ ] Install the delivered build and confirm the final E2E results.

Before asking for this approval, Pico must finish the Protobuf/ONNX privacy and
SDK qualification, verify the actual signed installer, and report any remaining
platform gaps. CI passing does not settle those gates. These are technical tasks
owned by Pico; you are not being asked to invent dependency declarations.

## 9. Decide on App Review and release

- [ ] Approve the final listing, screenshots, review notes and questionnaires.
- [ ] Separately authorize submission for App Review.
- [ ] Choose release timing and authorize release after approval.

Creating certificates, signing a candidate, uploading a beta, submitting for
review and releasing are distinct steps. None has been silently selected by
this checklist.
