# MA-028 implementation verification

Date: 2026-09-27. Implementation base: `e440685` (clean worktree).
User authorization: `/implement MA-028 use tdd and sdd`, with local commits
explicitly allowed. Push and PR creation were subsequently authorized with the
portable tracker move; merge and release remain outside that authorization.

Status: implementation and available local deterministic gates passed.
Scoped Linux native-input scenarios passed with OS-injected events, but a user
follow-up exposed a native Escape/maximize reset defect. That defect is now
repaired and passes repeated native checks, a red/green boundary regression,
changed-file GoLand inspections and a fresh full `make verify`. Windows native
and physical-keyboard qualification is now complete at `10f16a0`; see the
[Windows evidence](command-admission-windows-qualification-2026-09-27.md).
Linux physical-input/literal Make-run and full macOS acceptance remain open. Fresh CI,
CodeQL, Qodana SARIF and Codex code/security reviews pass at `4674cca`; see the
[PR 66 review record](command-admission-pr-66-review-2026-09-27.md) for exact
evidence and the final documentation-head gate. See the
[Linux qualification record](command-admission-linux-qualification-2026-09-27.md).
The execution plan stays in `plans/`.

## Ownership and structural review

T0 reviewed the change against the accepted D1-D7 decisions and the base diff.
`commandpolicy.go` accepts values only and returns admission, refusal, yield and
target kind. `commandadmission.go` observes UI-owned facts and applies feedback
or yielding only after an affirmative decision. Menus consume named decisions
alongside presentation facts; stale callbacks recheck the same policy.
The generic yielding menu/shortcut wrappers and Favorites runner are removed.

Payload capture, request tokens, confirmations, source identities, storage,
worker cancellation, queues and stop/join barriers remain with their previous
owners. No worker, package, runtime dependency, translation or native payload
was added. Fyne remains pinned at v2.8.0; existing dependency notice obligations
remain unchanged and are checked by Make.

A new restricted visit needs shared context/policy changes, not new predicates
in every menu/shortcut adapter. Remaining surface checks dispatch feature-local
keys or capture feature-specific subjects; they are not duplicate entry policy.
Settings-owned setters, scan/sort/committed-write reconciliation, deletion
confirmation, export format choice and slideshow advancement are continuations,
not fresh unrelated main-window commands. `loadImage` makes that distinction
explicit while keeping the existing display load path.

Clipboard admission now refreshes menus on entry and on the existing result
queue before completion is signaled. Region-copy completion joins that same
refresh. Cancellation after navigation discards the result but restores menus;
terminal shutdown still releases its signal without waiting for UI delivery.
The closed flag is atomic because that terminal check is made by the worker.

## Command/route ledger

Routes: M = explicit menu; A = its accelerator; S = registered shortcut;
K = plain/modified key; B = bare guarded action; H = Host/link/gesture;
D = drop; O = OS-open; C = asynchronous continuation. A listed accelerator is
source/test-driver coverage except for the explicitly observed Linux/Windows
routes in their qualification records; ticket 10 remains open. An omitted
route is N/A because no such binding/entry exists, not an untested alternate
implementation. Related feature-owned controls are identified explicitly.

All decision observations are gathered by root `commandContext`; the final
column names the existing capture/effect owner. Tests refer to
`TestCommandAdmission*` suffixes plus existing named regressions.

