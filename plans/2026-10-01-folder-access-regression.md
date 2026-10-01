# Store folder-access regression

Route: Deep, platform-native panel lifetime plus captured bookmark resolution.
Owner: Pico inline; no delegation, no new dependency or notice obligations.

Ronin reports wrong initial folder and an interactive confirmation that cannot
complete after selecting the correct parent. Preserve exact-parent authority,
file-only cancellation, sibling discovery and remembered explicit grants.

Task graph: native reproduction -> presentation fix -> moved-source regression
-> focused race/native qualification -> inspection -> commit/push/review -> build.
All tasks are T0 inline, with zero spawns and focused local suites; full CI remains
the GitHub native Linux/amd64/platform gate.

## Acceptance

1. In the sandboxed PicFetch test app, select one controlled JPEG, choose its
   exact parent and confirm. The panel must close and title report both fixture
   images. Cancel a different folder request: only the selected image opens.
   Verify through the Computer Use native fixture loop.
2. Native panels start on UI and complete without blocking Fyne's dispatcher;
   the tracked chooser worker owns the retained native URLs until serialization.
   Verify normal Open, folder permission, cancel and Save in the same app.
3. A moved file bookmark suggests its resolved current parent and attaches that
   folder's scope, with balanced acquisition/release. Command:
   `go test -race -tags no_emoji,nodynamic ./internal/filepicker`.
4. Wrong/broader folders, revoked grants, cancellation, directory/multiple
   selections and persisted grant reuse retain existing boundaries. Same suite.
5. GoLand every changed code file including weak warnings, make verify-build,
   fresh GitHub code/security review, CI, CodeQL and post-suppression Qodana.

Files: internal/filepicker/darwin.go, folders.go and existing folders_test.go;
native completion coverage in existing darwin_test.go if needed after assessment.
No native OS operation runs in ordinary unit tests.

## Diagnosis evidence

- Existing packaged sandbox app at 382a40d: exact fixture Photos folder selected,
  Return leaves permission panel open. AX reports disabled confirmation. Record
  `.scratch/folder-access-regression/repro.md` contains the native red assertion.
- Ranked URL comparison, stale bookmark, restored panel directory and completion
  hypotheses were tested. Outside-sandbox NSURL comparison passes every spelling.
- Native thread sample shows runModal on the correct macOS main thread; another
  fresh Open dialog can also stop updating its confirmation button.
- Asynchronous native presentation with a temporary default-run-loop probe makes
  Open and Allow Folder Access update and complete. Actual sandbox app loads the
  selected second.jpg with `(2/2)`. Native trace shows file-reference selection
  and expected path standardize equally. Temporary logging/probe loops are
  removed from the final completion-based implementation.
- Moved-bookmark test observed red: chooser received old parent rather than the
  resolver's current parent. Authorizer now resolves the selected file before
  constructing or comparing any proposed folder grant.
- The separate diagnostic app's folder grant was blocked by automatic approval
  review as outside scope. It was cancelled; qualification instead used PicFetch
  with only generated fixture images and the authorized production flow.

- Final completion-based sandbox native rerun: exact parent with spaces accepted;
  window title is first.jpg `(1/2)`. Cancelling a separate fresh fixture closes
  the dialog, yields a single-image title and Right cannot reach second.jpg.
- Focused filepicker race suite and actual appleappstore-tagged suite pass.
  GoLand Project inspections of all three changed Go files, weak warnings
  included, report no findings. `make verify-build` passes.

Native Save also accepted a new destination and wrote a valid 981-byte PNG.
Computer Use capture refreshes then timed out. A native thread sample shows
normal GLFW event polling on UI and the following Open request waiting only on
its tracked worker, so no post-Save native-modal deadlock is present. Remembered
native reuse remains unverified until capture service recovers.
Latest-head hosted gates and the fresh deliverable package remain pending.

Windows CI at d402f38 exposed a test assertion comparing URI forward slashes to
native TempDir backslashes. The raw native-guards artifact confirms both paths
name the correct resolved `002` folder. The assertion now compares filepath.Clean
values; no production behavior changed. Fresh Windows CI must verify the correction.

### Final code gate and local test package

Verified code revision: f36f1cade1504a3f126c564be91f7554e098b710.
[Fresh code review](https://github.com/frathe/picfetch/pull/75#issuecomment-5927983355)
and the [separate security-focused round](https://github.com/frathe/picfetch/pull/75#issuecomment-5928247620)
report no findings after all eight thread dispositions. Recursive scan work
budget is explicitly deferred availability hardening, not an implemented fix.

[Full CI](https://github.com/frathe/picfetch/actions/runs/36838100555),
[CodeQL](https://github.com/frathe/picfetch/actions/runs/36838100685) and FOSSA pass.
Open PR CodeQL alerts are empty. [Qodana](https://github.com/frathe/picfetch/actions/runs/36838100783)
artifact 11150212850 has this exact revision and 12 post-suppression SARIF
results, all unused-export false positives: the prior ten plus
StripJPEGMetadataVerified (EXIF constructor) and ReadAndProbeSnapshot (metadata
worker). Production callers were checked; no actionable findings remain.
Original local inspection scope/profile and its loader IDE-tag limitation remain
as recorded; tagged compiler checks and configured Qodana verify that source.

Universal ad-hoc sandbox package: version 1.1.11, build 483,
bin/apple-store-e2e-2026-10-01-f36f1ca/PicFetch.app. Native signatures,
entitlements, dependency closure, deployment minima, resources and seals pass;
all 24 manifest payload hashes were independently checked. No submission,
distribution signing or merge occurred. The manifest preserves the original
source/worktree provenance; later evidence-only commits carry forward unchanged
code results with this revision stated.
The exact final package starts Open With consent at the correct fixture folder
(including spaces). Return and a mouse click on Allow Folder Access for the
selected exact directory both dismiss it and load (1/2) or (2/2). The grant
reuses for a sibling and survives a complete quit/relaunch. Cancel opens only
the selected file, Right cannot reach a sibling, and reopening prompts again.
Direct diagnostic runs quit normally with status zero. An initial capture attempt
was inconclusive (UI timeout/exited instance); subsequent controlled repeats
qualified these effects. Final-bundle direct Finder drag remains unqualified;
native Open/Save was qualified during implementation and the shared authorization
path plus moved-parent regression remain covered.
