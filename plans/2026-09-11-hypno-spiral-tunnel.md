# Hypno Spiral tunnel image stream

**Route:** Deep. **Status:** accepted five-ticket plan, refined 2026-09-12;
implementation started on Ronin's `/implement` request. **Owner:** Pico / lead.
Implementation with SDD/TDD is authorized; commits remain unrequested.

The [specification](../.scratch/hypno-spiral-tunnel/spec.md) owns behavior and
tuning defaults. The [ticket index](../.scratch/hypno-spiral-tunnel/issues/README.md)
owns the execution frontier and per-slice acceptance evidence. This plan maps
that same sequence to package boundaries, files, and verification.

## Result and visual direction

The existing full-screen Spiral becomes an infinite stream of static previews
from the images loaded when the egg opens. Preserve the clear spiral centre,
launch-time duplicate filtering, main/random order, three-card ceiling,
15-to-85-percent base opacity, and the accepted controls.

Pico's touch is a calm depth progression: small photographs linger near the
centre, grow and accelerate gently, bend with the visible spiral, and keep
moving through full exit. Upright detail and soft inward feathering preserve
recognizable pictures. Limited drift between batches and a light preference
for separated launch bearings prevent mechanical repetition without jitter.
Nearer cards cover farther ones; their combined image layer still leaves at
least 15 percent of the spiral visible.

The spec distinguishes Ronin's requirements from provisional numerical tuning.
Native motion, not a screenshot or a shader-source check, decides whether
these choices produce the intended result. Do not introduce another approval
gate for that tuning.

## Settled contracts

- Snapshot current non-nil source URIs in main-view order at closed-to-open.
  Read the canonical duplicate visibility once. Known stacks contribute their
  highest-native-resolution representative; unknown groups remain eligible.
  Do not start or await duplicate analysis, narrow to grid selection/search,
  or observe subsequent list/model changes. Source bytes themselves are not frozen.
- Re-trigger raises the existing session. Gesture preset selection remains
  intact without resetting its snapshot, flights, clock, or pending work.
- Main order wraps. Random uses complete shuffled cycles and avoids an
  immediate boundary repeat when another candidate exists. Source ordering
  and spatial randomness are independent.
- An admission requires a ready preview, a free slot, and elapsed time since
  the last actual admission. An overdue entrance uses a newly free slot
  immediately; no extra gap and no catch-up burst. Zero visible cards is valid
  during startup, failure, or sparse timing.
- Existing flights retain their captured parameters. Profiles last 3-7
  successful admissions; failures/waits do not count. Randomness changes take
  effect at a profile boundary, including exact neutral values at zero.
- A source can repeat concurrently after wrap when fewer than three usable
  identities exist. Otherwise wait for the selected identity to leave without
  changing order. Determine usability from observed cycles, not eager decoding.
- Keep fixed three-slot textures, at most one prepared preview, and one serial
  decode lane across sessions. Bound retained previews separately from the
  canonical decoder's transient full-source memory.
- Both frame and preview deliveries require a current session on the instance
  UI queue. Cancellation releases workers without waiting for UI. Close on UI,
  join and drain off UI; basic safety belongs to the first async slice.

Full ranges, units, failure backoff, opacity/feather semantics, and route bounds
are in the spec. Keep one source of truth rather than duplicating its table here.

## Architecture and seams

ARCHITECTURE.md was read before design. Existing packages remain; no new
dependency, native runtime, asset, or standalone rendering framework is planned.

| Owner | Contract |
|---|---|
| internal/ui | Own Spiral, compose both trigger doors, freeze source/duplicate values, and integrate shutdown/harness cleanup. Proposed private seams: viewer.tunnelSources(), viewer.openSpiral(), viewer.openSpiralForGesture(bool). |
| internal/ui/help | Emit a callback from the manual secret. Proposed SetOnSpiral(func()); remove direct renderer ownership without introducing viewer/duplicate dependencies. |
| internal/ui/spiral | Receive a copied value through Show([]fyne.URI) and ShowForGesture(bool, []fyne.URI). Own the session, scheduler, shader, controls, cancellation, and Close/Settle. No viewer or shared controller. |
| internal/imaging | Parameterize the canonical thumbnail path through LoadThumbnailAtEdgeContext(context.Context, fyne.URI, int) (image.Image, error). Tunnel requests 512 pixels; existing grid entry points remain 200 pixels. |

