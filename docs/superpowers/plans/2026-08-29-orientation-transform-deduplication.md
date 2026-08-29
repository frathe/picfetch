# Orientation Transform Deduplication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve Qodana's four `DuplicatedCode` findings in
`internal/imaging/orientation.go` without changing pixels, public APIs, or the
performance characteristics of the per-pixel hot path.

**Architecture:** First lock down the five private transforms with a
non-zero-origin characterization test and a repeatable benchmark. Then replace
the five copied loops with one affine cursor loop whose transform selection is
performed once, outside the pixel loops. Keep that refactor only if it passes
the correctness and benchmark gates; otherwise restore the direct loops and
apply narrow source-level suppressions to Qodana's four reported copies.

**Tech Stack:** Go 1.26.7 module (verified locally with Go 1.27.0), standard
library `image` and `testing`, Qodana Go 2026.2 / GoLand inspections, existing
Makefile verification targets.

**Spec:** `todos.md` section
`Qodana: orientation.go's five transforms differ only by coordinate map`.

## Global Constraints

- Read `AGENTS.md` and `ARCHITECTURE.md` before editing; the latter is the
  authoritative package map.
- Preserve the signatures and semantics of `ApplyOrientation`, `RotateSteps`,
  `flipH`, `flipV`, `rotate180`, `rotate90CW`, and `rotate270CW`.
- Preserve zero-origin output bounds, source-bound offset handling, exact pixel
  placement, return type (`*image.RGBA` behind `image.Image`), and the identity
  paths returning the original image.
- Do not put a function value, interface method, transform switch, or branch in
  the per-pixel inner loop.
- Do not introduce dependencies, generics, code generation, assembly, unsafe
  code, or format-specific fast paths for this Qodana cleanup.
- Do not edit `qodana.yaml`; a fallback suppression must live next to the code
  and explain why the duplication is intentional.
- Do not add `TODO` or `FIXME` comments. Open work belongs in `todos.md`.
- Do not edit `ARCHITECTURE.md`: no package is added, removed, renamed, or
  moved by this plan.
- Preserve the pre-existing staged/unstaged `.aiignore` changes and all other
  unrelated user work.
- Do not regenerate golden screenshots; this viewer-independent change does
  not affect UI layout.
- Agents and the parent must not run `git commit`. After each accepted task,
  the parent gives the user the suggested commit message and waits for
  confirmation before dispatching the next task.
- Dispatch one implementation agent at a time. The tasks share a worktree and
  are intentionally dependent; parallel edits would make benchmark baselines
  and review ownership ambiguous.

---

## Current State and Baseline

- `orientation.go` contains five identical bounds/allocation/nested-loop bodies
  whose only meaningful differences are output dimensions and destination
  coordinates.
- Qodana reports four copies at the first statements of `flipH`, `flipV`,
  `rotate90CW`, and `rotate270CW`; `rotate180` is the unreported reference copy.
- `orientation_test.go` covers EXIF orientations and quarter-turn wrapping, but
  its fixtures all start at `(0, 0)`. A shared mapper could therefore mishandle
  `Bounds().Min` without an existing test catching it.
- There is no orientation benchmark, even though the TODO explicitly requires
  checking the cost before consolidating the loops.
- Baseline command run while writing this plan:

  ```text
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps'
  ok github.com/frathe/picfetch/internal/imaging
  ```

## Design Options

### Option A — measured affine cursor loop (recommended)

Represent each transform as an output size, a starting destination coordinate,
and two two-dimensional strides: one stride for moving to the next source
column and one for moving to the next source row. Select that specification
once before allocating/copying, then use integer cursor increments in the
shared loop.

| Transform | Output | Start `(x,y)` | Column stride | Row stride |
|-----------|--------|-----------------|---------------|------------|
| horizontal flip | `w × h` | `(w-1, 0)` | `(-1, 0)` | `(0, 1)` |
| vertical flip | `w × h` | `(0, h-1)` | `(1, 0)` | `(0, -1)` |
| 180° | `w × h` | `(w-1, h-1)` | `(-1, 0)` | `(0, -1)` |
| 90° clockwise | `h × w` | `(h-1, 0)` | `(0, 1)` | `(-1, 0)` |
| 270° clockwise | `h × w` | `(0, w-1)` | `(0, -1)` | `(1, 0)` |

