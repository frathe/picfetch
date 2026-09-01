# PicFetch — TODOs

## Done

### What's Changed

#### New Features

![Trane comparing files](https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/trane_comparing_images.webp)

##### Compare two images

- Grid View can now compare exactly two explicitly selected images with
  **Actions -> Compare selected images** (`Cmd+D` on macOS, `Ctrl+D`
  elsewhere). The opaque in-window comparison opens immediately, loads both
  fitted 50/50 panes concurrently, and returns through **Back to Grid** or
  `Esc` without changing selection, filter, highlight, scroll position, title,
  or file set. Selected files hidden by a filter remain eligible, and a load
  failure returns safely to the untouched grid.
- Comparison now permanently identifies both sides with translucent corner
  badges and an exact left-to-right window title. Matching basenames expand to
  the shortest distinguishing folder/file suffix. A compact translucent
  top-right toolbar keeps **Back to Grid** available while loading and enables
  **Swap** once both images are ready; Swap exchanges both roles immediately
  without another decode or any change to the covered grid.
- Comparison is now an exclusive main-window mode. Its toolbar, `Esc`, F1 help,
  and normal window closing remain available, while ordinary viewer, grid,
  file, Favorites, and Actions commands are disabled and guarded at their
  direct entries. A full-surface input shield keeps typed and pointer input out
  of the covered grid. Drops, file-dialog requests, and native Open With
  deliveries are discarded with **Return to Grid View before opening files**.
- Zoom and pan now stay linked across the comparison panes. Wheel and
  `+` / `-` zoom both, dragging either pane or using `Shift`+wheel pans both,
  `0` fits and centers both, and `1` shows both at actual pixel size. The
  shared normalized center is clamped so neither pane exposes blank space.
- Comparison can now switch from fixed side-by-side panes to a full-viewport
  swipe reveal. The ready-gated **Swipe** / **Side by side** control retains
  the linked view and divider position; dragging the themed divider changes
  only the reveal, while ordinary drags still pan both images. `Left` / `Right`
  move the divider by 5 percentage points, their Shift variants move it by 1,
  and `Home` / `End` choose 0% / 100%.
- Layout switches and main-window resizes now preserve the shared point of
  interest and zoom while recomputing each image's fitted scale and clamping
  away blank space. Actual size remains an absolute 100% scale, and **Swap**
  preserves the layout, transform, and divider while exchanging swipe roles.
  Every new comparison resets fitted and centered, side by side, at 50%.
- Comparison now preserves the canonical source fidelity of every supported
  format. Raster panes retain their full decoded first frame, EXIF orientation
  and RAW embedded previews follow the normal imaging path, and animations stay
  frozen on frame zero. SVGs rerender per pane at the current device-pixel size
  after zoom, layout, resize, and Swap, with cancellation and stale-result
  protection. Existing encoded-input and vector-raster limits remain honest;
  failures return to the unchanged grid instead of reducing or removing a
  source, while two successful full decodes may use their combined memory.

#### Bugfix

- Slow update downloads no longer fail merely because receiving the archive
  takes longer than the updater's network timeout.

#### Internal

## ACTIVE DEVELOPMENT

## TODO

### Compare two grid-selected images

- **Manual Comparison:** When the ctrl-key is hold down and one of the sides is paned of zoomed,

### Compare two grid-selected images additions
- fix: 
  - Warning:(128, 6) Variable 'container' collides with imported package name
    ~/Projects/picfetch/internal/ui/compare_test.go
  - Warning:(75, 6) Variable 'container' collides with imported package name
- only the side the cursor is on is affected. so the lock between the two sides is released as long
  the crtl key is hold down. WHen letting go of ctrl the two sides are glued back together. but at
  the current position and zoom level. from there one pan and zoom again influences both sides.
- swipe view should be the default view in comarison mode..

### Functional test coverage

Audit baseline, 2026-09-01: package-local statement coverage came from a
passing run of:

```sh
go test -count=1 -skip '^TestE2E' \
  -coverprofile=/tmp/picfetch-cover-clean.out ./...
```

