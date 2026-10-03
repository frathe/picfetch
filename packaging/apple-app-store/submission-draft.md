# Mac App Store submission draft

Prepared 2026-09-30. Local preparation only: nothing uploaded or published.
The current ad-hoc bundle is a qualification artifact, not a submission build.
Release readiness still depends on the technical and human gates below.

## Listing copy

Name: **PicFetch**

Subtitle: **Explore your photo collection**

Approved by Ronin and saved in App Store Connect on 2026-10-01; subtitle
readback after page reload confirms persistence.

Primary category: **Photography**, approved by Ronin on 2026-10-01.
Primary language: **English (US)**, approved by Ronin on 2026-10-01.
Price: **Free**, approved by Ronin on 2026-10-01.
Saved in App Store Connect on 2026-10-01; Current Price readback shows zero-price
tiers, with United States as the base. Pricing covers 175 territories independently
of availability, which remains 174 available and one excluded (France).
Availability: **All permitted territories except France for the initial release**.
Ronin approved postponing France on 2026-10-01 pending encryption clarification.
App Store Connect readback confirms 174 countries or regions available on release
and France Not Available after Ronin clicked Confirm. Future territories remain
enabled by the availability setup.
Copyright owner: **Florian Rathe**, confirmed by Ronin on 2026-10-01.
Release timing: **Automatically release after Apple approval**, explicitly
approved by Ronin on 2026-10-01. The existing App Store Connect setting matches.
Upload and App Review submission remain separate pending approvals and gates.
Secondary category, age rating and seller/trader details require Ronin's decision.

Description approved by Ronin on 2026-10-01 (technical release gates still apply):

> PicFetch helps you browse and explore image collections on your Mac. Open
> individual pictures or folders, switch between an image and a thumbnail grid,
> compare pictures, run a slideshow, and organize reusable collections with
> Favorites.
>
> Explore image metadata and discover where geotagged photos were taken with
> Location Map. Maps use OpenStreetMap when you choose to open them.
>
> Similarity Explorer and visual search analyze images locally on your Mac.
> Their optional first-use model download is about 372 MB. After setup,
> analysis works offline; your pictures are not uploaded for analysis.
>
> PicFetch has no account requirement, advertising or analytics. You choose
> which files and folders it can access. Updates arrive through the Mac App
> Store.

Approved keywords: `photo,image,viewer,slideshow,compare,metadata,EXIF,favorites,similarity,mosaic,wallpaper`
Saved and read back in App Store Connect on 2026-10-01, with Save disabled.

Do not advertise minimum-OS or physical Intel qualification until those tests
pass. The generated bundle declares macOS 14.0 on Apple Silicon and 13.4 on
Intel, based on the pinned runtime requirements. Rosetta tests are separate
from a physical Intel run.

Approved support URL: https://github.com/frathe/picfetch/issues

Approved privacy URL: https://github.com/frathe/picfetch/blob/main/PRIVACY.md

Ronin approved both GitHub URLs on 2026-10-01. Unauthenticated HTTPS checks
returned HTTP 200 for both pages; page titles confirmed the repository issue
tracker and privacy file, and the privacy response included the policy text.
The research browser still returned a cache miss, so verification used direct
unauthenticated requests. Review contact details remain to be supplied.

## Review notes saved in App Store Connect

Saved and verified on 2026-10-01. Recheck against the final signed artifact before
submission; saving metadata did not submit the app for review.

No login, subscription, purchase or reviewer account is required.

Getting started:
- Open local images or drag an image from Finder into PicFetch. When requested, approve access to its containing folder to browse sibling images with Left and Right.
- Test the grid, comparison and slideshow using a folder containing several images. Save a Favorite, quit and reopen PicFetch, then load the Favorite to test persistent file access.
- Save/export, Copy Image, Reveal in Finder and Trash are available. Please use disposable test images for save/overwrite and Trash operations.

Optional local image analysis:
- Open Similarity Explorer to set up local similarity analysis. First use offers a download of model data and processor configuration from Hugging Face, approximately 372 MB; internet access is needed for this download. Ordinary image viewing does not require the model.
- The executable analysis runtime is bundled with the app. Images are analyzed locally, not uploaded. After model setup, analysis works offline.
- Use a small folder for a quick initial analysis. Larger collections can take longer. After processing, open an image group, view an image and return to Explorer, or use Find more like this.