This removes the duplicated traversal without a callback or inner-loop switch.
It does add two cursor increments per pixel, so the benchmark gate—not taste—
decides whether it stays.

### Option B — keep direct loops and suppress the four findings

Leave the production code structurally unchanged and add
`//goland:noinspection DuplicatedCode` at the four reported functions, each
with a short explanation that a shared callback/branch would tax every pixel.
This is the lowest-risk fallback and matches the repository's existing policy
of keeping justified Qodana suppressions beside the code.

### Option C — mapper callback, generic strategy, or code generation (rejected)

- A `func(x, y, w, h int) (int, int)` mapper adds an indirect call for every
  pixel.
- Generic mapper types risk dictionary dispatch and make performance depend on
  compiler specialization details.
- An enum switched inside the inner loop replaces duplication with five-way
  branching per pixel.
- Generated copies retain the duplication and add a maintenance tool solely to
  satisfy an inspection.

None improves this small, stable transform set enough to justify its cost.

## Open Decisions and Recommended Defaults

The plan is executable with the defaults below. Record overrides in this file
before dispatching Task 1 so every isolated subagent receives the same policy.

| ID | Decision | Recommended default | Alternative |
|----|----------|---------------------|-------------|
| D1 | Refactor or suppress immediately | Run Task 1 and the measured affine refactor in Task 2, then fall back only if it misses the gate. | Run Task 1's characterization steps, skip Task 2, and execute Task 3 directly; benchmark steps are optional unless D3 keeps the benchmark. |
| D2 | Performance gate | Same `allocs/op` and `B/op`; geomean time no worse than `+3%`; no individual transform with a statistically significant regression worse than `+5%`. | Require no measurable slowdown, or name another threshold. |
| D3 | Benchmark lifetime | Keep `orientation_benchmark_test.go` so future hot-loop changes remain measurable. | Use it only for this comparison and remove it before the implementation commit. |
| D4 | Commit cadence | Stop after every accepted task so the user can commit the reviewed unit before the next agent starts. | Review all tasks serially but suggest one combined final commit. |

## File Map

| File | Planned responsibility |
|------|------------------------|
| `internal/imaging/orientation.go` | Modify in Task 2 for the accepted affine implementation, or in Task 3 for fallback suppressions. |
| `internal/imaging/orientation_test.go` | Add source-bound-offset characterization before production code changes. |
| `internal/imaging/orientation_benchmark_test.go` | Add the stable five-transform benchmark when D3 uses the recommended default. |
| `todos.md` | Move the completed Qodana item to `Done → Internal` only after inspection and tests pass. |
| `ARCHITECTURE.md` | Unchanged. Its existing `orientation.go` entry remains accurate. |

## Subagent Routing

Every later `spawn_agent` call must use `fork_turns: "none"` and set both
`model` and `reasoning_effort` explicitly. Give the agent the Global
Constraints, the resolved D1–D4 choices, and only its own task section.

| Task | Model | Effort | Why |
|------|-------|--------|-----|
| 1 — characterization and baseline benchmark | `gpt-5.6-terra` | `high` | Precise, bounded Go test work; strong enough to catch bounds and fixture mistakes without spending frontier compute. |
| 2 — affine hot-loop implementation and performance analysis | `gpt-5.6-sol` | `xhigh` | The only compiler/performance-sensitive step; merits the strongest available coding model and deeper reasoning. |
| 3 — conditional suppression fallback | `gpt-5.6-luna` | `medium` | Mechanical restoration and four local comments with exact acceptance criteria. |
| 4 — inspection and backlog closure | `gpt-5.6-luna` | `medium` | Narrow validation/documentation update after the code path is settled. |

No task needs Opus, and Opus is not in this session's subagent allowlist. The
hardest work is separable and is routed to `gpt-5.6-sol` instead.

---

### Task 1: Characterize Source Bounds and Establish the Benchmark

**Subagent:** `gpt-5.6-terra` with `reasoning_effort: high`

**Files:**

- Modify: `internal/imaging/orientation_test.go`
- Create: `internal/imaging/orientation_benchmark_test.go` when D1 uses the
  measured route or D3 keeps the benchmark (keep or remove later according to
  D3)
- Do not modify: `internal/imaging/orientation.go`

**Interfaces:**

- Consumes: existing private transforms with signature
  `func(image.Image) image.Image`.
