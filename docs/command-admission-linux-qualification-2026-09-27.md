# MA-028 Linux native-input qualification

Date: 2026-09-27. Requested and assessed by T0 on the local desktop.
**Current status: Linux qualification is complete.** On 2026-09-27 the user
confirmed that the physical-input checks had already been tested successfully
but had not been documented, and that only macOS remains. This retrospective
operator confirmation closes Linux V10 alongside the retained native evidence.
No new Linux run, capture, test timestamp or tested revision is asserted for
that confirmation. The existing Make-built launch is accepted for Linux under
the user's completion instruction; it remains recorded as direct execution
after `make build`, without claiming a literal `make run` invocation.

## Earlier automated qualification and repair record

The limits below describe the original automated sessions. They do not undo
the subsequent operator confirmation above. Windows qualification and fresh
remote gates are recorded in ticket 10 and PR 66. macOS qualification is now
complete in its separate record, and ticket 10 is resolved.

**Repair status: the maximized-window Escape defect is fixed; native
checks, focused regressions, changed-file inspections and `make verify` pass.**
See the repair section below. The earlier diagnostic evidence is retained.
The earlier interpretation of the small rendered surface as a capture limitation
was incorrect; the positive command-admission observations are not an overall
Linux acceptance pass by themselves.

Original scoped result: the four native scenario groups passed with OS-injected keyboard and
pointer events. This is actual GLFW/XWayland runtime evidence, not Fyne's test
driver. It is **not physical-keyboard operator evidence**; that literal part of
AC10, the other platforms and the external gates were open at this handoff.
The initial qualification changed no production code or dependencies; the later
authorized repair is recorded separately below.

## Revision and environment

- Clean build revision: `6db8d73cd1c2ce924691aea3921b0f38f5c436bb`, including
  implementation `9dc3a81`; `go version -m bin/picfetch` reported
  `vcs.modified=false`, Go 1.27.1, linux/amd64, cgo, `no_emoji,nodynamic`.
- Binary SHA-256: `af0c125d21a06ade28cc9e628ac986110e5fbbc049502a38fec475b5e15b7130`.
- Ubuntu 24.04.5 LTS, kernel 6.8.0-142-generic x86_64, GNOME Wayland session,
  XWayland `DISPLAY=:1`, `WAYLAND_DISPLAY=wayland-0`; Fyne v2.8.0.
- `PATH=/snap/go/current/bin:$PATH make build` passed. The resulting native
  `./bin/picfetch` was launched directly to supply fixture argv. This is the
  Make-built production binary, **not a literal `make run` invocation**; both
  Make targets use the same application tags and generated inputs.
- Normal application identity retained. XDG config/cache/data were isolated
  under `/tmp/picfetch-ma028-linux-uocSPo`; no real collection, Favorite or
  standing preference was changed. The desktop clipboard was used for fixture
  text/images. Fixtures were copies of `internal/ui/testdata/explorer/astronaut.png`
  and `coffee.png`, not original user images.

Optional local artifacts are in
`.scratch/ma-028/linux-native-2026-09-27` on the original workstation.
They are intentionally ignored tracker evidence, not shipped assets. The directory
contains the harness source/binary, launcher shim, build/app logs, selected
screenshots, fixture/payload files, `payload-checks.txt` and `SHA256SUMS`.
The temporary profile and original captures remain at the path above. An
inconclusive pre-test clipboard capture was removed without being included in
the retained evidence.

## Input and hold mechanism

`native-input.c` uses Xlib/XTEST, targets only the owned visible window, raises
and focuses it, checks focus, then sends key down/up or mouse events with 20 ms
event pacing. It does not call PicFetch functions or Fyne test APIs. Native close
is a `WM_PROTOCOLS/WM_DELETE_WINDOW` request. The source hash is
`fe25051a13370ae7145403270beeca93ea447904dbe030fd83ae5a9d356b5adf`.

