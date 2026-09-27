# MA-028 Linux native-input qualification

Date: 2026-09-27. Requested and assessed by T0 on the local desktop.
Result: the four native scenario groups passed with OS-injected keyboard and
pointer events. This is actual GLFW/XWayland runtime evidence, not Fyne's test
driver. It is **not physical-keyboard operator evidence**; that literal part of
AC10, the other platforms and the external gates remain open in ticket 10.
No production code or dependencies changed during this qualification.

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

Retained local artifacts are in
[`.scratch/ma-028/linux-native-2026-09-27`](../.scratch/ma-028/linux-native-2026-09-27/).
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
surface; menu state was checked on that rendered surface. Full compositor/layout
qualification is not asserted by this command-admission exercise.

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

## Limits and remaining acceptance

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
- No confirmed command-admission defect required a production fix. The earlier
  Make/race and 68-file GoLand evidence remains attached to unchanged code at
  `9dc3a81`; the native binary came from documentation-only descendant `6db8d73`.
  No broad suite or GoLand reinspection was rerun for this evidence-only update.

Delegation: one additional bounded read-only assignment to the existing scout
checked launch-storage isolation and hold seams. T0 performed every native
interaction, artifact inspection, assessment and tracker update. The diagnostic
skill was used to separate inconclusive input-harness attempts from app behavior;
only the throwaway input pacing changed, not production code.