- Produces: `markedImageBounds(image.Rectangle) *image.RGBA`,
  `TestOrientationTransformsNonZeroBounds`, and
  `BenchmarkOrientationTransforms`.

- [ ] **Step 1: Confirm the current focused tests are green**

  Run:

  ```bash
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps'
  ```

  Expected: PASS. Stop and report if it fails; this task does not repair an
  existing pixel bug.

- [ ] **Step 2: Generalize the marked-image fixture without changing callers**

  Replace the body of `markedImage` and add `markedImageBounds` immediately
  below it:

  ```go
  func markedImage(w, h int) *image.RGBA {
      return markedImageBounds(image.Rect(0, 0, w, h))
  }

  func markedImageBounds(bounds image.Rectangle) *image.RGBA {
      img := image.NewRGBA(bounds)

      for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
          for x := bounds.Min.X; x < bounds.Max.X; x++ {
              img.SetRGBA(x, y, color.RGBA{
                  R: uint8(x - bounds.Min.X),
                  G: uint8(y - bounds.Min.Y),
                  A: 255,
              })
          }
      }

      return img
  }
  ```

  Keep the existing `markedImage(w, h)` signature so the current tests remain
  unchanged.

- [ ] **Step 3: Add the non-zero-source-bounds characterization test**

  Add this test after `at` and before `TestApplyOrientation`:

  ```go
  func TestOrientationTransformsNonZeroBounds(t *testing.T) {
      sourceBounds := image.Rect(5, 7, 8, 9)
      src := markedImageBounds(sourceBounds)
      w, h := sourceBounds.Dx(), sourceBounds.Dy()

      tests := []struct {
          name      string
          transform func(image.Image) image.Image
          wantBounds image.Rectangle
          dest      func(x, y int) (int, int)
      }{
          {"flip horizontal", flipH, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return w - 1 - x, y }},
          {"flip vertical", flipV, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return x, h - 1 - y }},
          {"rotate 180", rotate180, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }},
          {"rotate 90 clockwise", rotate90CW, image.Rect(0, 0, h, w), func(x, y int) (int, int) { return h - 1 - y, x }},
          {"rotate 270 clockwise", rotate270CW, image.Rect(0, 0, h, w), func(x, y int) (int, int) { return y, w - 1 - x }},
      }

      for _, test := range tests {
          t.Run(test.name, func(t *testing.T) {
              got := test.transform(src)
              if got.Bounds() != test.wantBounds {
                  t.Fatalf("bounds = %v, want %v", got.Bounds(), test.wantBounds)
              }

              for y := range h {
                  for x := range w {
                      destX, destY := test.dest(x, y)
                      gotX, gotY := at(t, got, destX, destY)
                      if gotX != x || gotY != y {
                          t.Errorf("source (%d,%d): pixel at (%d,%d) = (%d,%d), want (%d,%d)",
                              x, y, destX, destY, gotX, gotY, x, y)
                      }
                  }
              }
          })
      }
  }
  ```

  This is a characterization test, so it must pass before the refactor. Its
  purpose is to make a later regression fail, not to manufacture a RED state.

- [ ] **Step 4: Run the complete orientation test set**

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/imaging/orientation_test.go
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps|TestOrientationTransformsNonZeroBounds'
  ```

  Expected: PASS.

- [ ] **Step 5: Add a stable benchmark for all five private transforms**

  Run this step when D1 uses the measured route or D3 keeps the benchmark. If
  D1 selects immediate suppression and D3 selects a temporary benchmark, skip
  Steps 5–6.

  Create `internal/imaging/orientation_benchmark_test.go`:

  ```go
  package imaging

  import (
      "image"
      "runtime"
      "testing"
  )

  func BenchmarkOrientationTransforms(b *testing.B) {
      src := markedImage(640, 480)
      tests := []struct {
          name      string
          transform func(image.Image) image.Image
      }{
          {"flip-horizontal", flipH},
          {"flip-vertical", flipV},
          {"rotate-180", rotate180},
          {"rotate-90-clockwise", rotate90CW},
          {"rotate-270-clockwise", rotate270CW},
      }

      for _, test := range tests {
          b.Run(test.name, func(b *testing.B) {
              b.ReportAllocs()
              var got image.Image
              for b.Loop() {
                  got = test.transform(src)
              }
              runtime.KeepAlive(got)
          })
      }
  }
  ```

- [ ] **Step 6: Capture the reviewed pre-refactor benchmark**

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/imaging/orientation_benchmark_test.go
  go test -timeout 5m -run '^$' -bench '^BenchmarkOrientationTransforms$' \
    -benchmem -benchtime=500ms -count=10 ./internal/imaging \
    > /tmp/picfetch-orientation-before.txt
  test -s /tmp/picfetch-orientation-before.txt
  ```

  Do not run the benchmark under `-race`; the race instrumentation would make
  the timing comparison meaningless.