| Command | Applicable routes and seam | Proving cases; capture/effect owner |
| --- | --- | --- |
| Save Changes | M/A/S/B `saveRotation`; C file-work completion | YieldOrdering, Queries, BusyCopy, Routes/save, Async; display/filework |
| Copy (editing) | A/S `editingShortcut` | TextEditing; focused Entry owns text selection/clipboard |
| Copy (context image intent) | M/B `copySelection` | Routes/clipboard, CopySelection regressions; region, Grid files, displayed image priority |
| Explicit displayed-image copy | B `copyImageToClipboard` | TextEditing explicit action, clipboard regressions; immutable display capture/clipboard worker |
| Copy Grid files | B `copyGridSelection`, reached by context Copy | Routes/clipboard; Grid targets/clipboard worker |
| Copy Path | M/A/S/B `copyPathToClipboard` | Routes/clipboard, ClipboardBusyPreservesActiveOperationAcrossCopyRoutes; current URI/text clipboard |
| Start/repeat Copy Selection | M/A/S/B `startRegionCopy` | CopySelection regressions; region stable capture/geometry |
| Region confirm/cancel | K/H, already-owned feature controls | CopySelection regressions; region token/clipboard lifetime |
| Select All | S/B `editingShortcut` / `selectAllInGrid` | TextEditing, Routes/clipboard; focused Entry or Grid selection |
| Open chooser | M/A/S/B/H dropzone `openFileDialog` | OpenChooser*, Routes/open; chooser lifetime |
| Open URI collection | B/H `OpenFiles`, D/O `handleDrop`, C chooser delivery | Async queued chooser, OpenWithHandler*, modal-drop regression; scan/sort/display |
| Open Favorite | M/A/S/B `Feature.Open/openFavorite`, H `OpenFavorite` | Routes/open favorites, Favorites host tests; Favorite storage/previews/common collection path |
| Restore previous session | H/B `restoreSession` | Routes/open restore; saved session/common open path |
| Close Files | M/B `closeFiles` | Routes/open, BusyCopy, CopySelection regressions; scan/sort/collection owners |
| Add Favorite / save results | M/A/S/B `AddCurrentList/AddFiles` | Routes/open favorites, favorite dialog notifications; captured list/naming workflow |
| Manage Favorites | M/A/S/B `ShowManage` | Routes/open favorites, Favorites tests; dialog/store |
| Overwrite/remove Favorite | H/K, owned confirmation/rebuild | Favorites tests; no new unrelated-command gate |
| Export prompt | M/A/S/B `promptExport` | Routes/files, ModalOwnership, export regressions; existing ChoiceCard |
| Export format/options | K/H/B `exportAs`, C write reconciliation | ExportCancellation*, ExportCommitted*, FileMutationReconciliation*; captured pixels/filework |
| Trash request | M/A/S/B `requestDelete/deleteGridSelection` | Routes/files, ModalOwnership, deletion regressions; URI targets/deletion feature |
| Trash confirm/cancel | K/H, owned card | ModalOwnership, deletion regressions; completed OS effects reconcile by identity |
| Reveal | M/A/S/B/H info link `revealCurrentFile` | Routes/files, reveal regressions; captured current path/reveal worker |
| Wallpaper | M/A/S/B `setAsWallpaper` | Routes/files, wallpaper regressions; captured pixels/wallpaper lane |
| Mosaic wallpaper | H, existing separate-window control | Wallpaper/mosaic regressions; captured request and existing wallpaper lane |
| Viewer | M/A/K/B `showViewer` | WindowCommandAdmissionMatrix, Visits; root return composition |
| Grid Show / toggle | M/A/B `showWindowGrid`; K G retains local toggle | WindowCommandAdmissionMatrix, Routes/windows; Grid visits |
| Picture-frame Show / toggle | M/A/B `showWindowPictureFrame`; K/B toggle | WindowCommandAdmissionMatrix; slideshow owner |
| Compare selected | M/A/S/B `compareSelected` | Compare*, Routes/windows; exactly two Grid sources/comparison owner |
| EXIF | M/A/K/B/H info link `showWindowExif` | Routes/windows, WindowCommandAdmissionMatrix; EXIF singleton |
| Settings | M/B `showSettings` | MenuCallbacksRecheckAdmission, CopySelection regressions; settings snapshot/singleton |
| Manual | M/A/K/B/H About/Spiral links `Help.ShowManual` | ModalOwnership, Compare*, Help tests; Help singleton |
| About | M/B `Help.ShowAbout` | Routes/windows, MenuCallbacksRecheckAdmission; Help owner |
| Release Notes | M/B `Help.ShowReleaseNotes` | MenuCallbacksRecheckAdmission, Help tests; Help owner |
| Licenses | M/B `Help.ShowLicenses` | MenuCallbacksRecheckAdmission, Help tests; Help owner |
| Discussions | M/B `Help.ShowDiscussions` | MenuCallbacksRecheckAdmission, HelpMenu; browser intent |
| Owned community hyperlink | H Explorer setup/About `OpenDiscussionsLink` | HelpMenu; dialog control, not an unrelated main-window command |
| Finis | H/B `Help.ShowFinis` | Help tests, policy coverage; Help singleton |
| Hypno Spiral | H/B secret/gesture `openSpiral/openSpiralForGesture` | Policy, CopySelection/Spiral regressions; frozen source snapshot/Spiral owner |
| Explorer | M/A/K/B `showExplorer`, C setup/preparation | Async, VisualSimilarityExplorer; feature setup/analysis, root source preparation |
| Location Map | M/A/K/B `showLocationMap`, C validation/preparation | Async, Visits, LocationMap; map metadata/camera, root visit composition |
| Map return, image/cluster and Explorer cohort/unassigned | K/H, owned visits; selected image reaches `ShowImage` | Visits, WindowCommandAdmissionMatrix, Explorer/LocationMap; feature-local cameras/frozen membership |
| Mosaic | M/A/K/B `showMosaic` | Routes/maps, mosaic regressions; source-pool capture, separate-window owner |
| Select sort / cycle | M/B `setActionsSort`; K/B `toggleSort` | Routes/search, ActionsMenu*, policy; filesort lifecycle |
| Hide duplicates | M/A/K/B `toggleHideDuplicates`; Grid D policy before local handling | Visits, Routes/search; dupes model/Grid |
| Browse variants | M/A/K/B `showActionsVariant/browseCurrentDuplicates`; C accepted grouping | Async variants, Visits; Grid frozen group |
| Find more like this | M/A/S/B `findMoreLikeThis`; C setup | Routes/search, FindMoreLikeThis*, Async; captured generation/reference/search owner |
| Search Back/Exit/save results; selection analysis | H/K, feature-owned controls | FindMoreLikeThis*, VisualSimilarityExplorer; search/Explorer owners; AddFiles rechecks naming entry |
| Next/previous | K/B/H EXIF `StepImage` | StepImage*, YieldOrdering, Visits; display navigation |
| First/last; Grid-selected image | K/B/H `ShowImage` | Async navigation, Visits cohort handoff; display navigation |
| Rotate clockwise/counter-clockwise | M/A/K/B `rotateBy` | Routes/navigation, CopySelection*, CompareOrientation; display owner |
| Reset rotation + fit | K/B `resetRotation`, then existing zoom fit | CopySelection*, Policy; display/zoom |
| Actual size and zoom +/- | K; M/A/B zoom actions | Routes/navigation, CopySelection*, Policy; zoom geometry |
| Pan/pointer zoom | H, owned viewport controls | Existing zoom/CopySelection regressions; feature-local geometry |
| Merge | M/A/K/B `toggleMergeMode` | Routes/navigation; standing root setting |
| Info overlay | M/A/K/B `toggleInfoOverlay` | Routes/navigation, ActionsMenu*; existing info owner |
| Shuffle | K/B `toggleSlideshowShuffle` | Routes/navigation; slideshow setting |
| Interval | K Up/Down in frame mode | Policy, slideshow/key regressions; slideshow owner |
| Settings setters / automatic load/advance | H/C, already-owned effects | Async committed save, FileMutationReconciliation*, existing slideshow tests |
| Escape / native window close | K/H, ordered owner dispatch, no generic admission wrapper | EscapeUnwindsModesBeforeReset, CopySelectionBusyBlocksOtherCommands; Linux native close passed while busy and modal; remaining platforms need ticket 10 |

