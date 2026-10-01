# Fixed screenshot dimensions

Route: Deep (terminal input, launch, UI composition and native desktop geometry).
Owner: Pico, inline; zero delegates because the contracts share hot context.

Ronin requests `-fixed-size-mode`, an Up/Down and Return console picker for
1280 x 800, 1440 x 900, 2560 x 1600 and 2880 x 1800. The selected dimensions
measure physical pixels of the entire window, including the title bar, excluding
the screenshot shadow. Every PicFetch window uses that size for this invocation.
Panels, image loads, maximize, picture-frame and Spiral cannot change it.
Saved main and secondary geometry must survive the session. Native OS panels
remain managed by the OS. Resolutions larger than a screen require a suitable
display for unobstructed screen captures.

No dependencies are added: Fyne v2.8.0, x/term and x/sys are already shipped;
existing notice delivery remains unchanged. Mac frame/content conversion follows
[AppKit's window geometry contract](https://developer.apple.com/documentation/appkit/nswindow/contentrect%28forframerect%3A%29?language=objc).

## Acceptance and tasks

Task graph: 1 -> 2 -> 3 -> 4. All tasks owned by T0 inline.

1. CLI and console: `launch.Resolution`, four value presets, flag parsing,
   optional `=WIDTHxHEIGHT` for scripts, raw terminal selection with restoration
   and cancellation before desktop effects. Files: internal/launch, consolehelp,
   main startup and existing tests. Verify focused parser, picker and startup
   tests with `go test -tags no_emoji,nodynamic ./internal/launch ./internal/consolehelp .`.
2. Window constraint: application decorator in internal/screenshots routes all
   NewWindow calls to a constrained window, prevents resize/fullscreen, contains
   content minimums, preserves native/desktop operations and measures native
   decorations/backing scale. Native metrics stay in winpos build-tag files;
   native maximize honors fixed windows. Verify screenshots and winpos tests;
   native macOS capture measurements for all four presets.
3. UI: construct the decorator before feature windows, retain original persisted
   geometry in launchOverride. Verify main/secondary geometry preservation and
   mode transitions using existing root harness; maintain shard assignments,
   exact Qodana exclusions and architecture map. Focused root UI tests plus
   `make check-test-shards`.
4. Land: `make verify-build`, affected race regressions, GoLand inspections of
   all changed code (weak warnings included), commit/push, fresh GitHub code and
   security review, CI, CodeQL and post-suppression Qodana. Full race suite runs
   on native Linux/amd64 CI. Then make a fresh local universal sandboxed App
   Store package and deliver its path and console command. Never merge/upload.

Budgets: zero spawns; one local lead assessment per task, further rounds only
for findings; focused suites locally, full suite once on hosted CI per revision.
The completed work and verification evidence are archived here. The implementation
sequence below is retained as history; final code qualification is recorded at the end.

## Historical local evidence before the first review, 2026-10-01

- Parser, picker, startup and constrained-window regressions were observed red
  before implementation (unknown flag, zero selection, missing selected startup
  size and unconstrained window respectively), then green.
- Root UI regression was observed red at saved 700 x 500 instead of the requested
  1280 x 800. It now covers image loading, Grid, picture-frame, secondary windows,
  preservation of all saved geometry and persistence of unrelated settings.
- Deliberately removing original geometry restoration and fixed-window native
  maximize guards made their respective tests fail. Both mutations were restored.
- Focused race command: `go test -race -tags no_emoji,nodynamic
  ./internal/launch ./internal/consolehelp ./internal/screenshots ./internal/winpos
  . ./internal/ui -run 'TestFixed|TestResolutionPicker|TestWindowConstraint|TestLaunchOptions'
  -count=1` passes. The first race run exposed an unjoined sort in the new test;
  the test now waits on the existing sort completion signal before reading state.
- Real executable tested with `.scratch/fixed-size-mode/terminalcheck.py` and a
  pseudo-terminal: Down/Return selects 1440 x 900; Ctrl+C and SIGTERM abort with
  status 1; raw terminal flags, cursor and alternate screen restore. Darwin's
  transient PENDIN bit is excluded from comparison; all other fields match.
- Native AppKit frame measurements after resize/fullscreen/maximize and large
  content attempts: exactly 1280 x 800, 1440 x 900, 2560 x 1600 and 2880 x 1800
  physical pixels on the 2x Retina display. A fractional Fyne scale (requested
  1.25, reported 1.3) also preserves exactly 1280 x 800. Measurement probe and
  JSON records are retained locally under .scratch and /private/tmp. Direct
  `screencapture` could not create a window image, so these are native frame
  measurements, not PNG qualification claims.
- GoLand per-file inspections (Project profile, errorsOnly=false) covered every
  changed Go file, including all platform variants and tests. Weak import-order
  and unnamed-field warnings were fixed and affected files re-inspected. No
  actionable findings or timeouts remain. This is the documented IDE fallback,
  not a local Qodana scan; fresh hosted post-suppression SARIF remains required.
- Windows amd64 cross-compilation and vet pass for screenshots, winpos and
  consolehelp. Native Windows/X11 rendering is unverified; no such pass is claimed.
- `make verify-build` passes. Docker is unavailable locally, so canonical shard
  validation and the full native Linux/amd64 race suite are delegated to CI.
  A direct macOS shard attempt sees the pre-existing Darwin-only menu test and
  therefore cannot qualify the canonical Linux inventory. The added test is
  assigned to ui-1, count updated, and the new test path excluded exactly in Qodana.

| Task | Spawns budget/actual | Lead review rounds | Full suite |
|------|----------------------|--------------------|------------|
| CLI / picker | 0 / 0 | 1 | no |
| Window policy / native metrics | 0 / 0 | 1 | no |
| UI / persistence | 0 / 0 | 2 (fixture completion fix) | no |
| Land | 0 / 0 | clean code/security at f36f1ca | hosted CI passed at f36f1ca |

At this initial stage, the PR also contained security-priority follow-up threads
on metadata removal/export identity. Both were subsequently fixed in a664aad
before the fresh package was built. Recursive scan limits were classified by the
bot as deferrable availability hardening and remain an open backlog item.

## Historical review fixes after 1439746

Four confirmed code-review findings are fixed inline. Signal notification now
starts before any raw terminal mutation and stops after cursor, terminal and
ANSI restoration. Fixed windows retain ordinary Fyne padding. Unknown native
chrome still reports a limitation, but fallback sizes divide by canvas scale.
Content layout refresh checks native dimensions as well as Fyne scale, and queues
one correction on UI; hidden/closed windows discard it, with no new workers.

Padding and 2x fallback tests were observed red (padding removed and 2560 x 1600
instead of 1280 x 800). Tests now cover native/non-native fallback and 2x -> 1x
changes. A separate simulated native 2x -> 1x -> 2x backing transition keeps
Canvas.Scale at 1 and preserves the physical target. Disabling correction made
that guard fail at 640 x 400; it was restored and the race suite passes.

Focused screenshot/console/root UI race regressions pass. All four actual Retina
native frames still equal their selected pixels after forbidden resize,
fullscreen/maximize and large-content attempts. The real executable passes the
pseudo-terminal Ctrl+C, SIGTERM and Down/Return restoration checks. Windows
cross-compilation passes. GoLand inspections of all three changed Go files,
including weak warnings, are clear after fixing one inferred-type warning.
`make verify-build` passes. Actual mixed-monitor movement remains unqualified;
its independent backing conversion is covered by the per-instance native-metric
seam test. The README states the unknown-chrome limitation explicitly.

Hosted evidence for 1439746: full platform/race CI and CodeQL pass; artifact
11146375885 from Qodana run 36828289980 has the exact revision and 10 final
post-suppression SARIF results, all unused-export false positives with production
callers (the prior eight plus ScreenshotContentSize and SelectResolution).
That review found the four issues addressed above. The required clean code and
security rounds subsequently completed at f36f1ca, as linked below.

### Historical review: termination during picker restoration

Thread 4153347460 found that a signal arriving after Return could remain unread
when deferred `signal.Stop` ran. The final cleanup now stops delivery and checks
pending notifications after cursor/raw/ANSI restoration; arrival during a terminal
read also prevents accepting Return. A deterministic regression reproduced the
selection/read race before the fix; cleanup covers both SIGINT and SIGTERM.
Focused race, PTY restoration and changed-file inspections recorded with the fix.

### Qualified code gate and local test package at f36f1ca

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
The final Store app and the standalone native sizing fixture were captured by
Computer Use; both actual JPEG screenshots measure exactly 1280 x 800 pixels,
including titlebar and excluding shadow. Evidence is retained in
.scratch/folder-access-regression/final-store-approved.jpeg and
.scratch/fixed-size-mode/final-1280x800.jpeg. Existing four-preset native frame
measurements and scale/backing regressions remain valid; mixed-monitor movement
is still unqualified.

### Completion-record maintenance

The final documentation review at 2e392a9 identified active-directory placement
and stale historical gate wording. This completed record is now archived under
finished_refactorings, with todo links updated and earlier snapshots labeled or
reconciled. Application code and the qualified test package remain unchanged.
Fresh review and CI results for subsequent documentation revisions are posted
in PR #75; original local code evidence retains its analyzed revision above.

### Subsequent review fix: animated help restoration

Security-focused thread 4154399879 at c8b5b4e found the same late-SIGTERM
window in animated help. The deterministic final-write callback reproduced
success after SIGTERM before this fix; disabling the new pending-notice guard
reproduced both termination cases again. Ctrl+C remains a successful skip, even
during cleanup, but cannot mask a queued SIGTERM. Screen cleanup checks notices,
and Write stops delivery and checks again only after platform ANSI restoration.
Signal registration now precedes ANSI setup. No workers, dependencies, test files
or top-level root UI tests were added; existing exact Qodana exclusion applies.

Focused console races and startup help/launch regressions pass after restoration.
Windows console cross-compilation, make verify-build and make build pass. Actual
PTY normal completion, Ctrl+C and SIGTERM confirm correct exit status, cursor,
alternate screen, complete help and unchanged terminal settings. Evidence logs
are /private/tmp/picfetch-help-cleanup-{red,negative,race,verify,pty}.log and
/private/tmp/picfetch-help-startup-race.log. Initial broader local commands hit
a sandbox cache-access error; the authorized cache-enabled retry passed.

GoLand Project per-file inspections (errorsOnly=false, including weak warnings)
re-ran on both changed Go files after the negative mutation was restored: zero
findings and no timeouts. Analyzed worktree parent is c8b5b4e; file digests:
- internal/consolehelp/help.go: 2a26aec0b473d7b816c4d9eb1ecf181591e51bb588cfa70bfb7e965b26ede711
- internal/consolehelp/help_test.go: c4c579aa475b198d8f5a16d40095beac8f8bb56d5918474408442e954dd0497f

The f36f1ca bundle and qualification above remain evidence for that source, not
for this new help change. Latest-head code/security rounds and hosted checks
follow this fix; final results are recorded in PR #75. Refresh the local App
Store test package after the new code gate clears.

### Subsequent review fix: independent termination delivery

Thread 4154549312 reproduced a full interrupt queue dropping the following
SIGTERM. Help now captures SIGINT and SIGTERM in independent per-invocation
buffered channels. Ctrl+C remains successful; a queued termination retains its
own delivery capacity and is checked during and after terminal restoration.
The saturation case failed before this fix; deliberately sharing the two queues
again broke successful skips and SIGTERM reporting, then restoration passed.
Focused console races and GoLand Project inspections of help.go/help_test.go
(errorsOnly=false, zero findings/timeouts) pass. Latest hosted evidence is
recorded through the ongoing PR #75 review loop; no new dependency or test file.