The process-local `PATH` puts a qualification-only `wl-copy` launcher first.
It calls the real `/usr/bin/wl-copy` with unchanged arguments and publishes the
real PNG, then waits on a FIFO when the hold flag exists. A 300-second timeout
bounds that wait. PicFetch therefore remains in its real clipboard-worker busy
state until release. This is an external-command completion hold scoped to this
child process, **not an instance encoder override**. No application seam,
global environment, installed binary or clipboard implementation was changed.

Exact build/launch and helper forms used (from the repository root):

```sh
PATH=/snap/go/current/bin:$PATH make build
gcc -Wall -Wextra -Werror /tmp/picfetch-ma028-linux-uocSPo/native-input.c \
  -lX11 -lXtst -o /tmp/picfetch-ma028-linux-uocSPo/native-input
env XDG_CONFIG_HOME=/tmp/picfetch-ma028-linux-uocSPo/config \
  XDG_CACHE_HOME=/tmp/picfetch-ma028-linux-uocSPo/cache \
  XDG_DATA_HOME=/tmp/picfetch-ma028-linux-uocSPo/data \
  PATH=/tmp/picfetch-ma028-linux-uocSPo/shim:/snap/go/current/bin:$PATH \
  ./bin/picfetch -- /tmp/picfetch-ma028-linux-uocSPo/images/first.png \
  /tmp/picfetch-ma028-linux-uocSPo/images/second.png

# Window ID observed with wmctrl -lp for this run only; re-resolve on a new run.
/tmp/picfetch-ma028-linux-uocSPo/native-input 0x0320002b key Control_L c
/tmp/picfetch-ma028-linux-uocSPo/native-input 0x0320002b key Alt_L Shift_L f
/tmp/picfetch-ma028-linux-uocSPo/native-input 0x0320002b type ma028_editor_marker
/tmp/picfetch-ma028-linux-uocSPo/native-input 0x0320002b key Control_L a
/tmp/picfetch-ma028-linux-uocSPo/native-input 0x0320002b key Control_L c
wl-paste --no-newline --type text/plain
import -window 0x0320002b /tmp/picfetch-ma028-linux-uocSPo/screens/05-editor-selectall.png
```

Run inputs separately and observe the completed transition before the next
dependent action. `XSync` flushes X requests; it is not a PicFetch worker/render
completion barrier. Inspect actual payloads and screenshots, not launcher exit
alone. The first zero-delay/overlapping input attempts were inconclusive; the
confirmed observations below used paced events and settled state. Initial
capture names such as `04-editor-selected.png` or `07-favorite-confirmed.png`
are not assertions and are deliberately excluded from the retained pass set.

## Observed scenario results

Screenshot numbers below refer to `screens/` in the retained artifact directory.

