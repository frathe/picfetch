# Spiral soft image entrances

Route: Standard follow-up within the accepted Spiral qualification work.
Owner: Pico, lead inline. Baseline: e6416ff; initially clean working tree.
Ronin requested fully transparent entrances and launches closer to the centre
after viewing the native trial. Implementation is authorized; commits are not.

## Contract

- Multiply the existing, live-configured opacity by a smooth 0-to-1 fade over
  the first 0.75 seconds of each admission. Motion and GIF playback begin at
  admission; the fade does not pause or restart either clock.
- Reduce the protected centre disc from 8% to 3% of the shorter physical
  canvas dimension. Keep the existing size range, aspect ratio, whole-card
  clearance, curve, acceleration and complete-exit rules.
- Preserve the three active texture slots and one prepared preview. Ronin
  asked whether preloading three could help; it can absorb decode delays,
  but no decode bottleneck has been established. Rendering FPS is a separate
  measurement. No preload expansion belongs to this increment.
- The native qualification uses the live-window tool and FPS overlay as
  Ronin requested. Retain exact observations and distinguish the overlay's
  UI-update timing from GPU presentation and single-frame motion evidence.

## Work

### T1 — Soften entries and narrow centre clearance

Owner: T0 inline. Files: flight.go, shader.go, existing tunnel_test.go in
internal/ui/spiral. Depends: none. No new interface or dependency.
Seams: the already accepted flight geometry/retirement boundary and native
Spiral shader. A software painter cannot verify GLSL pixels.
Test: fully transparent admission; gradual monotonic fade, normal opacity
after 0.75 seconds; launch bounds closer to the core, safe through growth at
all existing sizes/aspects, complete exit, unchanged clocks across rebasing.
Verify: `go test ./internal/ui/spiral -run '^TestTunnel(Flight|ClockAndResize|ResizeRecovery)$' -count=1`.
Budget: 0 spawns; one lead review plus fixes; no full suite during iteration.

### T2 — Native review and handoff

Owner: T0 inline. Depends: T1. Files: this plan, todos.md, qualification
record and the local specification/tickets as needed.
Verify: `go test -race ./internal/ui/spiral -count=1`, changed-file GoLand
inspections, native live-window/FPS trial, `make verify`, `make build`, and
`git diff --check`. Preserve actual environment failures in the evidence.
Budget: 0 spawns; one final gate. No new root UI runnable/test file, locale
string, package or dependency is planned, so those metadata inventories stay
unchanged.

Task graph: T1 -> T2. All implementation, review and fixes remain with Pico.

## Evidence

RED: the revised flight test reported admission opacity 0.15 instead of zero.
After adding the fade, the closer-entry guard independently failed at a
94.026-pixel launch radius in an 800x600 viewport. Reducing the launch offset
made both guards pass. The complete geometry matrix retains the smaller
protected disc and complete exits across all existing sizes/aspects/angles.
The clock/resize regression now crosses a minute boundary at age 0.5 seconds,
inside the entrance fade, while preserving the same GPU/scheduler age.

Focused flight/clock/resize tests pass (0.589s); the complete Spiral race
package passes (2.749s). GoLand returned explicit empty error/warning results
for each of flight.go, shader.go and tunnel_test.go. The batch endpoint returned
no file entries, so individual inspections supplied the actual coverage.

The rebuilt isolated native trial renders the new shader at 3840x1600.
Live-window samples show closer entrances and preserved centre clearance;
its first six FPS readings are 63/63/63/63/60/63. Capture cadence cannot
resolve the complete subsecond alpha curve, and the UI-loop readout is not
a GPU presentation counter. `make build` passes and refreshes bin/picfetch.
The [durable qualification record](../docs/spiral-qualification-2026-09-12.md)
retains both the positive observations and later baseline FPS variability.

The final `make verify` finishes with exit 2. Formatting, metadata checks,
vet/build and all three UI race partitions pass (390.438s / 397.205s /
405.428s). Its only failing cases are the existing local amd64
`TestLinuxWorkerIsolation` and
`TestAssetInstall/worker_reaches_asset_check_and_exits` seccomp failures;
the latter's parent also fails. Both report `offline worker seccomp: invalid
argument`. Raw race events are in `.scratch/race-runs/20260912T140914Z-sNvm4r/`;
extracted results and changed-file inspections are retained under
`.scratch/hypno-spiral-tunnel/soft-entry-20260912/`. This is not a clean
complete local-gate claim.

The rebuilt trial exits normally after 571.844 seconds, with no more than
three sampled texture slots and warm live heap of 21.45–37.95 MiB (median
32.57 MiB). A final direct overlay reading is 63 FPS. The binary SHA-256 is
`4627e29adfb21750573aec14ba08fd0bc51ef4a2d2180e9b88fb2b5436b500c3`.
The lead's final review found the CPU/shader fade and launch bounds aligned,
with source alpha, opacity controls and lifetime behavior retained.
`git diff --check` passes. No commits were created. The follow-up is
implemented; the broader motion/backend limitations remain in ticket 05.

The pre-change native trial retained live-window and FPS observations under
`.scratch/hypno-spiral-tunnel/motion-qualification-20260912/`.
Initial Ripple FPS samples were 63/62/63/60/62/63. Later samples in both
presets varied widely, including with five-second pauses between snapshots;
no preset-specific or decode-related cause has been established. Do not
describe the initial six samples as sustained 60 FPS qualification.

## Ledger

| Task | Spawns budget/actual | Review rounds | Full suite |
|---|---|---|---|
| T1 | 0 / 0 | 1 | no |
| T2 | 0 / 0 | 1 | 1, completed with the known seccomp failures |