## Observed red/green evidence

Initial regression failures were observed for unavailable Save yielding via
menu/shortcut, bare Save failing to yield, modal bare file/window/map/sort/
navigation/open entries, focused editor delivery, busy Save enablement, whole-bar
modal enablement, Favorite dialog menu notifications, fresh ShowImage beneath
a prompt, native Copy interception selection, the Explorer cohort handoff,
delayed variant presentation and stale underlying Entry focus. Each was fixed
inline and rerun green.

Explicit negative verification used Go build overlays in a temporary directory,
not edits to the working tree: removing modal refusal failed
`TestCommandAdmissionPolicy`; adding cancellation to the query adapter failed
`TestCommandAdmissionQueries` with “availability query had side effects”.
The final run used no overlay. All nine named guard groups are present.

Accepted D4 corrections replace old drop/OS-open-cancels-delete and
drop-cancels-Explorer-setup expectations. The stale-search reference test now
uses an already-admitted load beneath setup, preserving its generation/reference
assertion. A combined run also exposed an existing Settings window leak in a
Copy Selection fixture; that test now closes the window it created.

Test isolation incident: the first new file-action red run omitted OS stubs
and reached real reveal/wallpaper handlers. The user was notified immediately;
the desktop may have changed. Stubs were installed, the meaningful red rerun
was isolated, and all later runs use the normal test seams.