| Scenario | Native steps and actual observation | Evidence |
| --- | --- | --- |
| Positive editor Copy and Select All over an image | Open Add to Favorites with Alt+Shift+F; type `ma028_editor_marker`; Ctrl+A visibly selects the complete Entry; Ctrl+C produces exactly that text, not image data. Image remains first of two. | 03, 05; `editor-copy-confirmed.txt`; payload check |
| Same editor ownership over Grid | G, Alt+Shift+F; type `ma028_grid_editor`; Ctrl+A/C yields that exact text. Escape closes the dialog. Underlying two-item Grid with its original selection is pixel-identical before/after. | 12, 13, 14; ImageMagick AE = 0 |
| Admitted image intent, shortcut and explicit menu | Ctrl+C on the image and separately Actions -> Copy image produce 512 x 512 PNGs. Each decoded image is pixel-identical to `first.png`; the two encoded clipboard PNGs also match. | 11; `shortcut-copy.png`, `explicit-copy-confirmed.png`; AE = 0 |
| Real modal refusal and confirmation | While Favorite naming owns input, Ctrl+R does not reveal/change the image, Ctrl+Shift+C does not replace the editor clipboard text, and clicking the underlying Actions menu cannot open it. Clicking the dialog's Add button saves precisely the two fixture paths in the isolated Favorite. | 06, 09; isolated `file-list.json` |
| Busy menu availability and invocation refusal | Rotate the image so Save is meaningful; activate Copy Selection, draw a rectangle, Return, observe held launcher. File Open/Save/Export/Close/Settings and Help items are disabled. Actions' unrelated items are disabled; permitted Zoom and Copy controls remain available. Clicking disabled Save does nothing; Ctrl+S produces the expected finishing-copy feedback. Source hash is unchanged. Ctrl+Shift+C preserves the cropped PNG. | 18, 19, 20, 22, 23; `region-copy.png` is 189 x 166; byte-identical `region-after-refusal.png` |
| Busy exit and positive recovery | Release the FIFO; region overlay ends. In a second observed hold/release cycle, File Save and other applicable commands return enabled without reopening the collection. Ctrl+Shift+C then produces the exact fixture path. | 26 -> 27; successful path assertion |
| Delete/export owner controls | Shift+Delete opens a real delete card focused on Cancel. Ctrl+C and Ctrl+E preserve the card and fixture-path clipboard. Return activates Cancel, leaving the source untouched. Ctrl+E opens the export card; Ctrl+C/Shift+Delete cannot bypass it; Right moves focus from PNG to JPEG, not to another image. Escape cancels export. | 29, 30, 32, 33; unchanged source hash and clipboard assertions |
| Native close while busy | With held launcher PID 378298 observed, send native close to app PID 372857. App exits with code 0 and its window disappears while launcher 378298 is still held. Release that launcher only afterward. | Process/session observations recorded below |
| Native close while modal | Relaunch the isolated profile (PID 378392), cancel the export card, open Add to Favorites, observe the actual modal, then native close. App exits with code 0. | 34; process/session observations |

The first hold's drag was `(610,260)` to `(1010,610)` in the maximized image.
Later cycles used `(180,180)` to `(400,420)` in the restored logical image
surface. Menus were opened by native pointer input, e.g. File `(31,16)` and
Actions `(195,16)`; explicit image Copy was `(228,540)` in the original layout.
These are run-specific coordinates, not a portable scripted test. Captures after
the maximized Grid visit retained a larger X drawable around the smaller image
surface; menu state was checked on that rendered surface. The user subsequently
confirmed the mismatch on the actual desktop. It is a real defect, not grounds
to exclude the window state from qualification; see below.

The retained shim contains the exact FIFO protocol. During each meaningful
busy check, a fresh launcher process was observed before testing refusal/close;
a leftover `hold-entered` marker alone was not used as readiness proof. Release:

```sh
pgrep -a -f '^timeout 300 sh -c read -r signal'
# Only after observing the owned FIFO reader:
timeout 3 sh -c 'printf "release\n" > /tmp/picfetch-ma028-linux-uocSPo/release.fifo'
```

The close test specifically observed PID 378298 both before close and after the
app session returned exit 0; `wmctrl -lp` no longer contained app PID 372857.
The second app session also returned exit 0. Final checks found neither test
window nor held timeout process. Both source fixture hashes were unchanged;
no Trash action was confirmed and no Save/export was committed.

Payload checks were rerun against retained artifacts: editor text matched,
both image copies had zero differing pixels, Grid before/after had zero
differing pixels, and the refused Copy Path preserved the region PNG bytes.
`sha256sum -c SHA256SUMS` can recheck artifact integrity.

## Limits at the original automated handoff

- No hardware keyboard/mouse operator supplied these events. AC10's literal
  physical-shortcut checks and `make run` procedure are not silently waived.
  The observed Linux runtime groups are checked separately in ticket 10.
- This is GNOME Wayland with an XWayland GLFW app, not native Wayland-client,
  other desktop, Windows or macOS evidence. AppKit compilation/runtime remains
  unverified. No PR, push, CodeQL or Qodana-CI/SARIF result is claimed.