All call sites move atomically in ticket 01, so the first vertical slice builds.
Empty sources still open the shader-only egg. Do not reuse Mosaic's source
helper, because its selection rules serve a different feature.

### Renderer and geometry

Reuse canvas.Shader and canvas.NewShaderAnimation. Permanently seed three
texture keys, provisionally traveller0/1/2, with transparent placeholders;
replace a texture only at admission/release. GLSL calculates position, size,
curve, feather, and opacity. Keep desktop/GLES declarations aligned and explicit
sampler calls compatible with the existing backend. Do not use dynamic sampler
arrays or require an arbitrary total uniform count.

Slot metadata carries immutable birth/duration, aspect, bearing, signed curve,
and safe-exit parameters. Additional metadata may support correct depth order.
Composite by normalized flight depth, far to near, with stable admission-id
ties independent of which reusable slot holds the image. Source alpha and
feather multiply base opacity; cap aggregate image-layer coverage at 85 percent
before compositing onto the opaque procedural background.

Keep whole growing card bounds outside the protected disc around the actual
live core. Use the shorter physical frame dimension for normalized geometry.
Compute full exit from the current frame and card extent; randomness changes
only additional margin beyond safe clearance. Base opacity reaches 85 percent
at the viewport edge, independent of the remaining off-screen travel.

Source reconnaissance establishes that Ripple and the dominant Nautilus layers
follow rotational speed's sign in shader coordinates. The gesture flag selects
a preset; it is not the winding direction. Preserve the last nonzero sign
while paused. Follow currently updates the actual centre directly; preserve
that behavior and calculate photo geometry around that same live centre.

### Time, work, and cleanup

Use a coherent monotonic session clock for births, shader age, pacing, and
retirement. Do not accumulate presumed 16 ms steps. Validate shader time
precision for long sessions; if rebasing is necessary, preserve existing
background phase and active flight ages.

The existing frame ticker's pre-dispatch generation check is insufficient for
a queued callback after reopen. Recheck identity inside queued frame callbacks
as well as preview callbacks. Use buffered acknowledgement before dispatching
another frame, with cancellation independent of callback execution.

Install a per-instance UIQueue seam on Spiral using the established uitest
pattern. A serial preview lane remains shared across reopen so a slow old
decoder cannot create overlapping unbounded workers. Stale results are discarded
on UI; release retained textures/previews/source slices on close. Closing must
not wait on file I/O. Settle observes all admitted work and drains deliveries;
the harness closes a running infinite session before asking it to settle.

## Approved task graph

~~~text
01 frozen source + first real flight
                 |
02 continuous main-order stream + speed/gap controls
                 |
         +-------+-------+
         |               |
03 random flow       04 resize / Follow /
   + control            lifetime hardening
         |               |
         +-------+-------+
                 |
05 native qualification + final documentation
~~~

The 03/04 branches express dependency independence, not permission for
concurrent edits to the same Spiral files. Their shared motion/session seams
exist after 02. Use injected neutral/varied profiles for 04's contract tests
without implementing 03's workflow twice.

### 01 - Open a frozen, duplicate-aware tunnel

**Owner:** lead inline. **Depends:** none.
**Files:** create internal/ui/tunnel.go, internal/ui/tunnel_test.go,
internal/ui/spiral/tunnel.go, internal/ui/spiral/tunnel_test.go, and
internal/ui/spiral/uiqueue.go; modify viewer.go, features.go, gesture.go, run.go,
harness_test.go in internal/ui; help.go, manual.go, easteregg_test.go in
internal/ui/help; spiral.go, shader.go, shader_test.go and existing lifecycle
tests in internal/ui/spiral; thumbnail.go and thumbnail_test.go in
internal/imaging. Update ARCHITECTURE.md, qodana.yaml and UI shard metadata.
**Contract:** Both real triggers produce a copied duplicate-aware snapshot and
one safely decoded, rendered, curved, feathered, fully retired traveller.
The permanent three-slot renderer initially uses one active slot. Empty and
all-failed initial passes remain responsive. Basic async cancellation, queue,
close/reopen, and shutdown integration are included.
**Test:** Write intended RED cases for 01-A through 01-F before implementation.
Tests use normal viewer construction, held readers and the drainable queue.
**Verify:** Run the exact focused commands and early native moving demonstration
in [ticket 01](../.scratch/hypno-spiral-tunnel/issues/01-open-frozen-duplicate-aware-tunnel.md).
**Budget:** 0 implementation spawns; one lead review, further fixes/rechecks as
findings require; no full-suite run.

