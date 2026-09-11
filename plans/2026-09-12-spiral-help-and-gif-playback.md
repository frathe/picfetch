# Spiral help, GIF playback and image controls

Second scope addition: Ronin requested an image-size slider. Default 1x;
0.5x–2x in 0.05 steps. Scale the original start/end footprint together for
new admissions; in-flight geometry remains immutable. Route admission,
retirement and GLSL must use the same captured scale, preserving the clear
centre at all sizes. The former fixed footprint is now the 1x default. Retain
the value across same-process reopen; extend the scrollable panel by one row.
RED/GREEN oracle: `TestTunnelImageSize` plus size extremes in
`TestTunnelFlight`; complete Spiral race tests and native visual checks.

Scope addition during verification: Ronin requested an image-transparency slider
that shifts the opacity range. Preserve the default 15–85% visibility, apply
  changes to in-flight cards and the combined layer cap immediately, retain the
setting across egg reopen, and keep the centre/feather/source-alpha rules.
Ronin accepted retaining the original 85% visibility ceiling. Keep at least 1%
base visibility at the transparent
extreme. The slider shows the resulting transparency range, rather than an
unexplained numerical offset. Add it in the free fourth row of the image column.

Slider acceptance: `go test -race ./internal/ui/spiral` must cover real control
membership, live uniform changes, range boundaries, overlap ceiling and reopen;
locale parity checks and native slider exercise remain required. An extra
read-only opacity-test scout found no automated GLSL framebuffer assertions;
uniform/control tests must not be described as rendered-pixel evidence.

Route: Deep follow-up to the accepted tunnel plan. Owner: Pico, lead.
User requested both implementation changes on 2026-09-12. No commits authorized.
Baseline: c725cae; initial working tree clean.

## Behavior

- H in Spiral toggles its own help overlay, per Ronin’s follow-up. F1 retains
  the main manual binding. Update the hint in both locales.
- GIFs play their composited frames and source delays, independently per admitted
  image, starting at frame zero. They loop while that image is in the tunnel.
- Preserve the source snapshot, image order, three-slot limit, motion, opacity,
  static formats and serial decode/cancellation lifecycle.
- Bound animated preview retention and native decode work; oversized animations
  retain a static first-frame fallback consistent with existing imaging policy.

## Work and verification

1. Reproduce native F1 routing, then implement Ronin’s chosen H binding with
   RED/GREEN overlay-toggle tests. Keep native F1/menu behavior unchanged.
2. Add bounded animated preview decoding through canonical imaging/compositing.
   RED/GREEN: composited GIF pixels, unequal/zero delays, bounds, budget fallback,
   cancellation; existing static preview tests stay green.
3. Play frames in Spiral's existing acknowledged UI clock, without a worker per
   card. RED/GREEN: frame-zero admission, timing/looping, independent flights,
   long frame stalls, teardown and stale results.
4. Native GIF/H trial, changed-file GoLand inspections, focused races,
   localization/metadata checks, make verify and make build. Record actual
   environment failures; keep the prior native-capture gaps explicit.

Seams remain the accepted canonical imaging, native/window input, and real
Spiral session/renderer boundaries. One read-only GIF scout while the lead
reproduces F1; no implementation or review delegates. The full gate was repeated after the scope expanded.
No new dependencies or shipped assets are planned; existing third-party notices
and package notice delivery remain applicable.

## Evidence

Native baseline reproduction: starting from the real manual secret, Spiral was
focused; F1 brought PicFetch Manual to the front. The old canvas-handler-only
unit tests bypass the native menu accelerator and cannot catch this regression.

The native-menu regression test also went RED: AppKit retained the F1 key
equivalent on the application manual item. Before changing native behavior,
Ronin chose H instead. The temporary native test/accessor were removed; the
existing platform menu implementation remains unchanged.

H toggle RED: the old dispatcher left the overlay visible after H. GREEN:
`go test ./internal/ui/spiral -run '^TestOverlayToggleKeys$' -count=1` passed.
Native rebuilt trial: H hid the real help overlay, H restored it with the new
hint, and F1 still brought the manual forward. Ronin confirmed "it works".

Animated preview RED: the new API was absent, then the retained-budget test
showed the initial implementation falling back instead of reducing resolution.
GREEN: adaptive previews preserve both frames and unequal/zero source delays
within the byte budget. The frame-count guard was negatively verified by
temporarily admitting 4,097 frames: the fallback test failed; the 4,096 limit
was restored. DisposalPrevious, logical-canvas transparency, cancellation and
native-decode fallback cases are also covered.

Real shader playback RED: at 100ms the old tunnel still showed the red first
frame instead of blue. GREEN: frame boundaries, loop restart, delayed clocks,
live order changes, independent repeated admissions and close texture cleanup.
`go test -race ./internal/imaging ./internal/ui/spiral` passed (24.263s / 2.994s).

All nine changed Go files were inspected with GoLand, including weak warnings.
The compositor's previous nil-analysis suppression was replaced by a non-nil
per-iteration canvas reference; reinspection is clean without suppressions.
The last preview test/message adjustment was also reinspected cleanly.

Resource contract: up to three live previews plus one ready/decoding preview;
16MiB of retained RGBA pixels per animated preview, at most 512px longest edge.
Adaptive sizing retains animation within that budget. A serial native decode
gate allows at most 4,096 frames and 256MiB estimated paletted/canvas pixels
before scaling. These are pixel-accounting limits, not a claim about exact RSS
or GPU memory. No additional animation workers are introduced. Close drops
active and ready frame references even if a cancelled read is still settling.