- `/tmp` is on ext4 without usable casefold support. Enabling `+F` on the new,
  empty qualification-only `casefold` directory returned `Operation not
  supported`. No filesystem mount/format or host configuration was changed.
  `TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` therefore retains its
  earlier case-sensitive-filesystem skip; it did not pass here.
- Logs contain startup locale-C parsing and Fyne threading-model warnings.
  They contain no new panic or clipboard failure in the confirmed runs. These
  warnings were not resolved or counted as a clean static-analysis result.
- The window-reset defect below required the subsequent repair. The earlier
  Make/race and 68-file GoLand evidence remains attached to unchanged code at
  `9dc3a81`; the native binary came from documentation-only descendant `6db8d73`.
  No broad suite or GoLand reinspection was rerun for this evidence-only update.

Delegation: one additional bounded read-only assignment to the existing scout
checked launch-storage isolation and hold seams. T0 performed every native
interaction, artifact inspection, assessment and tracker update. The diagnostic
skill was used to separate inconclusive input-harness attempts from app behavior;
only the throwaway input pacing changed, not production code.

## Confirmed Linux Escape reset defect (diagnostic history)

The user's full-desktop screenshot at 12:03:45 shows that the small lower-left
surface is visible on screen, not just in `import` captures. T0 reproduced it
twice with the same unchanged production binary. The minimal case requires only
one loaded image, ordinary WM maximization and Escape; no Grid, dialog, saved
preferences or clipboard hold is necessary. The blank-launch baseline's native
client target is 624 x 409 here (logical `startW/startH` are 520 x 340).

Artifacts and the bounded agent-runnable geometry assertion are retained under
`.scratch/ma-028/linux-escape-reset-2026-09-27` (optional local evidence).
`results.txt` records the exact outputs; `escape-failure.png` reproduces the user
symptom and `control-pass.png` shows the control. The working directory is
`/tmp/picfetch-linux-escape-b56v16`. Launch for the minimal failure:

```sh
env XDG_CONFIG_HOME=/tmp/picfetch-linux-escape-b56v16/config-minimal \
  XDG_CACHE_HOME=/tmp/picfetch-linux-escape-b56v16/cache-minimal \
  XDG_DATA_HOME=/tmp/picfetch-linux-escape-b56v16/data-minimal \
  ./bin/picfetch -- /tmp/picfetch-linux-escape-b56v16/fixture/only.png
# In another terminal; re-resolve the owned window ID on another run:
bash /tmp/picfetch-linux-escape-b56v16/repro.sh 0x0320002b 624x409
```

The script maximizes with `wmctrl`, observes both native maximization atoms,
sends Escape through the retained XTEST helper and polls native geometry/state
for up to three seconds. It reports:

```text
before=624x615 expected-reset=624x409
maximized=1920x1131
FAIL: Escape expected 624x409 without maximization; actual=1920x1131
```

Both `_NET_WM_STATE_MAXIMIZED_HORZ` and `_NET_WM_STATE_MAXIMIZED_VERT` remain.
With a fresh profile, removing maximization and observing its completion
**before** Escape (`restore-first` third argument) makes the same assertion pass
at 624 x 409. Removing it only **after** the failure restores the old image
geometry, 624 x 615. A separate native `G`, `G`, Escape control (fourth argument
`grid`) passed, so this diagnosis does not assert that all Grid restores fail.

Cause: `clearToDropzone` calls `undoGridMaximize` and then `Resize` at
`internal/ui/viewer.go:616-622`. The helper returns early at lines 637-639 unless
Grid or Explorer recorded ownership of a maximize. An ordinary WM maximize
therefore never reaches `winpos.Unmaximize`, even though the reset promises the
compact welcome size. Fresh-profile and restore-first controls rule out a saved
size or a general failure to dispatch Escape.

