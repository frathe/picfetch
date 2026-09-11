# Native comparison renderer smoke, 2026-09-07

Accepted native coverage for ticket 26: actual macOS Retina rendering and native
Windows ARM64 rendering at reported window DPI 96 / monitor scale 100%. The
Windows continuation supplies user-operated linked/unlinked pan and divider
drag; the native held-work run verifies close and quit. Coverage is bounded to
these recorded environments; Windows x64 package startup failures remain open
under ticket 27.

Environment: macOS 26.6.2 (25G83), darwin/arm64, Go 1.27.1, Fyne 2.8.0, Apple M5 Max (40 GPU cores), built-in Liquid Retina XDR Color LCD at 3456×2234 Retina. No standard-density external display is available. Screenshot dimensions describe the captured window, not the physical framebuffer. `FYNE_SCALE` changes user scale and cannot replace a physical standard-density test.

## Repeatable setup

Run from the repository root. The temporary bundle changes only main's app ID through a Go build overlay, keeping test preferences/session separate from the normal app. The generated images contain no personal photos.

```sh
mkdir -p /private/tmp/picfetch-comparison-smoke
cp .scratch/maintainability/evidence/26-fixtures.go.txt /private/tmp/picfetch-comparison-smoke/fixtures.go
go run /private/tmp/picfetch-comparison-smoke/fixtures.go
python3 - <<'PY'
import json, pathlib, plistlib
root = pathlib.Path.cwd()
work = pathlib.Path('/private/tmp/picfetch-comparison-smoke')
app = pathlib.Path('/private/tmp/PicFetch Comparison Smoke.app')
identity = 'io.github.frathe.picfetch.comparison-smoke'
source = (root / 'main.go').read_text()
assert source.count('io.github.frathe.picfetch') == 1
(work / 'main.go').write_text(source.replace('io.github.frathe.picfetch', identity))
(work / 'overlay.json').write_text(json.dumps({'Replace': {str(root / 'main.go'): str(work / 'main.go')}}))
(app / 'Contents/MacOS').mkdir(parents=True, exist_ok=True)
(app / 'Contents/Info.plist').write_bytes(plistlib.dumps({
    'CFBundleIdentifier': identity,
    'CFBundleExecutable': 'PicFetch',
    'CFBundleName': 'PicFetch Comparison Smoke',
    'CFBundleDisplayName': 'PicFetch Comparison Smoke',
    'CFBundlePackageType': 'APPL',
    'CFBundleVersion': '1',
    'NSHighResolutionCapable': True,
}))
PY
go build -overlay /private/tmp/picfetch-comparison-smoke/overlay.json -o '/private/tmp/PicFetch Comparison Smoke.app/Contents/MacOS/PicFetch' .
'/private/tmp/PicFetch Comparison Smoke.app/Contents/MacOS/PicFetch' /private/tmp/picfetch-comparison-smoke/01-alpha.png /private/tmp/picfetch-comparison-smoke/02-beta.png /private/tmp/picfetch-comparison-smoke/03-rotated.png > /private/tmp/picfetch-comparison-smoke/app.log 2>&1
```

[Fixture generator](26-fixtures.go.txt): Alpha and Beta are 8192×6144, with 256-pixel checkerboard squares, four labeled corner blocks, and a central patch of alternating two-pixel white/black vertical stripes. Beta adds a magenta marker. The third source is Beta rotated clockwise into 6144×8192 pixels. Source labels rotate with the pixels.

## Actions and observed results

