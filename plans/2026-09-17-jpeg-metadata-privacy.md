# Complete JPEG metadata removal

Status: local implementation verified; hosted PR review/CI and human acceptance pending. Route: Deep (imaging, window, qualification).
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
- Hosted PR review/CI remains pending. Native Windows/macOS runtime execution is
  unverified; no platform-specific implementation was introduced.

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