The pinned Fyne/GLFW source explains the smaller drawn surface: Fyne v2.8.0
`internal/driver/glfw/window.go:56-70` immediately resizes the logical canvas,
then invokes native `SetSize` and processes the **requested** dimensions without
waiting for WM acknowledgement. GLFW's pinned X11 `x11_window.c:2204-2219`
sends `XResizeWindow` without clearing maximization; its ConfigureNotify branch
at lines 1486-1500 emits size callbacks only when native dimensions change.
Thus a rejected native shrink can leave the smaller logical canvas visible
inside the still-maximized window, consistent with both screenshots and the
native measurements. These driver details are source-level explanation, not
additional instrumented runtime measurements.

Coverage gap: `TestViewerReset` checks only `v.win.Canvas().Size()` at
`internal/ui/reset_test.go:63`, under the Fyne test driver. It cannot assert
native maximization or catch the logical/native mismatch. A fix needs a red
regression covering native restore-before-resize behavior, plus this actual
desktop loop. Preserve static-size semantics and repeat both ordinary and
Grid-owned maximize scenarios; do not claim a canvas-size assertion alone fixes it.

The relevant reset/maximize guard predates MA-028 by source history; no old
revision binary was run, so this is not a completed runtime bisection. No
production code was changed or fix applied in this diagnostic follow-up. All
five isolated diagnostic app sessions exited normally, and the failed
empty-window capture process was stopped. One additional read-only scout traced
the pinned driver sequence; T0 verified citations and owned all native probes,
diagnosis and record corrections. The diagnosing-bugs workflow stops at the
confirmed cause at that handoff; the user subsequently authorized repair below.

## Authorized repair and verification

The user's "continue the work" authorized the fix from base `8d68cd7`.
Dynamic `clearToDropzone` now requests native unmaximization even when Grid or
Explorer did not initiate it, before requesting the welcome size. The existing
owned-maximize path still clears its ownership and restores its remembered
position; its boolean result prevents a duplicate native restore request.
Ordinary image load/zoom policy is unchanged. Fixed-size reset still preserves
an ordinary maximized window and still restores Grid-owned maximization to the
pre-Grid fixed size. No worker, package, platform implementation, dependency or
user-visible string was added.

The per-viewer `unmaximizeWindow` boundary is initialized to the existing
`winpos.Unmaximize` operation during normal construction. The new regression
drives real Escape/Close Files with a window-manager fake that rejects Resize
while maximized, even while the Fyne canvas changes. Its initial observed red:

```text
native window after Escape: maximized=true size={1600 1000};
want restored 520x340 (logical canvas={520 340})
```

After the fix, `TestEscapeResetRestoresNativeWindow` covers ordinary Escape,
Close Files, Grid-owned reset, fixed ordinary maximize and fixed Grid restore.
A temporary Go build overlay omitting native restoration fails ordinary Escape
and Close Files while the fixed/owned controls pass. The unmodified final guard
passes again. This is a native-boundary regression, not a claim that the fake
replaces real WM qualification.

Native results on the same Ubuntu/GNOME/XWayland desktop, with fresh isolated
profiles and copied fixtures:

| Check | Observed result |
| --- | --- |
| Ordinary WM maximize then Escape | Five consecutive runs: 1920 x 1131/maximized -> 624 x 409/unmaximized welcome |
| Grid -> viewer -> Escape | 1920 x 1131 -> 624 x 409, maximization cleared |
| Original two-image/Grid/rotation sequence then Escape | Compact 624 x 409 welcome restored, no lower-left-only surface |
| Fixed ordinary WM maximize then Escape | Welcome fills the preserved 1920 x 1131/maximized window |
| Fixed Grid maximize then Escape | Configured 1080 x 720 native size restored, maximization cleared |

The native loop was tightened to wait for changed, stable geometry as well as
both maximization flags, and to require the welcome title at completion. An
early probe that saw the flags before the resize was not counted. Selected
screenshots were visually inspected; all owned app processes exited normally.
Input remains OS-injected XTEST, not a physical-keyboard operator claim.