The first complete Make run exposed two compatibility regressions: clipboard
capture accepts existing surface pixels without requiring a decoded frame count;
and direct Close Files must consume a pending startup picture-frame request. The
capability/reset fixes passed their original failing tests. Lead review also
preserved empty-view D preference toggling while its explicit menu remains
unavailable. A new completion assertion failed because Copy and Copy Path stayed
disabled after region copying; result-queue release now restores availability
before signaling completion. Region success, ordinary success/cancellation and
terminal no-UI cancellation all pass their focused regressions.

## Deterministic verification

- `go test -tags no_emoji,nodynamic ./internal/ui -list '^TestCommandAdmission'`:
  all nine groups listed; package passed.
- Final expanded AC/regression union, `-count=1 -json`: root UI passed
  in 47.929s after the clipboard menu correction. Executed cases/output are
  retained in `/tmp/picfetch-ma028-regressions-final.jsonl`: 124 top-level
  passes, no failures, and one pre-existing platform skip:
  `TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` requires a
  case-insensitive filesystem. All nine new guard groups ran; the skipped
  export case remains unverified and must run during platform qualification.
- Final command-admission/clipboard focused union: passed in 5.469s.
- `go test -tags no_emoji,nodynamic -count=1 ./internal/ui/help ./internal/ui/menus ./internal/ui/favorites ./internal/ui/explorer`:
  all four packages passed.
- `make check-test-shards check-qodana-test-exclusions`: passed, 716 root
  runnables assigned exactly once to three shards.
- `GOOS=windows GOARCH=amd64 go vet -tags no_emoji,nodynamic ./internal/...`:
  passed after the final clipboard correction; this is not native input evidence.
- First `make verify`: formatting, TUF root, exact Qodana exclusions, generated
  tag vectors/artwork, updater/AVIF notices, vet and build passed. Native
  Linux/x86_64 Docker race run failed on the clipboard/startup cases above;
  non-UI packages passed. The ui-2 panic prevented full partition coverage.
  Log: `/tmp/picfetch-ma028-verify.log`; retained artifacts:
  `.scratch/race-runs/20260926T232653Z-nMfhQj/`.
- Final `make verify`: passed (exit 0), including the complete native
  Linux/x86_64 Docker race suite. All non-UI partitions and all three UI shards
  passed; UI shard times were 476.794s, 380.424s and 387.604s. The root suite
  recorded 714 top-level passes and two existing environment/platform skips:
  the case-alias export test above and opt-in `TestHEICNativeImageOperations`.
  Other packages retained their existing HEIC/case-alias/platform/permission
  conditional skips; no test or worker-isolation policy was relaxed.
  Log: `/tmp/picfetch-ma028-verify-final.log`; retained JSON and container
  exit evidence: `.scratch/race-runs/20260926T234356Z-P6EUK7/`.