### 02 - Stream the frozen list in main order

**Owner:** lead inline. **Depends:** 01.
**Files:** internal/ui/spiral/tunnel.go, tunnel_test.go, shader.go,
shader_test.go, state.go, state_test.go, settings.go, settings_test.go;
translations/*.json; relevant source-snapshot integration subtests.
**Contract:** Infinite main-order admissions, exact actual-entry pacing,
three-slot ceiling, bounded serial lookahead, recoverable failed-source cycles,
depth-correct translucent overlaps, and Image speed/Image gap controls.
**Test:** Cover 02-A through 02-F with controlled admission timestamps, full-slot
waits, long ticks, small/partially unreadable sets, overlap and future-only
control changes. A sparse scene may legitimately contain zero images.
**Verify:** Use [ticket 02](../.scratch/hypno-spiral-tunnel/issues/02-stream-main-order-tunnel.md)
commands and moving demonstration; update localization and metadata in this slice.
**Budget:** 0 implementation spawns; one lead review plus needed fixes; no full suite.

### 03 - Add random Starfield flow

**Owner:** lead inline. **Depends:** 02.
**Files:** internal/ui/spiral/tunnel.go, tunnel_test.go, state.go,
state_test.go, settings.go, settings_test.go; translations/*.json.
**Contract:** Main/Random controls, complete shuffled cycles, boundary-repeat
avoidance, pending-order invalidation, 3-7-successful-entry batch profiles,
bounded gradual drift, and a bounded spatial-spacing preference.
**Test:** Cover 03-A through 03-F using seeded instance randomness; verify order,
profile stability/counts/bounds, zero reset, active-flight preservation, and
route choice without rejection loops or admission delays.
**Verify:** Use [ticket 03](../.scratch/hypno-spiral-tunnel/issues/03-add-random-starfield-flow.md)
commands and native motion at default, zero, and maximum randomness.
**Budget:** 0 implementation spawns; one lead review plus needed fixes; no full suite.

### 04 - Keep the tunnel safe through resize and teardown

**Owner:** lead inline. **Depends:** 02.
**Files:** internal/ui/spiral/tunnel.go, tunnel_test.go, spiral.go, spiral_test.go,
shader.go, shader_test.go, mouse.go, mouse_test.go, monitor.go, monitor_test.go;
internal/ui/tunnel_test.go, harness_test.go and run.go as needed.
**Contract:** Retained flight age through resize/Follow, live-centre clearance,
complete exit, bounded near-edge route selection, monotonic time, stale queued
frame/preview rejection, serial decode across reopen, responsive shutdown.
Do not defer the safety baseline from 01 to this ticket.
**Test:** Cover 04-A through 04-F with varied frame shapes, edge-centre locations,
clock jumps, held decoder completions and old-generation queued callbacks.
**Verify:** Use [ticket 04](../.scratch/hypno-spiral-tunnel/issues/04-keep-tunnel-safe-through-resize-and-teardown.md)
commands plus native Follow/resize/close scenarios.
**Budget:** 0 implementation spawns; one lead review plus needed fixes; no full suite.

### 05 - Qualify and document the finished tunnel

**Owner:** lead inline. **Depends:** 03 and 04.
**Files:** final tuning in the preceding implementation files; spec, tickets,
this plan, ARCHITECTURE.md, todos.md and any required translation/test metadata.
**Contract:** Complete all 05-A through 05-F with actual native moving evidence,
sustained default operation, visual tuning, appropriate regression coverage,
GoLand inspections, and the final repository gate.
**Test:** Reconcile the accepted behavior against the real integrated paths.
Do not count no-test matches, skips, static screenshots, or software shader
checks as proof of runtime motion.
**Verify:** [Ticket 05](../.scratch/hypno-spiral-tunnel/issues/05-qualify-and-document-finished-tunnel.md)
specifies focused checks, a 30-60-second native capture, at least ten minutes at
defaults across multiple cycles/batches, edge-case trials, make verify, and the
trial build. Inspect every changed Go source and report unavailable coverage.
**Budget:** 0 implementation spawns; one final lead review plus required fixes;
one final full gate, repeat only when changes/failures justify it.

## Verification and evidence rules

### Ticket 05 resumed, 2026-09-12

Ronin requested `/implement next todo`; the first open TODO is this final
qualification. Baseline: `324cec5`, clean working tree. Pico owns the native
trial, assessment, any fixes and verification inline; zero delegates. Reuse
05-A through 05-F and the existing task budget. Production feature code is
already implemented, including the later animated-GIF and control changes.

Work: retain an isolated native harness and public repository fixtures under
`.scratch/hypno-spiral-tunnel/qualification-20260912/`; sample actual shader
uniforms/textures, Go heap and process RSS through at least ten minutes at
defaults; collect a timestamped 30-60-second native screenshot sequence as a
moving record; exercise native controls and teardown; run the focused
acceptance commands, GoLand inspections and one canonical `make verify`.
Document sampling/capture limitations and exact failures. No new shipped
dependency, asset, package or benchmark API is planned. The scene uses the
existing CC0/public-domain Explorer fixtures, PicFetch's icon and demo GIF,
plus the existing synthetic two-frame GIF.

Observations, raw commands and artifact locations will be reconciled below
before marking acceptance criteria complete. Native evidence is macOS OpenGL
only; the known local amd64 seccomp error must be verified, not assumed away.

Native resize found a concrete defect: choosing Window > Exit Full Screen
restores a 1x1 canvas. The shader has a one-pixel minimum and startup never
requests a windowed size. Extend the existing `TestShowOpensFullScreenShaderWindow`
at the accepted Spiral/window boundary to require a usable restored canvas
(at least 640x480); verify its RED result, initialize a 960x600 window before
fullscreen, and repeat the real native transition. Files: `spiral.go` and
existing `spiral_test.go`; owner: lead; no new dependencies, strings, test
files or UI shard entries. Verify with the named test, complete Spiral race
tests, GoLand and native exit/resize/reopen. The first canonical gate was
deliberately interrupted after this finding; one fresh full gate will cover
the fix. This is the reason for exceeding the original single-gate budget.

Fullscreen regression RED: the existing test reported `{1 1}` instead of a
usable restored canvas. GREEN: the complete Spiral race package passes in
2.860s. Native repeat restores 960x600, resizes to 644x420, and returns to
3840x1600 fullscreen. Both changed files have complete, empty GoLand
inspections including warnings. `make build` and Windows/amd64 internal vet
pass. The preserved baseline ran for 609 default seconds with stable warm
RSS; a second sustained run uses the fixed executable. Exact measurements,
capture limits and the final gate are retained in
[the qualification report](../docs/spiral-qualification-2026-09-12.md).

Cost ledger for this continuation: zero spawns; all diagnosis, implementation,
review and fixes lead-owned. One gate was interrupted for the native defect;
one replacement full gate covers the final source. The new guard was observed
RED before the two-line production fix; native reenactment confirms the
software driver's initialization check reaches the real failure mode.

The ticket AC identifiers are the acceptance matrix. Their test names are
proposed, not existing proof. Observe an intended RED failure for important
guards, then record focused GREEN results. Add the top-level TestHypnoTunnel
to exactly one UI shard; new subtests need no additional top-level assignment.
Every new test file gets its exact Qodana DuplicatedCode exclusion immediately.
Keep architecture and locale parity current in each slice.

No new dependencies or assets are planned. Verify the actual final dependency
closure and existing distribution notices before any release-ready claim;
record exact versions and applicable obligations if the implementation changes
that closure.

Native evidence must record source count, presets, controls, display/backend,
capture location, run duration and a candid visual verdict. Test tall, wide,
transparent and static GIF previews; both rotation signs; Follow at edges;
small/mixed/all-failed lists; order changes; resize; and close/reopen. Distinguish
retained previews/GPU textures from transient decoder memory and expected
decode waits from scheduler pauses. Do not claim untested GL/GLES platforms.

The prior local amd64/seccomp failure in todos.md is context, not an automatic
waiver. If it recurs, retain actual output and separate it from feature results.
Inspect every changed Go file in GoLand and dispose of all confirmed findings;
unavailable inspections remain unverified.

Before finishing, record per-ticket outcomes, final tuning, tests/durations,
native evidence, inspection scope, and final diff review here. Move the accepted
plan to finished_refactorings/ only once implementation is complete and update
its links at that time.

## Routing and cost ledger

| Work | Route | Result |
|---|---|---|
| Earlier source/duplicate/shader feasibility reconnaissance | Bounded read-only scouts | Findings recorded in the existing research and original plan. |
| Current motion facts: winding, Follow, clock and frame dispatch | One bounded read-only scout | Confirmed against current Spiral source; facts incorporated above. |
| Current spec/ticket/plan refinement | Lead | Same approved 01 -> 02 -> {03,04} -> 05 graph; personal visual direction and contradictions resolved. |
| Implementation, review, fixes and visual qualification | Lead | Implementation and focused review complete; full gate completed with known environment failures; remaining native evidence below. One implementation-phase read-only scout; zero implementation/review delegates. |

## Implementation evidence

Implementation started with the approved seams: canonical preview decoding,
viewer trigger/snapshot integration, and the real Spiral session/renderer.
One read-only scout locates current trigger and shutdown wiring while the lead
implements. Each slice's RED/GREEN and native evidence is recorded below.


### Delivered implementation and RED/GREEN evidence (2026-09-12)

- **01:** Viewer owns Spiral and captures one canonical duplicate visibility
  over the main-order list. Help emits a callback. RED: missing callback and
  bounded-preview API; GREEN: real manual/gesture integration, highest native
  resolution, unknown groups, frozen membership, oriented/static GIF previews.
- **02:** Three fixed sampler slots, serial 512px previews, continuous Main order,
  live speed/gap controls, whole-card retirement and premultiplied depth layering.
  RED: stream stopped after its first source; GREEN: cyclic stream, capacity,
  immediate overdue admission without catch-up, live gap deadline and all-failed
  backoff/recovery. Existing grid preview size remains 200px.
- **03:** Shuffled cycles, order revisions, batch profiles lasting 3–7 successful
  admissions, bounded adjacent drift, varied launch bearings and order/randomness
  controls. RED: missing profile state/control; GREEN: six full shuffled cycles,
  preserved live flights on order change, profile ranges and batch lengths.
- **04:** Session/order checks at delivery, one frame acknowledgement at a time,
  one decode lane across reopen, responsive cancellation and scrollable controls.
  RED: reverse-then-pause lost the last direction; small panel overflowed.
  GREEN: both corrected, queued old frames/held reads discarded, texture references
  cleared on close, long elapsed time and resized flight age retained.
- Deliberately bypassed duplicate visibility, centre clearance and queued-frame
  generation checks. All three corresponding guards failed for the intended
  reasons, then source was restored. Output: `.scratch/hypno-spiral-tunnel/guard-red.txt`.
- The first race run exposed an overconstrained test: Follow can legitimately
  move a live card entirely out of view and retire it. The age-preservation
  fixture now keeps the card in view. No production change was needed.

Focused command:
`go test -race ./internal/ui/spiral ./internal/ui/help ./internal/imaging ./internal/ui -run 'Test(Tunnel|HypnoTunnel|Help_Secret|LoadThumbnailContext_CancelsSourceRead)' -count=1`

GREEN package results: Spiral 2.580s, Help 1.895s, Imaging 2.336s, UI 11.721s.
The complete Spiral package also passed before the final added lifecycle guards.
Locale parity and both no-Unicode-arrow guards passed. Shard validation reports
682 runnable UI tests across three shards; TestHypnoTunnel is in ui-1. Qodana's
exact new-file exclusions are synchronized. `make build` produced `bin/picfetch`.

### Review and implementation clarifications

Lead reviewed the working changes against HEAD, AGENTS.md and all five tickets;
no review was delegated. GoLand inspected all 22 changed Go files including tests
with `errorsOnly:false`. Fixed three potential nil dereferences in test failure
paths, and made the existing generic YCbCr fallback explicit to resolve the
missing-switch-cases warning. Reinspection reports no findings in these files.

The tunnel uses one monotonic elapsed clock for admissions, retirement and GPU
flight age. It rebases only its own uniform every 60 seconds, rewriting live
birth offsets in the same UI frame. The established background shader animation
continues independently, preserving its existing phase/behavior. This is the
concrete implementation of the spec's permitted rebase.

No dependencies, native libraries, or shipped assets were added. `go.mod` and
`go.sum` are unchanged. The existing renderer is fyne.io/fyne/v2 v2.8.0 and the
existing scaler is golang.org/x/image v0.45.0. Their source/license entries are
in THIRD-PARTY-NOTICES.md (BSD-3-Clause); macOS packaging copies that file and
LICENSE into the app Resources (Makefile package-mac). Existing decoder/native
runtime obligations remain those recorded in the same notices; no new runtime
or model download is introduced by this feature.

### Native trial and tuning

Used an isolated application ID `io.github.frathe.picfetch.tunneltrial`, leaving
normal PicFetch preferences/session storage separate. The scratch launcher uses
production ui.Run and five repository assets: header.jpg, TaneWithFrame.webp,
draganddrop.png, trane_digging.webp, and picfetch_functionality.gif. Desktop macOS
OpenGL rendered the actual shader on the current Retina display. A native
screenshot is retained at `.scratch/hypno-spiral-tunnel/native/defaults-native.jpg`.

The default session opened through the real manual secret around 00:39 local
and remained running until Ronin's close around 00:52. Intermittent native checks
showed cycling upright photos, translucent detail, soft edges and a clear centre.
This elapsed run was not continuously watched or instrumented for retained RSS.
Ronin independently watched it and reported: **“it looks very calming”**. Keep
speed 1.00x, gap 2.50s, randomness 35%, base duration 9s, bend 0.35 radians,
long-edge growth 0.10 to 0.25 of the shorter screen dimension, core radius 0.08,
and radial base opacity 0.15 to 0.85. No tuning change followed that verdict.

Native recording through QuickTime/Screenshot did not yield a usable video;
SystemUIServer inspection timed out. A 30–60 second moving artifact, measured
retained-memory plateau, exhaustive native extremes, and other GL/GLES backends
remain unverified. Screenshots and automated geometry checks do not close those
qualification items. No release or commit is claimed.

### Final repository gate

`make verify` completed with exit 2; output: `.scratch/hypno-spiral-tunnel/verify.log`.
Format, TUF, exclusion parity, vet and build passed. Docker race artifacts:
`.scratch/race-runs/20260911T225133Z-dxnltk/`. The known local Linux/amd64 seccomp
errors recurred in similarity.TestLinuxWorkerIsolation and
scripts/explorereval.TestAssetInstall. These are the only failed packages.
All three UI partitions passed: ui-1 386.531s, ui-2 402.445s, ui-3 415.794s;
TestHypnoTunnel passed in Docker (11.150s). Imaging passed in 42.789s and
Help in 49.839s. No clean full-gate result is claimed.
This plan stays active until the outstanding qualification evidence is closed.


Final review addendum: an actual Scroll event in the compact panel failed to
refresh the idle timer. Added viewport.OnScrolled activity tracking after
observing that RED; the complete Spiral race suite is GREEN (2.786s), both
changed control files have clean GoLand reinspection, and make verify-build
plus make build pass again. The full Docker run started before this last
control-only change; its Spiral package passed (2.300s). Final-control coverage
is the subsequent complete native Spiral race suite. No other production code
changed during that gate.

The rebuilt native trial also displayed the new panel correctly. An attempted
automated extremes sweep did not reliably move controls to the requested values;
it is not counted as an extremes pass. The successful user-observed default
trial remains the native visual evidence.

Temporary trial cleanup was confirmed through the native app list: PicFetch
Tunnel Trial and Screenshot are no longer running. No files were committed.