Location Map:
- Use photos containing GPS metadata. Opening Location Map or expanding the EXIF Location section requests map tiles from OpenStreetMap.
- Requests disclose the viewed map area and normal network information, but do not upload pictures, filenames or raw EXIF records. A map area need not represent the user's current location.

The app has no advertising, analytics integration or user-account requirement. Help includes the privacy policy and third-party notices. Mac App Store builds receive application updates through the Mac App Store, not PicFetch's GitHub updater.

Privacy policy: https://github.com/frathe/picfetch/blob/main/PRIVACY.md

### Internal qualification reminder

Do not submit the disposable worker-qualification app; its main executable is a
command-line test driver, not the viewer.

## Screenshot plan

Capture the real, final sandboxed viewer with a small synthetic or clearly
licensed image collection: main image and filmstrip, grid, comparison,
Similarity Explorer, and Location Map. Show actual features and readable UI;
exclude private filenames, user images and personal coordinates. Capture both
light and dark appearance if useful. App Store screenshots remain outstanding
because the computer-use connection did not respond. Generated mockups are not
substitutes for these screenshots. Confirm current upload dimensions in App
Store Connect when the app record exists.

## Privacy and encryption facts

See [the dependency/privacy audit](privacy-dependency-audit.md). Do not select
“No Data Collected” solely because image analysis is local: optional map and
model requests reach third-party servers, whose retention policies matter.

The app uses Go HTTPS/TLS for map and model downloads. The distribution closure
also contains hashing and signature-verification code. No custom cryptographic
algorithm, end-user encrypted messaging feature, VPN or user-file encryption
feature was added for the Store port. This is a technical inventory, not an
export classification. Do not set `ITSAppUsesNonExemptEncryption` until the
export questionnaire has been assessed for the actual shipped closure. Apple
requires the developer to determine the applicable documentation; see
[export compliance](https://developer.apple.com/help/app-store-connect/manage-app-information/overview-of-export-compliance/).

## Technical gates before submission

**Export questionnaire update, 2026-10-01:** Saved standard encryption implemented
outside/in addition to Apple OS encryption, with No for France following Ronin's
explicit postponement decision. TestFlight now shows Ready to Submit rather than
Missing Compliance. This is not App Review submission or a blanket legal exemption.
Apple Developer Support accepted case **102982575995**, asking whether this Go TLS
use qualifies for an exemption, what French document/process is required otherwise,
and whether postponed France availability needs separate TestFlight restrictions.
Reply pending by email; keep French testing/distribution postponed pending clarity.
The separate App Privacy case remains unresolved.

- Resolve the remaining Protobuf/ONNX privacy manifest coverage and final SDK
  validation. Abseil declarations and pre-sign Microsoft identity checks are now
  implemented; see the audit for source/version and notice delivery evidence.
- Qualify the final signed GUI: picker, Finder/Dock/window drops, restart,
  Favorites, moved and unavailable sources, Save/export, overwrite, Trash,
  clipboard, Finder reveal and wallpaper. Scope renewal has unit/race coverage;
  moved-folder relaunch and final UI behavior remain unverified.
- Negative source-grant tests pass on ARM and Intel under Rosetta. Finish full
  application cancellation/reopen testing and qualify real Intel hardware and
  both declared minimum OS versions.
- Run the full native Linux/amd64 CI suite, CodeQL and fresh post-suppression
  Qodana analysis. ARM Docker emulation cannot replace the isolation gate.
- Run the implemented `make apple-store-package-signed` route with real Apple
  identities/profiles and qualify its candidate. Portable policy and native
  unsigned-installer payload tests pass; they cannot establish distribution
  trust or App Store acceptance.

## Human inputs last

Follow [Ronin's ordered checklist](ronin-checklist.md) for account setup,
signing inputs, remaining E2E checks and submission decisions. Account setup can
proceed while Pico resolves the technical gates above.

1. Signed commit `62661a1` was pushed before Ronin's lunch break. He
   reauthorized committing and pushing the current documentation after returning;
   submission/release still require a separate decision.
2. Supply the developer Team ID and Store app/installer signing identities with
   their private keys in Keychain. This TestFlight route also needs separate
   profiles for `io.github.frathe.picfetch` and `io.github.frathe.picfetch.worker`;
   see the packaging README for the exact command and input names.
3. Create or select the App Store Connect record for `io.github.frathe.picfetch`.
4. Confirm legal contact, seller/trader status, age rating, pricing, availability,
   privacy questionnaire and encryption/export answers from the prepared facts.
5. Review the final screenshots/listing, authorize upload and TestFlight testing,
   then decide whether to submit for review and release.