- `git diff --check`: passed, including documentation and manifests.

## GoLand inspection evidence

The local Qodana guide was read. No callable IDE-local Qodana scan interface
was available, so the documented fallback was used: GoLand
`get_file_problems(errorsOnly=false)` for every changed/new Go file.
This includes weak warnings reported by that interface. All 68 files returned
empty findings, with no timeouts or skipped files, after the final code changes.
No suppression was added or broadened. This is the IDE inspection profile,
not proof of `qodana.starter` or config-exclusion equivalence; no fresh
post-suppression SARIF, CodeQL or GitHub CI result is claimed.

Analyzed revision: base `e440685` plus the implementation working tree.
The SHA-256 of sorted `sha256sum` lines for the following scope is
`a34d1e0bc21ad2060c94f4412e3efe2c976ffd63ef4e99ca83632e6806d64640` (reproducible before/after committing unchanged files).

```text
internal/ui/actionmenu.go
internal/ui/actionmenu_test.go
internal/ui/autoupdate.go
internal/ui/batch.go
internal/ui/browsing.go
internal/ui/clipboard.go
internal/ui/clipboardwork.go
internal/ui/commandadmission.go
internal/ui/commandadmission_test.go
internal/ui/commandpolicy.go
internal/ui/compare.go
internal/ui/compare_fidelity_test.go
internal/ui/compare_test.go
internal/ui/copyselection.go
internal/ui/copyselection_lifecycle_test.go
internal/ui/delete_test.go
internal/ui/drop.go
internal/ui/explorer.go
internal/ui/explorer/cohort_workflow.go
internal/ui/explorer/preset_workflow.go
internal/ui/explorer/setup.go
internal/ui/explorer_test.go
internal/ui/export.go
internal/ui/favorites/add.go
internal/ui/favorites/confirm.go
internal/ui/favorites/favorites.go
internal/ui/favorites/favorites_test.go
internal/ui/favorites/manage.go
internal/ui/features.go
internal/ui/filework.go
internal/ui/heic.go
internal/ui/help/about.go
internal/ui/help/finis.go
internal/ui/help/help.go
internal/ui/help/licenses.go
internal/ui/help/manual.go
internal/ui/help/manual_test.go
internal/ui/help/whatsnew.go
internal/ui/info.go
internal/ui/keys.go
internal/ui/load.go
internal/ui/locationmap.go
internal/ui/menu.go
internal/ui/menu_yield_test.go
internal/ui/menus/menus.go
internal/ui/menus/menus_test.go
internal/ui/mosaic.go
internal/ui/openfiles.go
internal/ui/openwith_test.go
internal/ui/reveal.go
internal/ui/reveal_test.go
internal/ui/rotate.go
internal/ui/save.go
internal/ui/session.go
internal/ui/shortcuts.go
internal/ui/slideshow.go
internal/ui/sort.go
internal/ui/sourcechange.go
internal/ui/tunnel.go
internal/ui/viewer.go
internal/ui/visibility.go
internal/ui/visualsearch.go
internal/ui/visualsearch_test.go
internal/ui/wallpaper.go
internal/ui/windowmenu.go
internal/ui/windowmenu_darwin.go
internal/ui/windowmenu_darwin_test.go
internal/ui/windowmenu_test.go
```

## Native evidence and remaining acceptance

