# Mac App Store requirements: Apple-source research

Research date: 2026-09-29. Scope: a native macOS app submitted to the Mac App Store. Sources are Apple documentation, App Store Connect help, and the developer agreement. This document does not assess PicFetch's implementation or certify submission readiness. Unchecked items require evidence from the final signed store artifact, runtime tests, or App Store Connect. Checklist scenarios below are proposed verification work, not claims that Apple prescribes those exact tests.

## Distribution and runtime constraints

[App Review Guidelines](https://developer.apple.com/app-store/review/guidelines/) §2.4.5 requires sandboxing; an Xcode-based, self-contained single-app installation; no installation in shared locations; no root escalation/setuid; consent for startup or surviving processes; no launch licensing/copy protection; and store-delivered updates. It prohibits downloading additional code or resources that add functionality or substantially change the reviewed app. All localizations belong in the bundle. §2.5.1 requires public APIs and current-OS compatibility. §2.5.2 also restricts downloaded executable functionality. §5.1.1 requires an accessible in-app privacy-policy link covering collection, uses, third parties, retention/deletion and consent withdrawal.

Verification proposals:

- [ ] Exercise all startup, quit, installation and update paths in the store build; record policy and process outcomes.
- [ ] Inventory downloadable assets and explain their reviewed purpose, distinguishing content from code and functionality-changing resources.
- [ ] Inventory APIs used by every shipped component, including native dependencies.
- [ ] Check all bundled locales and the privacy-policy entry point.

The [developer agreement](https://developer.apple.com/support/terms/apple-developer-program-license-agreement/) §3.3.1 further limits macOS store apps to documented APIs supplied with the default macOS installation, Xcode/Mac SDK, or Swift Playground; it excludes deprecated technologies such as Java. An API's presence on a development machine is insufficient evidence that it is supported for store distribution.

## App Sandbox, files and durable access

[App Sandbox](https://developer.apple.com/documentation/security/app-sandbox) is mandatory for Mac App Store distribution. Apple's [entitlement reference](https://developer.apple.com/library/archive/documentation/Miscellaneous/Reference/EntitlementKeyReference/Chapters/EnablingAppSandbox.html) defines `com.apple.security.files.user-selected.read-write` for user-selected file read/write access. Request only capabilities justified by actual operations; read-only access cannot support modifying originals or writing outputs.

[Accessing files from the macOS App Sandbox](https://developer.apple.com/documentation/security/accessing-files-from-the-macos-app-sandbox) explains that open/save panels grant access to selected URLs; selected directories include descendants. Protected or otherwise inaccessible items can still fail. Stored security-scoped bookmarks preserve access across relaunch. Resolve with security scope, refresh stale bookmark data, call `startAccessingSecurityScopedResource()` before use, and balance with `stopAccessingSecurityScopedResource()` after use. User-selected file entitlements do not authorize executing programs in arbitrary selected locations.

[Bookmark entitlements](https://developer.apple.com/documentation/professional-video-applications/enabling-security-scoped-bookmark-and-url-access) distinguish app-scoped and document-scoped persistence. [The access API reference](https://developer.apple.com/documentation/foundation/nsurl/startaccessingsecurityscopedresource()) warns that leaked scope resources can prevent further file access until relaunch.

Verification proposals:

- [ ] Inspect final entitlements for sandbox, required user-selected access and bookmark scope.
- [ ] Test native selection, Finder/Dock opening and drag/drop; record which URL authority each entry path receives.
- [ ] On a fresh account, exercise nested folders, external volumes and protected locations; failures must remain understandable.
- [ ] Close and relaunch after saving a collection/session; verify authority comes from resolved bookmarks rather than persisted path strings.
- [ ] Test moved files/folders, stale bookmarks, missing volumes and rejected scope acquisition.
- [ ] Exercise export, metadata edits and trash operations under their actual granted scope; audit overlapping worker scope lifetimes and balanced releases.

The last two scenarios require implementation evidence. This research establishes no blanket entitlement that grants access to every file referenced by a collection.

## Helpers and native dependencies

Apple's [helper embedding guide](https://developer.apple.com/documentation/xcode/embedding-a-helper-tool-in-a-sandboxed-app) expressly supports externally built command-line helpers. Its example embeds them in `Contents/MacOS`, signs them, and uses `com.apple.security.app-sandbox` plus `com.apple.security.inherit`. It warns against additional helper entitlements and incompatible `get-task-allow`. Hardened Runtime is recommended there, but explicitly not required for App Store distribution.

The [inheritance entitlement reference](https://developer.apple.com/library/archive/documentation/Miscellaneous/Reference/EntitlementKeyReference/Chapters/EnablingAppSandbox.html) says `posix_spawn`/`NSTask` inheritance carries static parent rights, not later Powerbox grants. It documents passing data or a bookmark to authorize files selected after launch. Main applications must not enable `inherit`. Other launch mechanisms need separate evaluation.

[Technical Note TN2206](https://developer.apple.com/library/archive/technotes/tn2206/) describes nested code placement and signing: sign nested components before enclosing bundles and place code in recognized bundle locations. A native library being bundled does not exempt its code or dependencies from signing and API requirements.

Verification proposals:

- [ ] Enumerate every executable, dynamic library, framework and executable runtime in the submitted bundle; record architecture, minimum OS, provenance and dependent-library paths.
- [ ] Confirm dependencies resolve to bundled components or supported OS libraries on a clean Mac without developer tooling.
- [ ] Inspect signatures/entitlements for app and each helper independently; verify nested code and outer bundle after packaging.
- [ ] Test each child-process route with a newly selected external file, cancellation and app quit; demonstrate the authority passed to the child.
- [ ] Establish whether same-binary worker modes require distinct signed embedded helpers: the standard helper recipe alone does not prove that re-executing the main app is compatible.

## Signing, package and submission

[Certificates overview](https://developer.apple.com/help/account/certificates/certificates-overview) distinguishes store app/installer certificates from Developer ID certificates. Apple's helper guide shows Apple Distribution for the app and tool; certificate selection must match the chosen distribution workflow. [Provisioning profiles](https://developer.apple.com/help/account/provisioning-profiles/create-an-app-store-provisioning-profile) require an app record with an explicit App ID and a Mac App Store Connect distribution profile; Xcode can manage profiles automatically.

[Packaging Mac software](https://developer.apple.com/documentation/xcode/packaging-mac-software-for-distribution) specifies a signed `.pkg` for store submission and Mac Installer Distribution signing, whose identity is named `3rd Party Mac Developer Installer`. Upload using Transporter or Apple's documented tooling. [Upload builds](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds) explains that bundle ID/version associate the uploaded artifact with its record and build string identifies the build. [Notarization](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution) is a separate direct-distribution process; store submission includes equivalent checks.

Verification proposals:

- [ ] Confirm valid developer membership, app record, explicit identifier, store profile and certificates; archive their identifiers without secrets.
- [ ] Inspect the package's extracted app, nested signatures, profile and entitlements; validate the installer signature and ensure no post-sign mutations.
- [ ] Record Transporter validation/upload logs, processing results, artifact checksum, version and build.
- [ ] Install the actual store/TestFlight distribution and exercise fresh installation, upgrade and another user account.

[Upcoming requirements](https://developer.apple.com/news/upcoming-requirements/) requires removing `com.apple.quarantine` from all macOS app files before upload. It also requires updated age-rating answers and verified EU trader status where applicable. Its April 2026 Xcode 26 SDK rule lists iOS, iPadOS, tvOS, visionOS and watchOS, not macOS. The upload-help page currently has inconsistent broad Xcode-14 guidance and an older macOS table; do not infer a precise macOS minimum from the mobile-platform rule. Recheck live requirements and validate with the current release tooling before submission.

- [ ] Record Xcode/SDK/deployment-target versions and successful current upload validation.
- [ ] Check the final bundle for quarantine attributes, complete age-rating answers, and resolve EU distribution/trader status.

## Privacy manifests and labels

[Privacy manifest files](https://developer.apple.com/documentation/bundleresources/privacy-manifest-files) explicitly says data-collection information applies on **all platforms**. Its required-reason API information applies to **iOS, iPadOS, tvOS, visionOS and watchOS**; macOS is absent. Do not import the iOS required-reason API obligation into a native macOS target solely because the same API exists there.

[Third-party SDK requirements](https://developer.apple.com/support/third-party-SDK-requirements/) requires manifests for listed SDKs in new apps, or updates adding listed SDKs, including repackaged SDKs and any version. Binary dependencies also require signatures. Its wording is broadly App Store based, so this research does not establish a macOS exemption for that separate requirement. Check the complete native/transitive closure against the live list and retain applicable vendor manifests/signatures. Unlisted dependencies still need accurate privacy assessment.

[App privacy details](https://developer.apple.com/app-store/app-privacy-details/) applies across Apple platforms. It defines collection in terms of off-device transmission retained beyond servicing a real-time request; processing confined to the device is not collected for these labels. Third-party practices count. [Manage app privacy](https://developer.apple.com/help/app-store-connect/manage-app-information/manage-app-privacy) requires accurate App Store Connect answers and a privacy-policy URL for macOS, including an explicit no-collection answer when supported by evidence.

Verification proposals:

- [ ] Inventory app and dependency network traffic, payloads, recipients, server retention, logs and tracking; include optional paths and third-party tile/image services.
- [ ] Match manifests, published privacy policy and store answers to that inventory; do not equate local photo processing with collection.
- [ ] Check listed/repackaged SDKs and preserve required manifests and binary signatures in the actual bundle.
- [ ] Keep a native-macOS applicability record; treat upload validator feedback as evidence requiring investigation rather than assuming mobile API rules.

## Encryption export determination

[Export compliance overview](https://developer.apple.com/help/app-store-connect/manage-app-information/overview-of-export-compliance) requires a determination for apps incorporating or accessing encryption, including OS cryptography. [Encryption documentation](https://developer.apple.com/help/app-store-connect/reference/app-information/export-compliance-documentation-for-encryption) distinguishes OS-only encryption (no documentation), standard algorithms supplied outside the OS (French declaration when distributing in France), and proprietary algorithms (CCATS and applicable French declaration). [The submission workflow](https://developer.apple.com/help/app-store-connect/manage-app-information/determine-and-upload-app-encryption-documentation) uses a questionnaire and allows a supported exemption to be recorded in `Info.plist`.

Verification proposals:

- [ ] Inventory TLS, signature verification, cryptographic libraries and other crypto implementations throughout the shipped dependency closure.
- [ ] Record the implementation provider and questionnaire rationale; obtain any required documents before review.
- [ ] Verify the final property-list declaration matches the determination. A bundled TLS implementation cannot be classified as Apple-OS-only merely because it makes HTTPS requests.

This is an Apple submission summary, not a completed legal classification for the app; the implementation and intended countries remain unassessed.

## Listing and review metadata

[App information](https://developer.apple.com/help/app-store-connect/reference/app-information/app-information/) specifies identity, name, categories and privacy link. [Platform version information](https://developer.apple.com/help/app-store-connect/reference/app-information/platform-version-information) requires screenshots, description, keywords, support URL and copyright. The support URL must reach actual contact details. Review information includes a reachable contact, useful testing notes and a non-expiring demo login when login is required. Subsequent releases need release notes. [Prepare for distribution](https://help.apple.com/xcode/mac/current/en.lproj/dev91fe7130a.html) also calls for macOS category and copyright in the bundle's property list.

Verification proposals:

- [ ] Prepare honest screenshots, localized descriptions/keywords, categories, icon, copyright, rating, availability and price.
- [ ] Open support/privacy URLs as an unauthenticated visitor; verify contact and policy content.
- [ ] Provide review notes covering file authorization, helper/dependency behavior and optional networking; include test assets and accounts as applicable.
- [ ] Confirm listing capabilities against the actual store artifact on every advertised architecture and minimum supported OS.

## Remaining uncertainty and evidence boundaries

Apple's current platform-specific privacy page is more precise than generic announcements about approved API reasons. SDK-list requirements and upload-tool version guidance still need live artifact validation. Archived entitlement and signing references are identified above and supported by the current helper guide where available. No sandbox tests, dependency inventory, signing validation, developer-account checks or submission were performed for this research. A compliant design and successful upload are useful evidence, but final App Review approval remains Apple's decision.