- [ ] **Step 7: Parent review and fix-up gate**

  The parent reads the full diff and checks:

  - production code is untouched;
  - every primitive is covered independently;
  - the source fixture has a non-zero `Bounds().Min`;
  - all destination pixels are checked;
  - benchmark setup happens outside the measured loop;
  - `.aiignore` and unrelated files are untouched.

  The parent fixes any drift with `apply_patch`, reruns Steps 4 and 6, and only
  then accepts the task.

- [ ] **Step 8: Suggested commit (parent does not commit)**

  If D3 keeps the benchmark:

  ```text
  test imaging orientation transforms across offset bounds

  Characterize each private transform independently and add a repeatable
  benchmark before consolidating the per-pixel loops.
  ```

  If D3 makes the benchmark temporary, suggest the same commit with only
  `orientation_test.go` staged; leave the benchmark file available for Task 2
  but out of the user's commit.

---

### Task 2: Implement and Measure the Affine Cursor Refactor

**Condition:** Run when D1 uses the recommended measured-refactor route.

**Subagent:** `gpt-5.6-sol` with `reasoning_effort: xhigh`

**Files:**

- Modify: `internal/imaging/orientation.go`
- Read/test: `internal/imaging/orientation_test.go`
- Read/benchmark: `internal/imaging/orientation_benchmark_test.go`

**Interfaces:**

- Consumes: the five existing private transform functions and Task 1's tests
  and benchmark.
- Produces: private `pixelTransform`, `pixelTransformSpec`,
  `pixelTransformSpecFor`, and `applyPixelTransform`; the five existing
  function names become thin wrappers with unchanged signatures.

- [ ] **Step 1: Verify Task 1's baseline artifacts**

  ```bash
  test -s /tmp/picfetch-orientation-before.txt
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps|TestOrientationTransformsNonZeroBounds'
  ```

  Expected: the file exists and tests PASS. If the benchmark file was lost,
  recapture it from the reviewed direct-loop tree before editing production
  code.

- [ ] **Step 2: Add the transform specification and shared copy loop**

  In `orientation.go`, insert the following after `RotateSteps` and replace the
  five copied bodies with the wrappers shown below. Keep the existing
  `rotate90CW` and `rotate270CW` doc comments immediately above their wrappers.

  ```go
  type pixelTransform uint8

  const (
      pixelFlipH pixelTransform = iota
      pixelFlipV
      pixelRotate180
      pixelRotate90CW
      pixelRotate270CW
  )

  type pixelTransformSpec struct {
      outW, outH       int
      startX, startY   int
      columnDX, columnDY int
      rowDX, rowDY     int
  }

  func pixelTransformSpecFor(kind pixelTransform, w, h int) pixelTransformSpec {
      switch kind {
      case pixelFlipH:
          return pixelTransformSpec{
              outW: w, outH: h,
              startX: w - 1, startY: 0,
              columnDX: -1, columnDY: 0,
              rowDX: 0, rowDY: 1,
          }
      case pixelFlipV:
          return pixelTransformSpec{
              outW: w, outH: h,
              startX: 0, startY: h - 1,
              columnDX: 1, columnDY: 0,
              rowDX: 0, rowDY: -1,
          }
      case pixelRotate180:
          return pixelTransformSpec{
              outW: w, outH: h,
              startX: w - 1, startY: h - 1,
              columnDX: -1, columnDY: 0,
              rowDX: 0, rowDY: -1,
          }
      case pixelRotate90CW:
          return pixelTransformSpec{
              outW: h, outH: w,
              startX: h - 1, startY: 0,
              columnDX: 0, columnDY: 1,
              rowDX: -1, rowDY: 0,
          }
      case pixelRotate270CW:
          return pixelTransformSpec{
              outW: h, outH: w,
              startX: 0, startY: w - 1,
              columnDX: 0, columnDY: -1,
              rowDX: 1, rowDY: 0,
          }
      default:
          panic("imaging: unknown pixel transform")
      }
  }

  func applyPixelTransform(src image.Image, kind pixelTransform) image.Image {
      bounds := src.Bounds()
      w, h := bounds.Dx(), bounds.Dy()
      spec := pixelTransformSpecFor(kind, w, h)
      out := image.NewRGBA(image.Rect(0, 0, spec.outW, spec.outH))

      rowStartX, rowStartY := spec.startX, spec.startY
      for sourceY := range h {
          destX, destY := rowStartX, rowStartY
          for sourceX := range w {
              out.Set(destX, destY, src.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY))
              destX += spec.columnDX
              destY += spec.columnDY
          }
          rowStartX += spec.rowDX
          rowStartY += spec.rowDY
      }

      return out
  }

  func flipH(src image.Image) image.Image {
      return applyPixelTransform(src, pixelFlipH)
  }

  func flipV(src image.Image) image.Image {
      return applyPixelTransform(src, pixelFlipV)
  }

  func rotate180(src image.Image) image.Image {
      return applyPixelTransform(src, pixelRotate180)
  }

  func rotate90CW(src image.Image) image.Image {
      return applyPixelTransform(src, pixelRotate90CW)
  }

  func rotate270CW(src image.Image) image.Image {
      return applyPixelTransform(src, pixelRotate270CW)
  }
  ```

  `pixelTransformSpecFor` is the only switch, and it runs once per image. The
  two nested loops contain no callback, interface strategy, switch, or branch
  other than loop control.

