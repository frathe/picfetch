# Mac App Store submission draft

Prepared 2026-09-30. Local preparation only: nothing uploaded or published.
The current ad-hoc bundle is a qualification artifact, not a submission build.
Release readiness still depends on the technical and human gates below.

## Listing copy

Name: **PicFetch**

Subtitle: **Explore your photo collection**

Suggested category: Photography. Secondary category, price, territories, age
rating, copyright owner and seller/trader details require Ronin's decision.

Description draft:

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

Keyword draft: `photo,image,viewer,slideshow,compare,metadata,EXIF,favorites,similarity`

Do not advertise minimum-OS or physical Intel qualification until those tests
pass. The generated bundle declares macOS 14.0 on Apple Silicon and 13.4 on
Intel, based on the pinned runtime requirements. Rosetta tests are separate
from a physical Intel run.

Candidate support URL: https://github.com/frathe/picfetch/issues

Candidate privacy URL: https://github.com/frathe/picfetch/blob/main/PRIVACY.md

Both URLs still require successful unauthenticated content checks after the
updated policy reaches the public branch. The research browser could not fetch
these pages on 2026-09-30; that is not evidence that the links are public or
broken. No contact email or legal identity has been invented.

## Review notes draft

No login, subscription or reviewer account is required.

1. Use Open to select a folder of local images. The app uses the macOS file
   chooser and stores security-scoped bookmarks to reopen granted files. A file
   selection authorizes that file; select a folder to browse its siblings.
2. Browse images, switch to the grid, compare pictures and save a Favorite.
   Relaunch and reopen the Favorite to exercise saved access. Permission failure
   can be addressed by explicitly selecting the source again with Open.
3. Similarity Explorer offers an optional download of pinned model data and a
   processor configuration from Hugging Face. Declining leaves ordinary viewing
   available. Executable ONNX code is bundled and updated with the app. Analysis
   uses an embedded XPC service and inherited image helper without network
   permission; operation-scoped grants let it read selected inputs and local
   model/cache files.
4. HEIC decoding uses Apple's system decoder in that isolated worker boundary.
5. Maps contact OpenStreetMap only after opening Location Map or expanding the
   EXIF Location section. Tile requests disclose the viewed map area and normal
   network request information, but contain no pictures or filenames.
6. Store builds do not check GitHub for app updates or install replacement app
   code. Help includes the privacy policy and third-party notices offline.

Replace these notes with observed instructions from the final signed artifact.
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

## Technical gates before account work

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
- Complete the Store certificate signing and productbuild route; validate the
  profile when supplied and inspect the final installer payload. Ad-hoc tests
  cannot establish distribution trust or App Store acceptance.

## Human inputs last

1. Signed Git commits and pushing the preparation branch were reauthorized by
   Ronin after the overnight work; submission/release still require a separate decision.
2. Supply the developer Team ID and Store app/installer signing identities with
   their private keys. TestFlight also needs a matching provisioning profile.
3. Create or select the App Store Connect record for `io.github.frathe.picfetch`.
4. Confirm legal contact, seller/trader status, age rating, pricing, availability,
   privacy questionnaire and encryption/export answers from the prepared facts.
5. Review the final screenshots/listing, authorize upload and TestFlight testing,
   then decide whether to submit for review and release.