The repository's actual `assets/picfetch_functionality.gif` decodes to 125
animated preview frames at 241x139, retaining 16,749,500 bytes, without fallback.
A separate two-frame native fixture uses 262,144 bytes at 256x128.

## Final file map and ownership

| Task | Files / seam | Acceptance command | Owner |
|---|---|---|---|
| H help | `spiral.go`, `overlays.go`, existing `spiral_test.go`, en/de catalogues | `go test ./internal/ui/spiral -run '^TestOverlayToggleKeys$'` plus native H/H/F1 sequence | Lead |
| Preview decoding | `imaging/preview.go`, `gif.go`, new `preview_test.go` | `go test ./internal/imaging -run '^(TestAnimatedPreview.*|TestDecodeAnimatedGIF.*|TestTunnelPreview.*)$'` | Lead |
| Playback / lifecycle | `spiral/playback.go`, `tunnel.go`, `tunnel_test.go`, `spiral.go` | `go test -race ./internal/ui/spiral` | Lead |
| Image controls | `settings.go`, `settings_test.go`, `state.go`, `shader.go`, `flight.go`, tunnel admission/tests and both catalogues | `go test -race ./internal/ui/spiral`; `go test . ./internal/ui -run '^TestTranslations'`; native trial | Lead |
| Handoff | architecture, tracker, Qodana exclusion, plan | `make verify`; `make build`; changed-file GoLand inspections | Lead |

Task graph: preview decoding -> playback -> final gate; H help is independent.
The public seam is `imaging.LoadAnimatedPreviewContext(ctx, uri, maxEdge,
animationBytes) (*LoadedImage, error)`; GIF pixels share the existing canonical
compositor, static formats share the existing thumbnail path. No new exported
Spiral API. No root UI tests were added, so shard assignments remain unchanged.

Scout delegation: one read-only survey of the existing GIF decode/timing/cache
paths while the lead reproduced native F1. G1-G5 passed: bounded factual query,
file:line evidence verifiable locally, no writes, broader cold search than lead
context, and previously unexplored imaging behavior. No review delegation.

Cost ledger: one scout spawn, reused for a second read-only opacity-test survey
after the scope addition; all implementation/review/fixes inline,
two completed full verification runs (the second was needed because Ronin added
controls after the first began). Standards/spec review found no remaining
implementation defect after the compositor cleanup. Prior tunnel video/resource
plateau qualification remains open in the original plan.

Transparency RED: `TestTunnelTransparency` found only six controls; the slider
was absent. GREEN: seven controls, actual membership in the live overlay tree,
default and shifted uniform ranges, 1% floor / 85% ceiling, activity timestamp,
unchanged active flights, range label, and same-process reopen. Focused control
and shader tests pass. The complete Spiral race package passes in 2.871s, and
root/UI `TestTranslations*` checks pass. All four newly changed control/state/
shader files were inspected in GoLand, with zero findings.

The first complete `make verify` finished: all three UI shards passed; only
the previously recorded `TestLinuxWorkerIsolation` and
`TestAssetInstall/worker_reaches_asset_check_and_exits` failed in local amd64
Docker with `offline worker seccomp: invalid argument`. Artifacts:
`.scratch/race-runs/20260911T232200Z-9PzNHt`. This run predates the slider; the
second run covers the final controls. The final `make build` and
`make verify-build` both passed after the size addition.

Image-size RED: the control test found only seven sliders. GREEN: 0.5x–2x range,
1x default, per-admission size capture, matching CPU footprint/GPU uniforms,
unchanged earlier flights and retained size on reopen. `TestTunnelFlight`
checks core clearance and complete exit at 0.5x, 1x and 2x across aspect ratios,
angles and portrait/landscape frames. The final native Spiral race package
passes (2.877s). Latest locale tests, `make verify-build` and `make build` pass.
All seven code/test files touched by sizing were reinspected in GoLand with
zero findings, bringing coverage to all changed Go files.

The second Docker run's JSON explicitly contains passing
`TestTunnelTransparency` and `TestTunnelImageSize`, as well as GIF playback.
It includes the final controls, despite having started before the size request:
the runner mounts the live workspace and compiles after container setup.

Native evidence: the production viewer trial verified H/H/F1, and screenshots
show both new controls with readable range/size labels. Desktop automation
repeatedly lost the full viewer's focus to the manual; a second trial entry
opened the actual `spiral.New` / `Show` feature as its only window. This uses
the production shader and controls, with the generated two-frame GIF and the
repository GIF. Screenshots show both the upper-white and lower-white GIF
frames in the moving scene. It is a feature smoke test, not another full-viewer
integration claim; the original viewer entry is retained alongside it.
Ronin then confirmed: "all sliders work as intended." Automation stopped so
the user could keep using the trial. Exact rendered alpha/GPU resource plateau
and the earlier moving-capture deliverable remain unmeasured.

Retained native screenshots:
`.scratch/hypno-spiral-tunnel/native/gif-controls-first.jpg` and
`.scratch/hypno-spiral-tunnel/native/gif-controls-second.jpg`.

Final Docker result: all three UI partitions passed (393.150s / 406.894s /
423.870s). Every package except the two existing local isolation failures
passed. The only failing tests were `TestLinuxWorkerIsolation` and
`TestAssetInstall/worker_reaches_asset_check_and_exits`; both report
`offline worker seccomp: invalid argument`. Artifacts:
`.scratch/race-runs/20260911T233251Z-wEJxbT`. The complete `make verify` therefore
exits 2; this is not a claim of a clean full gate. No commits were made.

`GOOS=windows GOARCH=amd64 go vet ./internal/...` also completed successfully.
Windows runtime rendering was not exercised; native visual evidence is macOS.