Verification artifacts are retained in
`.scratch/ma-028/linux-reset-repair-2026-09-27` (optional local evidence),
with working output at `/tmp/picfetch-linux-reset-fix-Y5iRmx`.
`repeat-native.sh` launches isolated windows and runs five ordinary plus Grid
and multi-image/rotation cases; `repro.sh` also supports the fixed-size controls.
`red-and-inspections.md` records the original red and exact inspected source
hashes. `negative-guard.log`, `guard-final.log`, focused logs, native logs and
screenshots retain the outputs. The mutation source is qualification-only,
under ignored scratch storage; production sources contain no instrumentation.

- Make native build passed. Tested binary SHA-256:
  `f2ef241cef14ece454d1ef743d4ab5bcbba3cfd12062724e72594a45931cba81`.
- Focused reset/static-size/command-admission union: 28 top-level passes in
  3.367s, including all five new boundary cases.
- Neighbor run: six top-level passes in 3.269s, including Explorer
  `close_files_memory`, `cold_cohort_size`, `return_to_map`, chooser retirement,
  actual Close Files menu, scan/sort Escape and Grid-visible zoom. The first
  selector omitted Explorer subtests; the corrected run explicitly includes them.
- Windows internal-package cross-vet passed with `CGO_ENABLED=0`; this is not
  native Windows qualification.
- GoLand `get_file_problems(errorsOnly=false)` returned no findings or timeouts
  for `viewer.go`, `build.go`, and `reset_test.go` after their final edits.
  No suppression changed. The earlier 65 unaffected Go files retain their
  original evidence; Qodana-CI/SARIF remains a separate, unverified gate.
- Final `make verify`: exit 0, including format/generated inputs/notices, vet,
  build, exact Qodana exclusions/shards and all native Linux/amd64 Docker race
  partitions. Root UI partitions passed in 489.127s (ui-1), 398.887s (ui-2),
  and 404.303s (ui-3); the new boundary test passed under race in 4.920s.
  Artifacts: `.scratch/race-runs/20260927T103730Z-3Tubvu`, plus `verify.log`
  in the repair evidence. Existing explicit skips remain, including the
  case-insensitive export case and native HEIC qualification; they are not
  counted as passed platform evidence.

T0 owns the regression, fix, native probes and all review. One further bounded
read-only scout assignment identified neighboring static-size/ownership tests;
no implementation or review was delegated. Platform/physical-input and external
CI limitations remain. The case-insensitive export gap was closed below.

## Case-insensitive export qualification

At code revision `75fd69ef5460e39bef8916765ea901a60f1f7045`,
`TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` passed on an actual
case-insensitive FAT16 filesystem mounted by Linux's `vfat` driver. This closes
the previously skipped filesystem-dependent AC9 observation; it does not qualify
Windows/macOS input or their native filesystem adapters.

T0 created a new 64 MiB image under `/tmp/picfetch-ma028-casefs-reoIdM`, formatted
only that file with installed `mkfs.vfat` 4.2, and attached it with UDisks.
The owned image was verified as `/dev/loop203`, mounted at
`/media/<user>/MA028FAT` (user component redacted) with `nosuid,nodev,noexec`,
and used only for the test's
`TMPDIR`. The ordinary and race test binaries were built on the normal workspace
filesystem; production code and the regression were unchanged.