"Effective" coverage below additionally instruments the named package while
running its existing higher-level tests. Percentages locate candidates;
completion means pinning useful PicFetch behavior through an established
interface, not reaching a blanket coverage target.

1. **P0 - `internal/update` (77.2% local / 78.8% effective).**
   - Through `update.Apply`, force a plist-backup failure after binary
     replacement begins (a conflicting `Info.plist.old` is the deterministic
     fixture). Verify the installed binary and plist remain the original
     versions and partial replacement files are removed.
   - Through `Client.Download` with an HTTP test server and fake external
     verifier, reject ZIP symlinks and TAR symlink/hardlink entries. No usable
     stage or file outside the staging directory may be created.
   - Do not exercise the real Sigstore implementation or unreachable platform
     stubs merely to cover their lines.
2. **P1 - `internal/filesort` (73.7% local / 89.9% effective).**
   - Pin `FromPref` / `PrefValue` round-trips for every `Modes` entry and the
     unknown-value fallback to `ByName`.
   - Verify missing files use the documented zero-key fallback for capture
     date, modification time, and size sorts, while `Order` leaves its input
     slice unchanged.
3. **P1 - `internal/ui/copyselection` (85.4% local / 88.1% effective).**
   - Through `Feature` and canvas gestures, verify all eight image-region
     selection handles resize the correct edges, including corner and crossed
     movement.
   - Move a committed image-region selection beyond each image edge and verify
     it clamps to the image while preserving its dimensions.
   - Do not test `resizeRect`, renderer no-ops, or Fyne event methods directly.
4. **P2 - `internal/favthumbs` (86.9%).**
   - Verify `Read` falls through from a corrupt JPEG preview to a valid PNG
     sibling.
   - Verify `Sweep` preserves directories and other non-regular files whose
     names resemble preview files.
5. **P3 - `scripts/releasenotes` (86.8%).**
   - Exercise `Build` and `ClearDone` with CRLF input and with `## Done` as the
     final section, preserving surrounding Markdown and changelog output.

Future tests stay at the exported seams named above. Each must be seen failing
against a deliberate behavior break before it is accepted, then pass its
focused package test and `make verify` at final handoff.

Do not chase the low numbers in `main`, `internal/uitest`, `scripts/synctuf`,
Fyne widget adapters, or the accepted OS-integration seams. `internal/session`
and `internal/favstore` already cover their useful persistence behavior; their
remaining gaps are chiefly framework/filesystem failure plumbing.
`internal/ui/autoupdate` and `internal/appearance` reach 91.4% and 96.4%
respectively when their higher-level tests are counted. Re-audit
`internal/ui/compare` only after the current command-isolation work lands; its
present uncovered input-shield methods are deliberate no-ops.

## not deemed worth implementing (edge cases)

- Windows releases are not Authenticode-signed. Controlled Folder Access and
  SmartScreen both judge by signature and reputation as well as by which
  program is writing, so an unsigned `picfetch.exe` can still be blocked
  even with the in-process swap (see Done → Bugfix above, where the block
  would now name `picfetch.exe` instead of `cmd.exe`). The real remaining
  fix is signing the Windows release build — Azure Trusted Signing or a
  purchased certificate — in `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the
IDE's 71. The 8 fragments CI is missing are 7 in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries
exactly 3 `#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in
db" warnings, naming exactly those two files and no others, emitted
immediately after the line `The Project analysis stage completed in 41s` —
so Qodana's own log shows detection succeeded and serialisation into the
report/SARIF failed afterwards. This is an upstream defect, not a picfetch
config problem: nothing here suppresses or excludes those two files, and the
drop happens before any project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes
this defect invisible going forward in this repository, because every
dropped fragment happens to live in a test file that the exclusion now
removes from the inspection entirely — recorded here so the defect is not
lost along with the rule that used to surface it. Of the 12-fragment
CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the
other 4 are the source-suppressed production fragments in the orientation
pixel loops recorded above, so nothing about that gap is left open — only
the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte
offsets and anchoring detail, and `plans/2026-08-29-qodana-serialisation-bug-report.md`,
Task 8's draft of the upstream report text — as of this writing not yet
submitted to JetBrains; check that file for whether it has been sent since.
