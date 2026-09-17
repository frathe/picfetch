# Restore ordinary JPEG metadata removal

Route: Standard bugfix with expanded documentation/test surface. Lead owns design,
implementation and review. One read-only history Scout (budget 1, actual 1) ran
alongside the reproduction; no implementation/review delegation.

## Evidence and decision

PR #37 rejects even the existing sRGB ICC fixture combined with sRGB EXIF.
`go test -tags no_emoji,nodynamic ./internal/imaging -run
'TestJPEGMetadataRemovalProfiles/sRGB_EXIF_with_sRGB_ICC' -count=1` failed with
`JPEG process or color interpretation is not qualified for metadata removal`.
Its corpus lacks the combinations emitted by cameras/editors. A 256 MiB estimate
also reserves RGB conversion for all color JPEGs, even ordinary YCbCr input.

Rebuild a minimal EXIF block from validated rendering values, never source IFD
bytes, offsets, padding or descriptive tags. Preserve orientation, chroma siting
and enumerated EXIF color declarations so their interpretation with a sanitized
ICC profile stays unchanged. Preserve encoded image data for all orientations;
remove the operation's orientation re-encode. Unknown numerical color-transform
extensions remain refused in this change. The optional user preference question
offered this approach as the recommended default; no answer received before work.

No new dependencies, runtimes or model assets. Existing shipped dependency and
notice obligations remain unchanged. No permission to commit/push was requested.

## Tasks and acceptance

1. Lead: regression matrix and minimal EXIF reconstruction in imaging. All eight
   orientations, centered/co-sited chroma, sRGB/uncalibrated/Adobe RGB declarations
   and ICC combinations must preserve exact compressed image bytes and rendering
   values; personal tags/previews/trailers disappear; repeated removal is a no-op.
   Malformed/duplicate rendering values and existing structural failures remain
   refused without a write. Verify with `go test -tags no_emoji,nodynamic
   ./internal/imaging -run 'Test(JPEGMetadataRemoval|StripJPEGMetadata|CanStripJPEGMetadata)'`.
2. Lead: resource admission for the unchanged decoding path. Retain 256 MiB and
   60 MiB hard limits; account actual ordinary YCbCr allocation and keep worst-case
   allocation for flexible sampling/RGB. Prove ordinary photo admission and
   excessive decoded-memory refusal before decoding in the same tests.
3. Lead: real EXIF-window action/commit/clean-state regression, revised confirmation
   and English/German manuals/catalogues, qualification record and ADR. Verify
   `go test -tags no_emoji,nodynamic ./internal/ui/exifwin -run TestJPEGMetadataRemoval`;
   run translation/manual checks, GoLand on every changed Go file and `make verify`.

Task order: 1 -> 2 -> 3 -> final gate. One review per task, plus final review;
one complete suite at handoff. Existing public APIs, UI workers/queues and
serialized atomic mutations remain intact. Native Windows/macOS runtime behavior
is not established by Linux verification. User-specific files remain unverified
until an example or rejection message is supplied.

## Completion evidence

- The original sRGB EXIF/ICC rejection and rotated/co-sited camera combination
  were observed failing before the implementation; both now pass. The 288-case
  camera matrix also checks exact retained orientation/chroma/color values,
  exact compressed bytes and repeat no-op behavior.
- Restoring the old memory admission rejected the modeled 6000x4000 progressive
  10 MiB photo with a 375,220,256-byte estimate; the corrected admission passes.
  Deliberately copying the source EXIF payload made the privacy guard fail.
  Both negative experiments restored the production implementation immediately.
- Review found that normalized ICC insertion could reorder EXIF/ICC declarations.
  A new regression failed on that order change; normalization now starts at the
  original first ICC segment's position. Its numerical transform stays exact.
- Focused imaging, EXIF-window, mutation/lifetime, translation and manual guards
  pass. GoLand inspected all eight changed Go files, including weak warnings;
  no findings. Changed files were re-inspected after review fixes.
- Independent check: copied the existing 1920x2560 `train.jpg` reference into
  `/tmp/picfetch-jpeg-compatibility`, added Make/Model/Artist/GPS, Orientation 6,
  co-sited chroma, sRGB and the existing v4 ICC using ExifTool, then called the
  public mutation. ExifTool output retained only Orientation, YCbCrPositioning
  and ColorSpace in EXIF, with neutral ICC description. Strict `djpeg` decodes
  before/after were byte-identical (SHA-256
  `d36c39976cfe836f713b3735242e449acc8cc82d0d266c5d88d7582ccaaa1c8a`).
  Temporary harness source was removed; user files were not modified.
- The revised confirmation was rendered through the existing UI evidence hook
  and visually inspected at 420x420: wrapped text fits, both choices visible,
  Cancel remains selected. Artifact: `/tmp/picfetch-jpeg-compatibility/jpeg-removal-confirm.png`.
- `make verify` passed formatting, TUF/generated assets/notices, Qodana exclusions,
  vet and build. Docker race run `.scratch/race-runs/20260917T171332Z-mhSpmc`
  completed all three UI shards successfully; the non-UI partition's sole failure was
  `TestAnalysisProtocolPreservesLimitErrorsAndConfiguration/complete`: the
  protocol-only worker has `-test.timeout=10s` and returned `unexpected EOF`
  after 10.31 seconds. Imaging and EXIF-window race packages passed, as did all
  other packages. The failed test does not call JPEG metadata removal. Its
  isolated native Linux/amd64 race retry passed: complete case 3.02s, package
  4.278s (`go test -tags no_emoji,nodynamic -race ./internal/similarity -run
  '^TestAnalysisProtocolPreservesLimitErrorsAndConfiguration$' -count=1 -v`).
  Log: `/tmp/picfetch-jpeg-unrelated-retry.log`. The full gate remains recorded as
  failed; an isolated retry is not a clean full-gate run. No unrelated deadline
  or worker-policy changes were made to obtain a pass.

The compatibility repair is complete and uncommitted. At this initial handoff,
native Windows/macOS runs, the user's exact JPEG examples and a completely green
simultaneous full-gate run remained unverified. The later
[camera-profile and clipboard follow-up](2026-09-17-jpeg-and-clipboard-followup.md)
qualifies the supplied real JPEG and records the latest verification results.
Initial user edits to `FyneApp.toml` and `todos.md`, plus the concurrent
`needs_checking.md` edit, were preserved.

Actual cost: one read-only Scout; two lead review rounds for task 1 (including
the declaration-order finding), one for tasks 2/3; one complete suite launched.
The extra review round addressed a newly established fidelity concern.