1. Press G, Space, Right, Space, physical Ctrl+D. Both sources appear fitted with the correct corner labels and side identities: [fit](26-native-fit.jpg). Actual production GLSL output rendered; no shader compilation error appeared in the log.
2. Scroll up over a photo while linked. Both views zoom together with the checkerboard and center landmarks aligned: [linked wheel zoom](26-native-linked-wheel-zoom.jpg).
3. Press physical Ctrl+L, move over the left pane, then press 1. The left view resolves the fine stripe detail while the right remains fitted: [unlinked source detail](26-native-unlinked-100-percent.jpg). Unlinked keys require a hovered pane; a key with no hovered pane intentionally has no target. Scroll over that pane changes its scale independently.
4. Press 0 over the left pane, Ctrl+L, and click Swipe. Both photos share the full viewport: [swipe](26-native-swipe-fit.jpg). End reveals Alpha across the surface and hides Beta's marker: [End](26-native-swipe-end.jpg). Home reveals Beta and its marker: [Home](26-native-swipe-home.jpg). Right moves away from the endpoint. Click Side by side to return; linked zoom still applies to both panes.
5. Click Swap. The magenta marker moves to the left and title/badges follow the source order, preserving scale: [swap](26-native-swap.jpg).
6. Press 1 then Escape. Comparison returns to Grid View, which still reports two selected items: [grid return](26-native-return-grid.jpg). This quick action sequence does not prove tile work was still pending at the instant of close.
7. Press Return on Beta, then R. Its temporary single-view rotation is visible: [viewer rotation](26-native-viewer-rotation.jpg). Press G, select Alpha and Beta again, and Ctrl+D. Comparison shows canonical unrotated source pixels: [canonical comparison](26-native-canonical-after-viewer-rotation.jpg). Comparison has no rotation command of its own. Comparing Beta with the physically rotated third source preserves their landscape/portrait geometry and rotated landmarks: [rotated source](26-native-rotated-source.jpg).
8. With comparison active, Ctrl+L, point into a pane and press 1, then Cmd+Q. The first native run aborted with an AppKit exception; see below. After the fix, the same isolated bundle rendered [source detail before quit](26-native-before-fixed-quit.jpg) and exited 0 without that exception.

Mouse-drag attempts did not visibly move the photograph. Screenshots ending in `-attempt.jpg` and `26-native-drag-attempt.jpg` are inconclusive attempts, not passes. A temporary overlay diagnostic confirmed the pane's Dragged callback was not reached. The [native event trace](26-native-drag-events.txt) shows down and up at the starting position, followed by movement at the ending position with Button:0. No movement arrived while the button was held, so this automation run cannot validate mouse pan or divider drag. The diagnostic changes are outside the working source; no app behavior was changed to accommodate the input trace.

## Shutdown defect and regression evidence

The [original native crash](26-native-shutdown-crash.txt) reports NSInternalInconsistencyException for mutation of a live main menu off the main thread. Its stack is OnStopped -> compare.Close -> comparisonClosed -> syncMenus -> refreshMainMenu -> mergeAppWindowMenus. Fyne executes lifecycle shutdown after its main queue has drained; another fyne.Do cannot restore that queue.

Shutdown now retires the viewer's title/menu updates before closing features. Native menu folds already queued before shutdown also check retirement. Worker cancellation, session persistence and Fyne's final preferences flush remain in the same shutdown path.

The maintained root UI guard `TestShutdownClosesComparisonWithoutRefreshingRetiredUI` first failed with changed closing title, rebuilt menu admission and six native-menu reads. It now checks feature closure plus preserved final title/menu state and no native-menu reads, including explicit late refresh/fold entry. Five negative overlays separately remove shutdown retirement, title retirement, menu-state retirement, menu-refresh retirement and queued native-fold retirement; every mutation fails the named guard. Logs: `/private/tmp/picfetch-maintainability-26-{shutdown-red,negative}.log`.

The affected native race selection passed (root UI 145.623s); the full comparison package passed uncached (3.397s). The original app exited 2; the fixed comparison/detail quit exited 0, with only the existing Fyne threading-migration notice in its log. The full `make verify` gate passed: native format/TUF/Qodana/vet/build and every Linux race partition, exact 665-test UI manifest (212/225/228). UI shard times were 308.207s/307.879s/246.080s; process exited 0. Log: `/private/tmp/picfetch-maintainability-26-verify.log`.

`GOFLAGS=-overlay=/private/tmp/picfetch-comparison-smoke/overlay.json make run` was also executed. Its unbundled executable could not be addressed by the desktop tool and was stopped with SIGTERM; that invocation supplies no additional interaction or clean-quit claim. The actual interaction evidence above comes from the explicitly identified temporary app bundle. No reference golden was regenerated and no artifact was published.

## Native close with demonstrably pending work

After the ordinary renderer checks, an isolated Go overlay used the existing per-renderer `generateTile` seam to hold each tile call until its supplied context was cancelled. This changes only the temporary smoke binary, preserving the production Fyne renderer, worker owner, queues, Close and shutdown paths. It verifies cancellation, not detail fidelity. Reproduce after the setup above:

```sh
python3 .scratch/maintainability/evidence/26-held-overlay.py
go build -overlay /private/tmp/picfetch-comparison-smoke/held-overlay.json -o '/private/tmp/PicFetch Comparison Smoke.app/Contents/MacOS/PicFetch' .
'/private/tmp/PicFetch Comparison Smoke.app/Contents/MacOS/PicFetch' /private/tmp/picfetch-comparison-smoke/01-alpha.png /private/tmp/picfetch-comparison-smoke/02-beta.png > /private/tmp/picfetch-comparison-smoke/held-work.log 2>&1
```