- [ ] **Step 3: Format and run correctness tests**

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/imaging/orientation.go
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps|TestOrientationTransformsNonZeroBounds'
  go test -timeout 2m -race -count=1 ./internal/imaging
  ```

  Expected: PASS.

- [ ] **Step 4: Capture the post-refactor benchmark**

  ```bash
  go test -timeout 5m -run '^$' -bench '^BenchmarkOrientationTransforms$' \
    -benchmem -benchtime=500ms -count=10 ./internal/imaging \
    > /tmp/picfetch-orientation-after.txt
  test -s /tmp/picfetch-orientation-after.txt
  go list -m -f '{{.Version}}' golang.org/x/perf@latest
  go run golang.org/x/perf/cmd/benchstat@latest \
    /tmp/picfetch-orientation-before.txt \
    /tmp/picfetch-orientation-after.txt
  ```

  `go run ...@latest` is a temporary analysis tool invocation; it must not add
  `golang.org/x/perf` to `go.mod` or `go.sum`. Record the resolved benchstat
  version and its complete comparison in the task report.

- [ ] **Step 5: Apply the D2 performance gate**

  Accept the candidate only when all are true:

  - `allocs/op` is unchanged for all five transforms;
  - `B/op` is unchanged for all five transforms;
  - benchstat geomean time is no worse than D2's resolved geomean threshold;
  - no individual transform has a statistically significant regression worse
    than D2's resolved individual-transform threshold;
  - if benchstat reports noisy/inconclusive samples near a threshold, repeat
    both sides with `-benchtime=1s -count=20` before deciding.

  If the candidate misses any gate, do not optimize beyond the agreed scope.
  Mark Task 2 rejected and proceed to Task 3.

- [ ] **Step 6: Parent review and fix-up gate**

  The parent compares each case against the mapping table and verifies:

  - `ApplyOrientation` and `RotateSteps` are byte-for-byte unchanged;
  - source reads still add `bounds.Min.X/Y`;
  - output dimensions and all five starts/strides match the table;
  - the inner loop has no callback or transform branch;
  - no public API or dependency changed;
  - the benchmark comparison satisfies D2.

  The parent fixes naming, layout, or test issues with `apply_patch` and reruns
  Steps 3–5. A mapping or performance-gate failure rejects the refactor rather
  than inviting an unplanned micro-optimization.

- [ ] **Step 7: Suggested commit if accepted (parent does not commit)**

  ```text
  share orientation pixel traversal through affine cursors

  Select output geometry and strides once per transform, then use one direct
  pixel-copy loop without callback dispatch or inner-loop branching.
  ```

  If rejected, do not suggest or create this commit; continue directly to
  Task 3 with the candidate diff still visible for controlled restoration.

---

### Task 3: Restore Direct Loops and Add Narrow Suppressions (Conditional)

**Condition:** Run only if D1 selects immediate suppression or Task 2 fails
the resolved D2 gate.

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `internal/imaging/orientation.go`
- Test: existing orientation tests from Task 1

**Interfaces:**

- Consumes: the reviewed Task 1 tree and, when applicable, Task 2's rejected
  diff.
- Produces: the original five direct loops plus four local
  `DuplicatedCode` suppressions. No signature or pixel change.

- [ ] **Step 1: Restore the reviewed direct-loop implementation when needed**

  Read the committed Task 1 version without changing the worktree:

  ```bash
  git show HEAD:internal/imaging/orientation.go
  ```

  If Task 2 changed `orientation.go`, use `apply_patch` to remove
  `pixelTransform`, `pixelTransformSpec`, `pixelTransformSpecFor`, and
  `applyPixelTransform`, and restore the five function bodies exactly from that
  output. Do not use `git checkout`, `git restore`, or `git reset`; unrelated
  user changes must remain recoverable.

- [ ] **Step 2: Add suppressions only to Qodana's four reported copies**

  Add `//goland:noinspection DuplicatedCode` immediately above `flipH`,
  `flipV`, `rotate90CW`, and `rotate270CW`. Give each function a name-leading
  explanation. Use these exact comments while leaving every loop statement
  unchanged:

  ```go
  // flipH keeps a direct pixel loop because a shared coordinate callback or
  // transform branch would add work to every pixel in this hot path.
  //goland:noinspection DuplicatedCode
  func flipH(src image.Image) image.Image {
  ```

  ```go
  // flipV keeps a direct pixel loop because a shared coordinate callback or
  // transform branch would add work to every pixel in this hot path.
  //goland:noinspection DuplicatedCode
  func flipV(src image.Image) image.Image {
  ```

  Extend the existing rotation comments as follows:

  ```go
  // rotate90CW rotates the image 90 degrees clockwise, swapping width and
  // height. It keeps a direct pixel loop because a shared coordinate callback
  // or transform branch would add work to every pixel in this hot path.
  //goland:noinspection DuplicatedCode
  func rotate90CW(src image.Image) image.Image {
  ```

  ```go
  // rotate270CW rotates the image 270 degrees clockwise (90 counterclockwise),
  // swapping width and height. It keeps a direct pixel loop because a shared
  // coordinate callback or transform branch would add work to every pixel in
  // this hot path.
  //goland:noinspection DuplicatedCode
  func rotate270CW(src image.Image) image.Image {
  ```

  Leave `rotate180` unsuppressed: it is Qodana's reference body rather than one
  of the four reported duplicates.