```sh
PATH=/snap/go/current/bin:$PATH go test -tags no_emoji,nodynamic -c \
  -o /tmp/picfetch-ma028-casefs-reoIdM/ui.test ./internal/ui
PATH=/snap/go/current/bin:$PATH go test -race -tags no_emoji,nodynamic -c \
  -o /tmp/picfetch-ma028-casefs-reoIdM/ui-race.test ./internal/ui
# From internal/ui, after confirming the test image is mounted:
TMPDIR="/media/<user>/MA028FAT/ma028tmp" /tmp/picfetch-ma028-casefs-reoIdM/ui.test \
  -test.run '^TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset$' \
  -test.v -test.count=1 -test.timeout=2m
TMPDIR="/media/<user>/MA028FAT/ma028tmp" /tmp/picfetch-ma028-casefs-reoIdM/ui-race.test \
  -test.run '^TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset$' \
  -test.v -test.count=5 -test.timeout=2m
```

The ordinary run passed in 0.34s. All five race repetitions passed (1.40s,
0.90s, 0.89s, 0.89s, 0.90s), without skips or race findings. The test independently
requires `Photo.png` and `photo.png` to identify the same file, exports a rotated
8 x 16 image through the alternate-case path, and verifies that reset retains
the written 16 x 8 pixels and clears rotation.

Earlier setup attempts are not passes: unprivileged `lowntfs-3g` mounting was
denied; the desktop-mounted kernel `ntfs3` filesystem remained case-sensitive
and correctly skipped the test; the first FAT attempt ran before that filesystem
was mounted and failed at temporary-directory setup. Only the confirmed `vfat`
runs above count. The existing locale-C startup warning remains unrelated and
was not represented as a clean static-analysis result.

The FAT mount was normally unmounted and its owned loop device detached.
Final checks found neither test mount nor a loop203 backing file. No existing
disk, global configuration, application preference or user image was changed.
Logs, hashes, setup history and cleanup checks are retained in
`.scratch/ma-028/case-insensitive-2026-09-27` (optional local evidence).
The full Make and GoLand evidence from the preceding repair carries forward for
unchanged code; no redundant full suite or new inspections were claimed.

### Native CI selection gap at the filesystem-only handoff

Read-only GitHub checks found no PR or workflow run for this branch. A bounded
read-only scout mapped native-suite selection; T0 verified the cited sources.
The Windows and macOS jobs invoke `scripts/nativeguards`, whose package lists
omit `internal/ui`. Therefore neither runs this case-alias test, and macOS also
misses `TestSetMenuItemModifierMask_ClearsDefaultCommand`, which contains MA-028's
Copy key-equivalent assertion. The runner rejects missing/skipped evidence only
for declared guards, so current CI does not reject these omissions.

A focused native UI suite is needed without including Linux-only golden tests.
The proposed regression seam is the existing nativeguards command boundary:
assert exact selection plus rejection of absent/skipped guard results. That
new seam awaits the requested TDD confirmation; publishing a PR and starting
the CI/review workflow also await explicit permission. Neither change was made
in this evidence-only follow-up. Physical-input and native Windows/macOS runtime
qualification remain separate open gates.

### Focused native CI suite implemented (base `574cf82`)

The user's subsequent request to continue ticket 10 accepted the proposed test
boundary. T0 implemented and reviewed `nativeguards -suite command-admission`
and added it to both Windows/macOS jobs. It selects only root UI's case-alias
export guard, plus `TestSetMenuItemModifierMask_ClearsDefaultCommand` on Darwin;
it does not select Linux goldens or HEIC qualification. Both inventory and
execution must include each required guard. Missing results, skips (including
children), failures and use of the Windows HEIC exemption are rejected.
The jobs retain the captured JSON through the existing always-run artifact step.
Existing full-package platform suites retain their selection.

Observed TDD and negative controls:

- Selection first failed on all three supported hosts with `unknown native
  suite "command-admission"`, then passed after the focused suite was added.
- The waiver regression first failed because the new Windows suite accepted
  the HEIC exception. Restricting the exception to Windows/Store suites fixed it.
- The workflow regression first failed for both native jobs with zero focused
  commands; adding the two steps passed. The test checks each job and its
  unconditional command/capture plus always-run artifact retention.
- Go build overlays deliberately removed required guards or the focused filter.
  The incomplete-evidence cases and all three host-selection cases respectively
  failed. Production sources were not mutated by these controls.
