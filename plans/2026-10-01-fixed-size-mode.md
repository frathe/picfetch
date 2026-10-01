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
Evidence and actual cost ledger will be appended as work completes. Active work
stays here until Ronin accepts it.

## Local evidence, 2026-10-01

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
| Land | 0 / 0 | pending GitHub loop | hosted CI pending |

The existing PR now also contains fresh security-priority follow-up threads on
metadata removal/export identity. Those confirmed older findings must be addressed
in the requested review loop before the new App Store package is built. Recursive
scan limits are explicitly classified by the bot as deferrable availability
hardening and remain an open backlog item.
