# Camera profile and clipboard completion follow-up

Route: Standard, two independent bounded fixes in imaging and clipboard. Lead
owns design, tests, implementation and review. One read-only scout traced
clipboard admission across the existing UI; no delegated code or review.

The attached real 4048x3036 JPEG is refused with ErrJPEGMetadataProfile. Its v2
matrix/TRC profile has vendor device-attribute bits and standard optional lumi,
meas, tech and vued tags. Qualify the numeric/enumerated fields, remove descriptive
identity, retain exact color parameters and all existing structural/resource
checks. Keep user photographs and their extracted profiles out of the repository;
derive synthetic regression inputs from the documented ICC layouts. Validate
against the actual attachment on a temporary copy and LittleCMS under all intents.

Linux image/file clipboard helpers fork an owner that retains output descriptors.
Cmd.Output waits for those pipes after the parent has exited, so the viewer's
shared admission never releases and Copy Path refuses indefinitely. Finish at
parent completion while retaining useful command errors and keeping the owner
alive. Do not weaken the viewer's protection against genuinely competing writes.
No new dependencies, shipped runtimes, notices or UI strings.

Tasks (Lead, independent, one focused red/green/review each):

1. Extend existing jpegprivacy tests and jpegicc normalizer for these standard
   optional fields. Refuse malformed fields and unknown transform tags without
   writes; strip vendor identity/descriptions and preserve exact transforms and
   image bytes. Verify: go test -tags no_emoji,nodynamic ./internal/imaging -run
   TestJPEGMetadataRemovalProfiles; temporary public-API attachment probe and
   testdata/jpeg-removal/reference.py check on extracted before/after profiles.
2. Reproduce inherited output-descriptor lifetime using controlled helper
   processes without touching the desktop clipboard. Fix command completion in
   internal/clipboard and retain useful stderr on failures. Verify: go test
   -tags no_emoji,nodynamic ./internal/clipboard and existing UI clipboard/action
   regressions. Review cleanup, resource ownership and diagnostic bounds.
3. Update qualification/todos/evidence, inspect every changed Go file with GoLand,
   and run make verify once after focused checks. Previous JPEG full gate had one
   unrelated load-sensitive analysis-protocol timeout; keep that evidence honest.

Limits: no promise for every ICC model or nonstandard vendor color interpretation;
Linux tests do not establish Windows/macOS native behavior. No commit/push.

## Evidence

- Original attachment probe returned state Unsupported with
  `ICC profile is not qualified for metadata removal`. All 12 synthetic
  v2/v4 field cases failed with the same error before the change.
- Those cases now pass. Sixteen malformed optional-field cases plus reserved
  media-attribute bits, unknown transforms and existing assembly checks refuse
  without mutation. Existing image/declaration/profile/lifetime tests pass.
- Public mutation succeeds on a temporary copy of the supplied JPEG. ExifTool
  shows only `ColorSpace=sRGB` in EXIF and no XMP/IPTC. Strict djpeg pixel output
  is byte-identical; LittleCMS checks 343 RGB samples for each of four rendering
  intents and reports max_abs_xyz=0. See the qualification record for the digest.
  Original attachment and private profile were not added to the repository.
- `TestClipboardCommandOwnerLifetime` failed before the change with
  `copy remained pending after launcher exit while clipboard owner held its
  streams`. It now passes while the controlled owner is still alive; the owner
  can continue writing its descriptors after launcher completion. The test does
  not use the real system clipboard. Failure diagnostics remain discoverable
  through exec.ExitError and are read into memory with a 64 KiB cap. Temporary
  diagnostic files are removed when the launcher finishes.
- Clipboard package tests pass under race detection. Existing viewer tests for
  image/grid/region/path admission, queued completion, cancellation and Copy Path
  menu dispatch pass. Shared admission remains held while real work is pending.
- GoLand inspected all four changed Go files in this follow-up, including weak
  warnings, and reported no findings. No new test files or UI test names were
  added, so Qodana/shard inventories are unchanged. Temporary probe source was
  removed.
- The lead reviewed source bounds, finite ICC enumerations, exact retained values,
  private-field removal, child descriptor lifetime and failure reporting. One
  review round per task; one full gate launched after focused checks.

Primary-source clipboard evidence: [wl-copy v2.2.1 owner handoff](https://github.com/bugaevc/wl-clipboard/blob/v2.2.1/src/wl-copy.c#L47)
retains stderr when forking; [Go Cmd.Output and Wait](https://go.dev/src/os/exec/exec.go)
wait for captured stream EOF. The runner uses the launcher's exit as completion,
stdout connected to the null device, and a real stderr file rather than pipes.
Native Windows/macOS command execution remains unverified.

## Final gate and handoff

`make verify` passed format, generated assets/notices, TUF checks, test inventories,
vet and compilation. Docker race artifacts are in
`.scratch/race-runs/20260917T174804Z-8nFBpT`; complete log:
`/tmp/picfetch-jpeg-clipboard-verify.log`. All three root UI shards passed
(426.161s, 443.280s, 441.416s). The non-UI partition had 62 passing packages,
5 skips and 2 failures. Changed clipboard (2.285s), imaging (156.315s) and
EXIF-window (35.873s) packages passed.

The two failures are in untouched test paths:

- `TestAnalysisProtocolPreservesLimitErrorsAndConfiguration/complete` returned
  unexpected EOF after 11.06s; its protocol-only helper has a 10s test deadline.
  This was also the previous full gate's sole failure.
- `TestShaderPaneRenderer_SameSourceViewChangeDoesNotCancelAllocatedTile` timed
  out waiting for RGBA tile allocation under its 3s deadline (4.08s reported).
  It does not read JPEG metadata or call clipboard code.

After the full run ended, an isolated native Linux/amd64 race retry of both
specific tests passed: analysis complete case 3.17s (package 4.450s), tile case
0.73s (package 1.766s). Log: `/tmp/picfetch-jpeg-clipboard-retry.log`.
The full gate remains failed; these retries do not turn it into a green full
run. No unrelated deadline, test or isolation policy was changed.

`make build` succeeded and produced the updated `bin/picfetch`. Changes remain
uncommitted. User edits in FyneApp.toml, needs_checking.md and unrelated todo
entries are preserved. Private photo/profile artifacts remain under /tmp only.
Actual follow-up cost: one read-only scout, one lead review per fix, one full
suite and one targeted retry of its two unrelated failures.