- `go test -race -count=1 ./scripts/nativeguards -v` passed after integration
  (1.058 s). Windows/amd64 no-cgo cross-vet of this runner also passed; this is
  tooling coverage, not Windows desktop or AppKit qualification.

The actual new CLI, not a fake runner, was also exercised twice:

1. Normal ext4: exit 1, correctly rejecting a skipped case-alias guard
   (`run=1 pass=0 rejected=true`).
2. Temporary FAT16/vfat: exit 0 with the required guard run and passed. The
   pre-existing disposable image was attached as `/dev/loop203`, verified against
   its backing file and automounted at `/media/<user>/MA028FAT` (redacted).
   This attachment
   had `rw,nosuid,nodev,...,showexec`; it was not the earlier `noexec` mount.
   It was unmounted after qualification and the auto-cleared device was confirmed
   released. No existing filesystem or user profile was changed.

An initial FAT runner attempt skipped because this host's Go 1.27 gives
`GOTMPDIR` precedence over `TMPDIR` in `testing.TempDir`. The compiler was kept
on ext4, and a test-execution wrapper unset only `GOTMPDIR` before `exec "$@"`.
The compiled runner then ran with `TMPDIR` on the FAT test directory and
`GOFLAGS=-exec=<wrapper>`. The failed initial attempt is retained separately;
only `native-vfat-final.json`/`.log` establish the passing observation. The
wrapper is qualification-only and is not part of production or CI.

GoLand's `get_file_problems(errorsOnly=false)` inspected both changed Go files
and the workflow, including weak warnings, with no findings and no timeout.
It used active IDE inspections; the tool does not report the profile name.
Exact analyzed SHA-256 values:

| File | SHA-256 |
| --- | --- |
| `scripts/nativeguards/main.go` | `33e6c3eaf525226771feab56a8612054fd9241a4728df3f85e11105852e562fe` |
| `scripts/nativeguards/main_test.go` | `c49f7ae7592229046fa9cda8d0383b56133aeeefe4724b2ed177111ef11e37e1` |
| `.github/workflows/ci.yml` | `96f716487c560f9dcfd29d3079aa3bb5206a42f9cf91f10ab413e409f9d85d80` |

This is the documented IDE inspection fallback, not a fresh Qodana SARIF pass.
The workflow test reuses pinned YAML v3.0.5 only in tests; its existing MIT /
Apache-2.0 license and notice were reviewed. No new dependency, shipped runtime,
notice obligation, package, UI string or test-shard assignment was introduced.
The existing exact Qodana exclusion already covers `main_test.go`.

The final `make verify` exited 0 for this source/workflow revision: formatting,
asset/notice checks, vet, build and all native Linux/amd64 Docker race partitions
passed. Root UI partitions passed in 476.803 s, 380.036 s and 390.196 s.
Existing platform/privilege/opt-in skips are preserved in the raw events; the
case-alias test skips on Docker's ordinary filesystem and is qualified by the
separate executed FAT pass above, not by that skip. The runner test package also
passed in the complete Docker race suite. Final source hashes still match the
three inspected files; Markdown link checks and `git diff --check` pass.
Raw local logs and setup details are retained under
`.scratch/ma-028/native-ci-2026-09-27`; full race artifacts are under
`.scratch/race-runs/20260927T113110Z-p0dfVe`. These ignored artifacts are optional
local audit material, not files available in another checkout.

Native Windows/macOS execution still has not occurred. Root UI's test harness
uses Fyne's test application; the Darwin guard checks isolated native menu
objects, not a running desktop's physical Cmd+C route. Neither the focused
suite nor its Linux qualification substitutes for the spec's physical-input
procedure. The user subsequently authorized publishing this branch and opening
a PR with the portable tracker move. Remote results remain unverified;
ticket 10 and the overall acceptance gate stay open.
