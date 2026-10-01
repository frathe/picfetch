# Store Copy Image blocker

Route: Standard; lead owns diagnosis, implementation and review; zero delegates.

## Problem and red loop

Ronin's first open todo: Copy Image fails in Store build 483. The current compiled
source 3453f1b bundle reproduces the exact error on a generated 16x12 RGB JPEG:
`open /tmp/picfetch_clip_3214527429.png: operation not permitted`.
Native Actions -> Copy image is the original end-to-end loop, driven through Sky.
Red log: /private/tmp/picfetch-clipboard-blocker-before.log. PNG encoding reached
the macOS backend before the denied temporary write; source access is not the
observed failure. Hypotheses: denied temp directory first; AppleScript child
authority second; upstream source/encode failure ruled out by that boundary.

## Acceptance and verification

1. Native macOS image copying passes PNG bytes directly to AppKit, with no temp
   file or subprocess. Regression exercises the operation through a local writer
   while temp storage is unavailable; native publication errors propagate and
   empty input cannot overwrite a clipboard.
   `go test -race -tags no_emoji,nodynamic ./internal/clipboard`
2. Whole-image/region/batch admission and captured pixels remain the existing
   UI producer path; affected root regressions pass.
   `go test -race -tags no_emoji,nodynamic -run 'TestCopyImage|TestCopySelection|TestClipboard' ./internal/ui`
3. Actual Store sandbox app copies the generated image and Preview File -> New
   from Clipboard opens the correct 16x12 colored image. Repeat image copy and
   retain errors/empty-input behavior. Qualification uses exact packaged source
   and does not read an old user clipboard after a failed publication.
4. Make verify-build, changed-file GoLand weak-inclusive inspections, fresh
   GitHub code/security-focused reviews, configured post-suppression Qodana,
   CodeQL and full hosted platform/race CI pass. Do not repeat the broad local
   race suite. Source-only fix commit/push and review-loop authorization carries
   forward from Ronin's workflow. Rebuild the local Store package for testing.

## Implementation scope

Use existing clipboard.go dispatch plus darwin.go/other.go platform pair.
The private operation receives its writer as an argument; no mutable package
seam is added. AppKit creates a fresh NSPasteboardItem/NSData inside one call
and eagerly writes PNG data; original Go memory is not retained. File-list
copy is unchanged; Windows/Linux keep their existing paths. Update obsolete
comments, package map, tests and the exact first todo without staging unrelated
App Store metadata/acceptance edits. Archive this record at completion.

No dependencies, runtime assets, entitlements or permissions are added. AppKit
is already linked and covered by the existing native platform closure/notices.
No blanket temp-directory or sandbox-policy workaround. TestFlight/distribution
signing, other open Store acceptance work and clipboard interoperability beyond
the qualified Preview path remain separate. Native worker policy remains intact.

## Implementation and local evidence

- Red: `TestCopyImageDarwin_DoesNotRequireTempStorage` failed on the old
  implementation with `no such file or directory` and publisher not called.
  The same operation now publishes identical bytes directly; failure propagation
  and empty-input preservation regressions pass without touching the desktop.
- Green: clipboard package race tests and root image/selection/shortcut/error
  regressions with `no_emoji,nodynamic,appleappstore` passed. `make verify-build`
  passed (format, configuration/assets/notices, vet and build).
- GoLand inspection API, current Project Default, `errorsOnly:false`, on all five
  changed Go files: native bridge, platform stub and UI glue clear. The temp-file
  resource warning was a false positive: `writeTempPNGFile` owns and closes the
  handle on every path, with existing failure coverage. Added a narrow source
  suppression for IDE parity; reinspection is clear. Existing Linux test duplicate
  at clipboard_test.go:114 is intentional and already exactly excluded in Qodana.
  This is IDE fallback evidence, not an IDE-local Qodana or CI-equivalent scan.
- Native candidate `bin/apple-store-clipboard-candidate-2026-10-01/PicFetch.app`,
  1.1.11 build 483: approved 16x12 test JPEG loaded; Actions -> Copy image now
  completes without the old denied-temp-file toast or clipboard error in the log.
  Candidate manifest records base a46076c plus the dirty source snapshot; it is
  not yet the final committed-source package.
- Preview inspection was rejected by automatic approval review because other
  open documents could be exposed. Permission requested; Preview paste and
  pixel-dimension qualification remain pending, not passed.

The publisher uses Apple's documented [NSPasteboardItem data publication](https://developer.apple.com/documentation/appkit/nspasteboarditem)
through [NSPasteboard writeObjects](https://developer.apple.com/documentation/appkit/nspasteboard).

## Remaining gates

Commit/push the reviewed source fix; complete Preview qualification if approved,
then refresh the package from the committed source. Fresh latest-commit code and
security-focused reviews, hosted CI, CodeQL and post-suppression Qodana assessment
remain pending. Archive this record only after required work is complete.