Host observation: Linux/x86_64; Docker reports native Linux x86_64. Fyne is
v2.8.0. The [Linux native qualification](command-admission-linux-qualification-2026-09-27.md)
ran the Make-built production binary at clean revision `6db8d73` (unchanged
implementation `9dc3a81`) on Ubuntu/GNOME Wayland via XWayland. OS-injected
keyboard/mouse events passed editor Copy/Select All, explicit image-menu intent,
modal refusal and owner controls, held-copy menu entry/recovery, and native close
while busy/modal. Screenshots, payload comparisons, exact harness and launch
details are retained. Physical-keyboard operation was not performed, and the
binary was launched after `make build`, not through literal `make run`.
A later user screenshot confirmed that the small surface inside a maximized
window was an actual defect. A fresh one-image native reproduction fails:
Escape keeps 1920 x 1131/maximized instead of the initial 624 x 409 client target.
Restoring native maximization before Escape makes the control pass. The Linux
record retains that diagnostic history and the subsequent authorized repair:
dynamic reset now restores native state before resizing. Five ordinary native
repeats, Grid, multi-image/rotation and both fixed-size controls pass. The new
boundary regression also fails when restoration is deliberately omitted through
a temporary build overlay. All three changed Go files have clean GoLand
inspection results, and fresh `make verify` passed with artifacts in
`.scratch/race-runs/20260927T103730Z-3Tubvu`. Exact final source hashes and
inspection scope are in the Linux repair evidence; the other 65 inspected
Go files retain their unchanged-code evidence above.
Windows and macOS desktops were unavailable locally. Windows internal-package
cross-vet passed. Subsequent PR 66 native CI compiled and passed the isolated
AppKit assertion on both macOS architectures and case-alias export on Windows
and macOS; see the [review-loop record](command-admission-pr-66-review-2026-09-27.md).
Those results do not establish physical-input/full-desktop acceptance.

Pinned-source tracing established that GLFW tries matching menu shortcuts
before focused widgets and canvas bindings, without checking Disabled.
Builtin Copy has a different shortcut name from PicFetch's CustomShortcut
menu hint, so the ordinary GLFW editor route remains separate. AppKit uses
Key/Mod equivalents and can invoke the image menu callback directly. The
Darwin bridge now clears only Copy's native key equivalent on every rebuild,
leaving physical Cmd+C to GLFW and explicit image-menu selection intact.
Consequently AppKit no longer draws that item's native Cmd+C hint. This is
part of the D5 route correction, not native runtime evidence.

Ticket 10 records the completed Linux OS-injected checks and repaired reset
defect separately and returns to ready-for-human: the literal physical-input
procedure and untested platforms still require native operators.
Windows subsequently completed all four groups, including physical editor
shortcuts. Complete the remaining Linux procedure and all four groups on macOS;
retain revision/OS/input route and actual payload/visible outcomes.
Do not archive the plan or claim MA-028
accepted until the remaining observations and required CI gates are complete.
The case-alias export gap is now closed at `75fd69e`: the existing regression
passed on a disposable Linux `vfat`/FAT16 filesystem, once normally and five
times with race instrumentation, with no skips. See the Linux qualification
record for commands, hashes, failed setup attempts and confirmed cleanup.
The original ext4 and kernel NTFS skips are retained as historical limitations,
not counted as passed observations. This follow-up changed no code, so the
preceding full Make and GoLand evidence carries forward.

The subsequently found CI selection gap is now fixed locally: Windows/macOS
jobs invoke a focused `command-admission` suite for case-alias export and the
Darwin Copy key-equivalent assertion, without Linux-only goldens. Its
command-boundary and workflow tests were seen red/green; missing/skipped guards
are rejected. The real runner passes on FAT16 and rejects an ext4 skip. See the
new native-CI follow-up in the Linux qualification record for inspection,
full-suite and negative-control evidence. That local work alone did not
establish Windows/macOS execution. Subsequent PR 66 native CI passed the
case-alias guard on Windows amd64 and both macOS architectures, plus the
Darwin Copy-menu assertion on both Macs; T0 inspected their retained test events.
At the local qualification handoff, GitHub had no PR or runs for this branch.
The user subsequently authorized publication and the review loop. Fresh remote
gates pass at `4674cca`, with exact runs and SARIF dispositions in the review
record. Final-head checks remain mandatory for the documentation-only follow-up,
and native CI does not replace the physical-input procedure.
