# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal
### Comparison test deadline under build contention

Completed on September 12: the regression now uses smaller synthetic frames,
the existing five-second comparison wait budget, cancellable worker waits and
reported cleanup failures. Both frames still require detail tiles, preserving
the cancellation dependency. The original fixture failed all three race runs
with four CPU contenders sharing one core; the final fixture passed all five
Ubuntu contention runs (2.9–3.8s). All three Settle regressions pass five Ubuntu
race repetitions each, and an overlay restoring the obsolete-tile deadlock
still fails the test as expected. Evidence: `/tmp/picfetch-compare-deadline/`.

Final `make verify` passes on this Ubuntu host: formatting, TUF, Qodana exclusions,
vet, build and all four Docker race partitions. The complete streams contain
2,605 top-level passes and five existing skips; all 682 UI assignments ran exactly once.
The changed regression also passes during the full suite (2.32s).
Raw gate evidence: `.scratch/race-runs/20260912T185421Z-jHjXwb/`.
GoLand inspection of `internal/ui/compare/vector_test.go` is complete, including
weak warnings. Its only findings are intentional duplicate setup in the unchanged
queued-completion and cancellation tests; the existing exact-file
`DuplicatedCode` exclusion in `qodana.yaml` covers both. No actionable findings
remain. A fresh native `go test -race -count=5 -timeout 2m -run
'^TestCompareSettle_' ./internal/ui/compare` also passes (10.338s).

### Qualify updater notices in native release CI

Notice reconciliation and delivery guards are implemented. PR #22's
[native Linux/amd64 full CI gate](https://github.com/frathe/picfetch/actions/runs/34755687251)
passes, along with its Windows and macOS native guards. Before release, observe
the new inspection of final signed Windows archives and both native MSIX bundle
payloads. This host has no Windows SDK; local unsigned archives for all six
targets pass. The audit preserves inferred MPL matcher attribution for
`go-pathspec`; its historical `fnmatch.translate()` source does not identify an
exact CPython version, so the retained Python license is explicitly an ancestry
reference. See the [evidence and limits](plans/2026-09-13-updater-notices.md).

### Mascot-circle hint for the Hypno Spiral

Completed September 14 at Ronin's request, including the wider speech bubble and
Escape-to-close fix for focused manual search. Ten consistent-direction circles around
welcome-screen Trane's head within twenty seconds open or raise Finis. Any Finis
accepts a fresh ten-circle/twenty-second attempt anywhere within his window to
reveal a lasting speech bubble. Clicking it opens an empty manual search; the
secret phrase stays English and its parenthetical hint is translated.

Implemented with SDD/TDD; focused race tests, locale checks, shard inventory and
local format/generated-file/vet/build checks pass. The
[archived evidence](finished_refactorings/2026-09-14-mascot-circle-hint.md)
preserves Ronin's native E2E feedback and the unavailable verification results.

### Find more like this

Implemented on `feature/find-more-like-this`: reference-first ranked Grid with
up to 30 other matches, progressive preparation, reference history, ordinary
result actions, captured Favorite saves, persistent analysis and Cache settings.
The worker reuses a bounded top-30 between batches; Favorite saves capture one
list, and persistence opt-outs remain effective when cache inspection fails.
Canceled partial inventories remain visibly incomplete; missing-file records are
retained as unavailable because the cache has no persisted volume identity.
First-use setup revalidates the captured reference before admitting search.

Ronin accepted the overall results from the 446-image
[local evaluation](docs/find-more-like-this/evaluation.md) and waived exhaustive
item judgments. Quantitative precision remains unmeasured. Positive/negative
examples remain a deferred extension, and existing dependency distribution gaps
remain in LATER below.

Focused race/native regressions, build checks, locale/manual checks, shard
validation and GoLand inspections have recorded evidence. The
[continuation record](finished_refactorings/2026-09-14-find-more-like-this.md)
records fixes, test failures/passes and qualification limits. [PR #25](https://github.com/frathe/picfetch/pull/25)
is the live source for the latest commit's Codex code/security review,
Qodana/CodeQL results and complete Linux/Windows/macOS CI. Completion of the
review loop still requires that live gate; older successful checks do not count.

## LATER

### Existing dependency distribution qualification

Resolve the existing shipped dependency-closure gaps before release: HEIC embedded heic 0.1.6 AGPL/commercial grant evidence, AVIF native component notices/libyuv pin, Fyne font notices and older x/sys inventory. See [exact audit evidence](docs/find-more-like-this/dependency-qualification.md). No decoder/model substitution is included in Find more like this.

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.

### Antivirus verdicts on unreleased builds

The September 11 local builds have likely false positives: Microsoft flags both
Linux architectures as `Trojan:Script/Wacatac.C!ml`; Trapmine alone flags Windows
AMD64 as `Malicious.high.ml.score`. Microsoft reports both Windows builds as
undetected. Windows ARM64 is 0/67, but Trapmine cannot process that file type.
All 99 distinct cached dependency directories and archives match the build
metadata and `go.sum`. Vendor review remains pending; no samples have been
submitted by the agent. Record final vendor determinations and rescan the final
release artifacts. See [the investigation](docs/antivirus-triage-2026-09-11.md).

### WinGet package identifier migration

Move the existing `io.github.frathe.picfetch` manifests in `microsoft/winget-pkgs` to the
new identifier `frathe.picfetch` before the next WinGet publication; See the
[maintainer's suggestion](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559).

### Revisit HEIC at its next dependency upgrade

[MA-023](needs_refactoring.md#ma-023) tracks retiring the HEIC fork when an
approved official release includes its leak fix. Its separate upgrade trigger
remains unchanged.

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- **Verifier size refactoring (MA-024):** Declined by Ronin on 2026-09-13.
  Keep upstream Sigstore/TUF verification. Potential savings of a few megabytes
  do not justify the security risk and maintenance burden of a custom or
  trimmed verifier. The refactoring plan and upgrade watch have been removed.

- **Retained decoded map tiles (MA-025):** Accepted by the user on 2026-09-09. The map loads only when opened, and
  checking the geolocation of thousands of images is outside expected use. The upstream decoded-tile cache remains
  unbounded; its long-session impact is unmeasured. No further measurement or implementation work is planned.

- Windows releases are not Authenticode-signed. Controlled Folder Access and SmartScreen both judge by signature and
  reputation as well as by which program is writing, so an unsigned `picfetch.exe` can still be blocked even with the
  in-process swap (see Done → Bugfix above, where the block would now name `picfetch.exe` instead of `cmd.exe`). The
  real remaining fix is signing the Windows release build — Azure Trusted Signing or a purchased certificate — in
  `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