Open comparison as above. Inspect the log for two `tile started` lines and no matching cancellation before closing. Press Escape: both calls report cancellation and Grid View retains two selections ([screenshot](26-native-held-work-close.jpg)). Reopen comparison, observe two more started calls, then Cmd+Q: both calls report cancellation and the native process exits 0. [Retained ordered trace](26-native-held-work.txt). No timeout or guessed delay determines readiness. Rebuild with the ordinary `overlay.json` after this probe to restore the normal temporary bundle. No diagnostic instrumentation was added to working production source.

## Windows continuation with user-operated input

The user runs the refreshed production packages through the
[isolated Windows launcher](27-windows-package-smoke.ps1.txt), with the same
8192x6144 Alpha/Beta fixtures. Automated VM input remains disabled because it
drops keystrokes; observations use the Computer Use screenshot tool only.
The returned [process records](27-windows-package-results/processes.json) and
[ordered transcript](27-windows-package-results/transcript.txt) attribute all
comparison observations below to the ordinary Windows ARM64 package, PID 6672,
window DPI 96, exit 0. Its stdout/stderr are empty. Both x64 package attempts
failed before creating a visible window; the initial visual-order attribution
was incorrect. The returned [environment report](27-windows-graphics.json), collected with the
[read-only probe](27-windows-graphics-details.ps1.txt), records Windows 11 Pro
10.0.26100 ARM64, Red Hat VirtIO GPU DOD controller / QEMU VIRTIO GPU, driver
22.7.38.43, one 1728x1043 monitor at 100% scale (API result 0). Together with
window DPI 96 this is representative standard-density output in a virtual
desktop, not a second physical monitor. All four existing Desktop graphics
libraries are ARM64; their names and hashes are retained. No renderer override
was set. Hardware acceleration is not inferred from the VM adapter name.
Production comparison shader output is visible in the retained frames, and
the clean process log contains no shader compilation errors.

After viewing Beta, the user pressed G, Ctrl+A, Ctrl+D. The
[comparison baseline](26-windows-fit.jpg) shows both complete sources fitted
side by side, Alpha left and Beta right, the magenta marker only on Beta,
correct title/badges, and the linked-state Unlink control. No spinner remains.
The user then dragged the middle of the left image right and down and confirmed
that both images moved together. The
[retained pan position](26-windows-linked-pan-occluded.redacted.md) shows Beta shifted
right/down and both images' visible upper edges aligned; Windows Start obscures
most of the left pane, so the full linked-motion observation is the user's
confirmation.

The user pressed Ctrl+L and dragged Alpha independently. The
[unlinked result](26-windows-unlinked-pan.jpg) shows the Link control and
"Entkoppelt: Links" status, Alpha moved back left/up, and Beta's corner and
magenta landmarks unchanged from the prior frame. Unlinked pan passes.

The user pressed Ctrl+L to relink, selected Wischen (Swipe), and dragged the
divider left. The [swipe result](26-windows-swipe-drag.jpg) shows the reveal edge
near 30% of the viewport, moved from its initial midpoint; Alpha is clipped
at that edge and Beta, including its magenta marker, is visible to the right.
The toolbar now offers Nebeneinander (Side by side) and Entkoppeln (Unlink).
The previously distinct photo positions remain visible. Native divider dragging
and the transition into swipe pass. A taskbar preview obscures a small part
of the bottom-left viewport but does not cover the inspected reveal edge.

The user selected Nebeneinander (Side by side) and scrolled up twice over
Alpha. The [linked zoom result](26-windows-linked-zoom.jpg) shows both
checkerboards enlarged by the same factor, with their separate positions
retained and the Wischen (Swipe) action available again. Return to side-by-side
and linked wheel zoom pass.

The user pressed Ctrl+L, hovered Alpha and pressed 1. The
[decoded-pixel detail result](26-windows-unlinked-detail.jpg) resolves the
fixture's fine black/white vertical stripes, which were gray at fit scale.
Beta's position, checkerboard size and magenta marker remain unchanged from
the linked-zoom frame. Unlinked zoom and large-source detail pass. Capture
resizing means the screenshot's pixel dimensions alone are not a display-DPI
measurement. Independent process/monitor APIs report DPI 96 and 100% scale.
The process subsequently quit cleanly (exit 0).