- [ ] **Step 3: Format and verify that behavior stayed identical**

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/imaging/orientation.go
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps|TestOrientationTransformsNonZeroBounds'
  go test -timeout 2m -race -count=1 ./internal/imaging
  ```

  Expected: PASS.

- [ ] **Step 4: Parent review and fix-up gate**

  The parent verifies the production diff contains comments only relative to
  the Task 1 baseline, all `out.Set` expressions are unchanged, exactly four
  directives exist, the directive spelling matches the repository convention,
  and no `qodana.yaml` change exists. Fix drift and rerun Step 3.

- [ ] **Step 5: Suggested commit (parent does not commit)**

  ```text
  suppress honest duplication in orientation pixel loops

  Keep the five direct transforms to avoid callback or branch overhead in the
  per-pixel hot path, and document the four Qodana suppressions at source.
  ```

---

### Task 4: Confirm the Inspection Result and Close the Backlog Item

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `todos.md`
- Optionally remove: `internal/imaging/orientation_benchmark_test.go` only when
  D3 explicitly selected a temporary benchmark
- Do not modify: `ARCHITECTURE.md`

**Interfaces:**

- Consumes: either Task 2's accepted affine implementation or Task 3's reviewed
  suppression implementation.
- Produces: a clean orientation inspection and backlog text matching the path
  actually shipped.

- [ ] **Step 1: Verify the inspection before claiming completion**

  Call GoLand's `get_file_problems` tool with
  `filePath: "internal/imaging/orientation.go"`, `errorsOnly: false`, and a
  120-second timeout. Confirm the response is not timed out and contains no
  `DuplicatedCode` finding. Also run:

  ```bash
  rg -n 'goland:noinspection DuplicatedCode|func applyPixelTransform' internal/imaging/orientation.go
  ```

  Expected:

  - accepted-refactor path: one `applyPixelTransform` and no duplication
    suppressions;
  - fallback path: four `DuplicatedCode` suppressions and no
    `applyPixelTransform`.

  If the IDE still reports a duplicate, stop. Do not close the TODO until the
  inspection is actually clean.

- [ ] **Step 2: Apply the resolved D3 benchmark policy**

  - Permanent benchmark: leave
    `internal/imaging/orientation_benchmark_test.go` in place.
  - Temporary benchmark: remove the agent-created benchmark file with
    `apply_patch`, then rerun the focused race tests. Do not delete any file
    that predates Task 1.

- [ ] **Step 3: Move the TODO entry to Done → Internal**

  Delete the complete
  `### Qodana: orientation.go's five transforms differ only by coordinate map`
  section from `## TODO`.

  For the accepted-refactor path, add this bullet under `## Done` →
  `#### Internal`:

  ```markdown
  - Orientation transforms: `applyPixelTransform` owns the shared bounds,
    allocation, and pixel-copy loop. Five affine cursor specifications select
    output geometry and strides before traversal, with no callback or transform
    branch in the per-pixel loop; characterization tests cover offset source
    bounds and the benchmark gate protects the hot path.
  ```

  For the suppression path, add this bullet instead:

  ```markdown
  - Orientation transforms: the five direct pixel loops stay separate to avoid
    callback dispatch or transform branching in the per-pixel hot path. The
    four reported `DuplicatedCode` copies carry source-local suppressions and
    explanations; characterization tests cover offset source bounds.
  ```

  Do not edit any other TODO item.

