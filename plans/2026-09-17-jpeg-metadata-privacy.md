# Complete JPEG metadata removal

Status: implementation and first hosted CI verified; human acceptance pending.
Current commit-bound review/check evidence: [PR #37](https://github.com/frathe/picfetch/pull/37).
Route: Deep (imaging, window, qualification).
Deliver the accepted `.scratch/jpeg-metadata-privacy/spec.md` through its selected
public mutation/inspection and real EXIF-window seams. Existing user documentation
edits are retained. The user subsequently authorized committing and pushing this
feature branch, creating a PR, and starting a subagent for its GitHub Codex review
loop. No merge/release or unrelated save/export changes.

## Contract and qualification

- Imaging owns `InspectJPEGMetadata(ctx, data) JPEGMetadataInspection`, whose
  `State` is `JPEGMetadataClean`, `JPEGMetadataRemovable` or
  `JPEGMetadataUnsupported`, and whose `Err` explains refusals. Existing mutation
  signatures, compatibility predicate and committed-result semantics remain.
- A private complete marker/scan policy reconstructs qualified APP0/APP14,
  removes previews, all other descriptive markers and trailers, validates primary
  structure, and applies the same policy to orientation-corrected output.
- Qualify 8-bit Huffman baseline and progressive gray/three-component JPEG,
  including sequential separate-component scans. Other processes/color models
  are refused. Upright coded data and decoded pixels must remain identical.
- Qualify ICC v2/v4 RGB matrix/TRC and gray/TRC input/display profiles with XYZ
  PCS. Retain only explicitly validated transform tags; neutralize descriptive
  header fields and required descriptions. Unknown transform fields, LUT profiles,
  ambiguous chunk assembly and model mismatches are refused. Common RGB/gray
  fixtures must pass independent LittleCMS transforms for all rendering intents.
- Production dependencies unchanged. Synthetic fixtures use installed
  libjpeg-turbo 3.2.0 and LittleCMS 2.19.1, development tools only. Their output is
  generated test data from our own pixels/profile parameters. Record exact tool
  versions, recipe, license review and color results with the corpus before done.

## Tasks

1. T0: imaging baseline privacy/inspection/refusal vertical slices.
   Files: new `jpegprivacy.go`, existing `save.go`, `jpegexif.go`, tests.
   Test: public operation output bytes, untouched refusal, clean/idempotent result.
   Verify: parent AC1/AC4/AC5 targeted subcases. Budget: no spawns, two reviews.
2. T0: extend scan qualification and orientation fidelity.
   Files: private parser and public operation tests plus synthetic corpus.
   Depends: 1. Verify: AC1/AC2/AC4/AC5. Budget: no spawns, two reviews.
3. T0: qualified ICC assembly/normalization and combined profile/process matrix.
   Files: new `jpegicc.go`, tests/corpus. Depends: 1; independent of 2.
   Verify: AC3 plus AC1/AC2/AC4/AC5 and independent checker.
   Budget: one fixture-tool task, two reviews; no full suite until final gate.
4. T0: real window availability/status, no-op and confirmation/documentation.
   Files: exifwin metadata/action/tests, manuals, translations.
   Depends: shared contract in 1. Verify: AC6/AC7, localization/manual guards.
   Budget: no spawns, two reviews.
5. T0: integrated review, GoLand inspections, Qodana exclusions, architecture,
   todos/evidence and `make verify` once. Parent AC1-AC8 commands are authoritative;
   first confirm named tests exist. Record any unavailable gate as unverified.

Graph: 1 -> (2, 3, 4) -> 5. Lead implements/reviews sequentially; the independent
fixture reference tool can run alongside parser work.

## Delegation and cost

Read-only scout completed fixture/tool and UI-harness reconnaissance. One fixture
tool task may reuse it: G1 bounded tool contract, G2 self-check command, G3 two
owned files only, G4 isolated ctypes/tool context, G5 lead holds parser context.
The tool needs API comprehension; it is not a regex transform. No spec, review,
production implementation or post-review fix is delegated. Total budget two
tasks, one concurrent agent, one final complete suite attempt.

## Evidence

- Recon: existing transaction/UI queue paths are reusable; no runtime dependency
  is needed. Installed JPEG and LittleCMS tools permit independent qualification.
- Red/green observed for preview removal, progressive/separate-component scans,
  ICC normalization, orientation/profile model consistency, empty-EXIF action
  availability, distinct statuses, no-op success suppression and confirmation
  width. Refusal regressions exposed conflicting color declarations and malformed
  ICC version encoding; both fixed and green.
- No-op commit, retained profile description and skipped orientation guards were
  deliberately broken individually: each failed at its intended public assertion;
  source restored after each check.
- All named AC1-AC6 tests run and pass. Existing specified imaging/EXIF-window
  mutation, confirmation, cancellation/navigation/close and retry regressions pass.
- Independent LittleCMS checks of all eight RGB/gray v2/v4 input/display profiles:
  zero XYZ difference for all four intents (343 RGB / 257 gray samples). Exact
  numeric-tag and coded-image/pixel assertions also pass. See the qualification
  record for support limits, corpus and tool/dependency provenance.
- GoLand inspected all changed Go files and the Python reference tool with errors
  and warnings enabled: no findings. The last modified ICC/test files are rechecked
  at the final gate. Confirmation screenshot inspected at 420px; no clipping.
- Native Linux/amd64 Docker platform check passed with Docker socket access.
- `make verify` passed for the first complete candidate: native Linux/amd64
  Docker partitions non-ui/ui-1/ui-2/ui-3 all passed, with 2,100 non-UI and 686
  root-UI top-level passes. Artifacts: `.scratch/race-runs/20260917T104259Z-RN7dr9`.
  See the final JFIF follow-up below for the revision boundary.
- Final JFIF follow-up: `make verify-build`, the complete specified focused
  imaging/EXIF-window race regression command, and GoLand reinspection of all
  three subsequently changed Go files passed. All six named AC tests are listed
  by the test inventory command.
- Hosted CI on `ef18e79868657d7e3e4f3d1f764aa924b30fccd2` passed all four Linux
  race partitions, validation, Windows tests and native macOS amd64/arm64 guards.
  Both CodeQL analyses passed, with no open PR alerts. Native desktop/manual
  runtime qualification remains unverified; no platform-specific implementation
  was introduced. See the hosted review record below for subsequent heads.

## Acceptance map

For AC1-AC5 use `go test -tags no_emoji,nodynamic -count=1 ./internal/imaging`
with `-run` set respectively to the anchored names below; for AC6 use
`./internal/ui/exifwin`. Confirm inventory first with `-list '^TestJPEGMetadataRemoval'`.

| Criterion | Named test / command |
| --- | --- |
| AC1 Whole-file metadata/secondary-media removal | `^TestJPEGMetadataRemovalPrivacy$` |
| AC2 Exact upright data/pixels and orientation exception | `^TestJPEGMetadataRemovalFidelity$` |
| AC3 Qualified color transforms; uncertain profiles refused | `^TestJPEGMetadataRemovalProfiles$` plus independent checker |
| AC4 Refusal preserves original without commit | `^TestJPEGMetadataRemovalRefusal$` |
| AC5 Inspection states and clean/idempotent no-op | `^TestJPEGMetadataRemovalInspection$` |
| AC6 Real action/status/confirmation/notification | `^TestJPEGMetadataRemovalUI$` |
| AC7 Existing transaction and window lifetimes | `go test -tags no_emoji,nodynamic -count=1 ./internal/imaging ./internal/ui/exifwin -run '^(TestFileMutation|TestFileTransaction|TestMetadataRemoval|TestMetadataResultsApplyOnceOnUIAndRejectQueuedOldData)'` |
| AC8 Repository gate | `make verify`, GoLand changed-file inspections |

Qualification: [support table, fixtures and color evidence](../docs/jpeg-metadata-removal-qualification.md).

Actual implementation cost: one agent spawned and reused for two independent
fixture tasks; all production implementation, review and fixes by the lead.
One complete local gate planned. The separately requested hosted review-loop
agent follows the user's later explicit delegation and commit authorization.

### Final review follow-up

The full Docker gate began on the first complete implementation candidate. Lead
review then found that the inherited orientation re-encode dropped JFIF pixel
aspect/density. A new public regression failed for that exact reason; the re-encode
now retains the validated JFIF header and swaps its density axes for orientations
5-8. The complete focused acceptance/lifetime set passed after the fix. Formatting,
vet/build, focused race tests and GoLand passed for this final change; the
requested GitHub review workflow runs the complete suite on the actual PR commit.
Do not interpret the earlier full Docker run as proof of this later source revision.

### Hosted review record

The user explicitly delegated the GitHub Codex review loop. The review-loop
agent owns assessment, fixes, focused verification and review dispositions for
[PR #37](https://github.com/frathe/picfetch/pull/37), without merging or releasing.
Final commit-bound review and check evidence is recorded on that PR so recording
the result does not create another unreviewed commit.

- Initial head `ef18e79`: no existing unresolved threads. CI run `35212934602`
  and CodeQL run `35212934599` passed. Qodana run `35212934587` passed its gate,
  but the root `qodana.sarif.json` contained two post-suppression findings.
- `GoUnusedGlobalVariable` on `ErrJPEGMetadataNotJPEG` is a false positive:
  `internal/ui/exifwin/metadata.go` compares the sentinel in both refresh and
  error presentation. Retained the API and added a documented declaration-local
  suppression, matching the existing cross-package suppression convention.
- `GoVarAndConstTypeMayBeOmitted` on the orientation result is valid. Removed the
  redundant `image.Image` annotation; the function already returns that interface.
- These edits change no behavior. Existing JPEG removal, file mutation/transaction
  and EXIF-window delivery regressions passed with `-race -count=1`; focused vet,
  formatting and GoLand inspections with warnings enabled also passed.
- A fresh Codex code/security review and Qodana/CodeQL/CI evaluation of the pushed
  cleanup head are required before the review loop completes. Review requests
  are posted only after prior reviews finish, avoiding duplicate queued work.

Follow-up review findings through `e5ebc50`:

- Confirmed ambiguous Adobe/JFIF versus RGB component declarations and misplaced
  JFIF. Public inspection/mutation regressions failed before the fixes and pass
  with refusal/no commit/unchanged source. Valid fixture metadata now follows the
  required leading JFIF rather than displacing it.
- Confirmed uninterruptible orientation/encoding. Orientation now checks between
  rows and directly retains the gray/RGB model. JPEG encoding unwinds only a
  private cancellation signal at buffered writes, since the standard encoder
  otherwise keeps processing after writer errors. Public cancellation regressions
  failed before each fix; encoding cancellation is injected through a test-owned
  context at the stable standard-library encoder boundary, without production
  hooks or scheduler timing assumptions.
- Confirmed alignment-dependent scan polling. Scan traversal now advances through
  every byte with a progress threshold, including marker runs. Existing complete
  scan/process acceptance tests and cancellation regressions cover the refactor.
- Focused imaging/EXIF-window race acceptance and transaction/lifetime regressions,
  vet, formatting and GoLand inspections of all four changed Go files pass.
- Qodana on `e5ebc50` showed that the original in-block suppression was ineffective
  and raised a documentation-format notice. The sentinel now has a properly named
  comment and a suppression on its own declaration; the next SARIF verifies it.
- GitHub's separate "Code scanning AI findings" service failed before analysis on
  both pushed heads with CAPI HTTP 400, "The requested model is not supported."
  This is distinct from the passing CodeQL jobs and the Codex security review;
  its unavailable result must remain explicit in final PR evidence.
- `f9ce1d7` Qodana confirmed the original two findings are gone and reported only
  `GoMaybeNil` on the orientation RGBA destination. This is a false positive:
  allocation initializes exactly one of the gray/RGBA pointers, and the gray
  branch handles its non-nil case. Added an explained statement-local suppression;
  existing gray/RGB orientation and cancellation regressions retain the behavior.

Follow-up review findings on `f9ce1d7`:

- SPIFF APP8 is unsupported interpretation data. A public refusal regression
  observed the previous committed rewrite, then passed after explicit refusal.
- Preserved full-decode integrity validation and added a 256 MiB removal-specific
  working-memory budget. Header-only admission precedes encoded copies and full
  decoding, using bounded int64 arithmetic and MCU/component sampling to include
  progressive coefficient arrays. Rotation and re-encoded output have separate
  reservations within that budget; source reads honor its 60 MiB encoded cap.
  A valid generated 50-megapixel JPEG observed the old full decode/clean result;
  the public regression now proves resource refusal before `jpeg.Decode`, no
  commit and unchanged source. The constant-color generator allocates no source
  pixel plane. Existing supported JPEG/profile/orientation fixtures still pass.
- Sampling accounting uses maxima across all components and reserves full planes
  for Go 1.27's flexible sampling path and possible RGB conversion. Output decode
  padding is independently rounded for the encoder's 4:2:0 layout. Final focused
  imaging/window race regressions, vet, manual/translation guards, formatting and
  warnings-inclusive GoLand inspections passed before the grouped fix commit.
- Root independently supplied the localized memory-error mapping and real-window
  confirmation regression: red with the generic message, green with the explicit
  memory refusal, unchanged bytes and no success notification. Focused window
  race tests, translation guards and GoLand passed for that slice.
- Both manuals now say replacement may change filesystem attributes/timestamps;
  neither preservation nor removal is guaranteed. Existing manual guards passed.
- The Codex security review of `f9ce1d7` completed at 11:35:59 UTC without added
  actionable findings. These follow-up changes still require a fresh code and
  security review on the next pushed head, plus its CI and post-suppression SARIF.

Compatibility follow-up after `cb9fc1c`:

- The new admission reader reused the viewing/export walker, which stops at legal
  marker-fill bytes before a frame. A public inspection/mutation regression
  exposed the unintended refusal. A removal-specific, cancellable frame reader
  now preserves the complete parser's support; the regression passes and proves
  exact primary-image preservation after removing a comment.
- Focused imaging/window race regressions, imaging vet, formatting and GoLand
  inspections of all three changed code files pass.
- `cb9fc1c` passed CI run 35218977951 and CodeQL run 35218977905, with no open
  CodeQL alerts. Qodana run 35218977861's post-suppression SARIF has zero results.
  Its security review completed without actionable findings; the compatibility
  follow-up still requires fresh hosted review/CI on its own pushed head.
- The fresh code review confirmed missing cancellation during admission's second
  header walk. The removal-specific reader fixes it by polling at every marker
  and during legal fill runs, while retaining the pre-allocation budget checks.
- The review also found EXIF-only Adobe RGB interpretation loss. Public tests
  observed destructive success before the fix in both TIFF byte orders; they
  now prove non-sRGB/uncalibrated or conflicting interoperability declarations
  are refused unchanged, while default sRGB/R98 removes successfully. Six
  explicit EXIF transform/colorimetry tags have their own red/green refusal
  coverage. The qualification record documents this conservative scope and its
  primary DCF source; conflicting ICC/EXIF precedence is not assumed.

Follow-up review findings on `6d8b7f6`:

- All normal CI/CodeQL jobs passed; CodeQL alerts were empty and post-suppression
  Qodana SARIF had zero results. The fresh security review completed without
  actionable findings. The fresh code review identified three remaining cases.
- Admission and full parsing now share one bounded marker-fill reader, which
  polls cancellation at most 4096 fill bytes apart. This removes the unchecked
  second fill pass while retaining the public marker-fill and cancellation
  regressions; no large stress input or timing-dependent test was needed.
- A public clean/no-op regression exposed canonicalization of legal fill before
  EOI. Output now retains the exact original EOI marker span, both when clean and
  when other metadata is removed; the observed red regression is green.
- A qualified non-sRGB ICC derivative combined with explicit EXIF sRGB exposed
  the inverse declaration conflict. Explicit EXIF ColorSpace or InteropIndex
  together with any ICC is now conservatively refused, independent of segment
  order. The public refusal regression observed committed rewrites before the
  fix and now proves unchanged bytes/no commit. Orientation-only EXIF plus ICC
  remains covered by the existing supported fidelity matrix.
- An independent follow-up found that output-header validation still used a
  non-cancellable reader. It now uses the operation context like admission/full
  decoding. A deterministic public-operation regression failed with a committed
  rewrite before the fix, then passed with cancellation/no commit/unchanged
  bytes at the standard-library output DecodeConfig boundary.
- Final focused imaging/window race regressions, imaging vet, formatting and
  warnings-inclusive GoLand inspections of all three changed Go files pass.

Follow-up review finding on `a6a75f6`:

- All normal hosted checks and security review passed, with zero post-suppression
  Qodana results and no open CodeQL alerts. Fresh code review found that rebuilding
  retained JFIF still removed legal marker fill. Public no-op regressions also
  confirmed the same class for Adobe and already-normalized ICC markers.
- Retained JFIF/Adobe preserve their original marker prefix. ICC normalization
  reports exact assembled-profile equality; an already-sanitized profile retains
  its original chunks, fill and scan positions even when other metadata is
  removed. Arbitrary ICC internals are still qualified and sanitized as before.
- ICC insertion uses the recorded leading-JFIF end, including fill. Orientation
  receives the already-qualified normalized ICC segments directly instead of
  passing through the tolerant export walker. Public filled-JFIF/profile checks
  verify upright and oriented output remains qualified and retains its transform.
- The new marker no-op regressions observed committed rewrites before the fix
  and are now green for clean files and files with removable comments.
- Focused imaging/window race regressions, imaging vet, formatting and
  warnings-inclusive GoLand inspections of all four changed Go files pass.

Follow-up review findings on `c3c20d2`:

- Normal CI, CodeQL and security review passed; CodeQL alerts and post-suppression
  Qodana results were empty. Fresh code review found an undeclared component-color
  path and a broken ADR link to the local, untracked issue-tracker specification.
- Three-component JPEGs without JFIF/Adobe now require qualified `1/2/3` YCbCr or
  literal `RGB` identifiers. Public upright/oriented tests observed unsupported
  `ABC` inputs being rewritten before the fix, then passed with refusal/no commit/
  unchanged source; both qualified identifier sets continue to remove successfully.
- The ADR now links to permanent tracked requirements/qualification documentation
  instead of the ignored issue-tracker file. The qualification table records the
  component-identifier support limit.
- Focused imaging/window race regressions (including a rejected identifier
  permutation), imaging vet, formatting and GoLand inspections of both changed
  Go files pass. Both replacement ADR links resolve to git-tracked files.