- [ ] **Step 4: Run focused verification**

  ```bash
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestApplyOrientation|TestRotateSteps|TestOrientationTransformsNonZeroBounds'
  go test -timeout 2m -race -count=1 ./internal/imaging
  ```

  Expected: PASS.

- [ ] **Step 5: Parent review and fix-up gate**

  The parent checks that the Done text describes the branch actually present,
  the TODO section appears exactly once (under Done, not TODO), benchmark policy
  D3 was honored, `ARCHITECTURE.md` is untouched, and the inspection is clean.
  Fix documentation drift and rerun Step 4.

- [ ] **Step 6: Suggested commit (parent does not commit)**

  ```text
  close the orientation transform Qodana todo
  ```

---

## Parent Controller Protocol After Every Subagent

The parent—not another subagent—owns integration quality after each task.

1. Run `git status --short` and compare it with the pre-task snapshot.
2. Read `git diff --check` and the complete diff for every changed file.
3. Reject edits outside the task's file allowlist; preserve `.aiignore` and all
   other pre-existing changes.
4. Check the task-specific review list and correct issues directly with
   `apply_patch`.
5. Rerun the task's focused commands after every parent fix.
6. Summarize what changed, paste relevant test/benchmark results, give the
   suggested commit message, and stop for the user's confirmation when D4 uses
   the recommended commit cadence.
7. Dispatch the next fresh agent only after the current tree is reviewed and,
   when applicable, the user confirms their commit landed.

If an implementer needs a correction, use `followup_task` on that same agent
for the first fix round. The parent still performs the final diff review and
may make small integration fixes itself.

## Final Repository Verification

After Task 4 and before claiming the overall work complete, the parent runs the
repository's CI-equivalent gates from the root:

```bash
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

Then verify:

```bash
git diff --check
git status --short
```

Acceptance requires:

- all commands pass;
- GoLand reports no orientation `DuplicatedCode` finding;
- the D2 performance gate passed, or the suppression fallback is the shipped
  implementation;
- `todos.md` accurately records the chosen outcome;
- no unrelated user changes were altered;
- no failed golden images or build artifacts were added.

Qodana CI itself still requires the repository's `QODANA_TOKEN`; local GoLand
inspection is the immediate verification, and the next authenticated CI run is
the end-to-end confirmation.

## Out of Scope

- Combining EXIF orientations 5 and 7 into single-pass transpose/transverse
  operations.
- Optimizing `image.Image.At`, `image.RGBA.Set`, or adding concrete-type fast
  paths.
- Changing rotation save/export behavior or UI display composition.
- Addressing the next TODO (`internal/uitest` APP1 wrapping), other Qodana
  findings, or Qodana CI reporting configuration.
- Updating `ARCHITECTURE.md`, translations, screenshots, dependencies, or
  release metadata.
