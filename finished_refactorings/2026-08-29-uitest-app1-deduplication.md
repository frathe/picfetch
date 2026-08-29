# Uitest APP1 Deduplication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the duplicated Exif APP1 framing/splice tails in PicFetch's
capture-date and GPS JPEG test fixtures with one private, byte-tested helper,
then close the matching Qodana backlog item only after GoLand reports the file
clean.

**Architecture:** Keep TIFF construction inside `CaptureDateJPEG` and
`GPSJPEG`, where the tag-specific layouts belong. Move only the shared
`Exif\x00\x00` payload framing, APP1 length encoding, and post-SOI splice into
`wrapAPP1(data, tiff []byte) []byte`; a direct unit test locks that byte
contract while existing capture-date and GPS consumers protect fixture
semantics.

**Tech Stack:** Go 1.26.7 module (verified locally with Go 1.27.0), standard
library `bytes`, `encoding/binary`, and `testing`, Qodana Go 2026.2 / GoLand
inspections, existing Makefile verification targets.

**Spec:** `todos.md` section `Qodana: uitest.go wraps an APP1 segment twice`.

## Global Constraints

- Read `AGENTS.md` and `ARCHITECTURE.md` before editing.
- Keep `internal/uitest` test-only and viewer-independent. Do not move this
  helper into `internal/imaging` or make production code import it.
- Preserve the exported signatures and behavior of `CaptureDateJPEG`,
  `GPSJPEG`, and `TempGPSJPEGURI`.
- Add exactly one private helper with the signature
  `wrapAPP1(data, tiff []byte) []byte` in `internal/uitest/uitest.go`.
- Preserve the current byte layout: JPEG SOI, APP1 marker `0xFF 0xE1`, a
  big-endian length equal to two length bytes plus the Exif payload, the
  `Exif\x00\x00` identifier, raw TIFF bytes, then the original JPEG bytes
  after SOI.
- Preserve the current preconditions instead of broadening the API: both
  callers pass a valid `EncodeJPEG` result and a small synthetic TIFF payload.
  Do not add validation, an error return, `testing.T`, or a maximum-size policy
  to the helper.
- Do not add a generic JPEG-segment framework. Production JPEG walking and
  metadata copying already live in `internal/imaging`; this helper has exactly
  two test-fixture callers.
- Do not introduce dependencies, generics, code generation, unsafe code, or
  Qodana suppressions for this cleanup.
- Do not edit `qodana.yaml`, translations, golden screenshots, dependencies,
  or production imaging code.
- Do not edit `ARCHITECTURE.md`: no package is added, removed, renamed, or
  moved, and its `internal/uitest` entry remains accurate.
- Do not add `TODO` or `FIXME` comments. Open work belongs in `todos.md`.
- Preserve `.aiignore` and all unrelated user work.
- Do not close the TODO until GoLand `get_file_problems` returns without a
  timeout and contains no `DuplicatedCode` finding for
  `internal/uitest/uitest.go`.
- Agents and the parent must not run `git commit` or alter the index. Suggested
  commit messages are handed to the user after review.
- Dispatch one implementation agent at a time. Review and correct each task
  before the next task begins; continue without a user checkpoint unless a
  real blocker or new design decision arises.

---

## Current State and Baseline

- `CaptureDateJPEG` builds a one-entry little-endian TIFF and then frames and
  splices it into a JPEG at `internal/uitest/uitest.go:296-304`.
- `GPSJPEG` builds a GPS sub-IFD and repeats the same framing/splice tail at
  `internal/uitest/uitest.go:497-505`.
- GoLand currently reports four `DuplicatedCode` weak warnings at lines 296,
  458, 497, and 547. Lines 296 and 497 are the repeated nine-line tail; the
  other two are secondary anchors in the same duplicate clusters.
- `internal/uitest` has no general fixture test file. The approved design adds
  `uitest_test.go` with a private helper-level byte test.
- Existing semantic consumers include:
  - `internal/filesort/filesort_test.go`'s
    `TestOrderedFiles_SortsByCaptureDate`, which consumes `CaptureDateJPEG`;
  - `internal/imaging/save_test.go`'s
    `TestExport_JPEGSourceKeepsMetadataOnJPEGDest`, which consumes `GPSJPEG`;
  - UI and EXIF-window tests that consume both fixture families and remain
    covered by the final repository suite.
- Baseline commands run while writing this plan:

  ```text
  go test -timeout 2m -race -count=1 ./internal/uitest
  ok github.com/frathe/picfetch/internal/uitest

  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  ok github.com/frathe/picfetch/internal/filesort

  go test -timeout 2m -race -count=1 ./internal/imaging -run '^TestExport_JPEGSourceKeepsMetadataOnJPEGDest$'
  ok github.com/frathe/picfetch/internal/imaging
  ```

  The filesort and imaging test binaries emit the existing macOS duplicate
  `-lobjc` linker warning. Treat that as baseline noise, but reject any new
  warning class introduced by this work.

## Approved Design and Rejected Alternatives

### Approved: private TIFF-to-Exif APP1 splice helper

`wrapAPP1` accepts the already encoded JPEG and the caller-specific TIFF
payload. It owns only the common representation boundary:

1. prepend the six-byte Exif identifier;
2. compute the JPEG segment length;
3. emit APP1 marker plus big-endian length;
4. splice after the source JPEG's two-byte SOI marker.

`CaptureDateJPEG` and `GPSJPEG` continue to own every TIFF tag, offset, byte
order, and value calculation. This removes the reported duplication without
coupling their unrelated TIFF layouts.

### Rejected: generic APPn/JPEG segment utility

A marker parameter, arbitrary payload API, validation, and segment-size error
contract would solve a broader problem than these two fixture builders have.
Production code already has its own tested JPEG segment abstractions, and
test-only code must not become their shadow implementation.

### Rejected: source suppression

Unlike the orientation hot loops, this duplicate test-fixture framing has no
performance reason to remain copied. A six-purpose-line helper is clearer and
cheaper to maintain than four Qodana suppressions.

## Resolved Decisions

| ID | Decision | Resolution |
|----|----------|------------|
| D1 | Direct helper test | Add `internal/uitest/uitest_test.go`; verify marker, high and low length bytes, Exif identifier, TIFF payload, and untouched JPEG remainder. |
| D2 | Validation/error policy | Preserve current trusted-fixture preconditions; no new validation or error return. |
| D3 | Scope of abstraction | Private `wrapAPP1(data, tiff []byte) []byte` with exactly two callers; no generic marker API. |
| D4 | Execution cadence | Run tasks continuously after go-ahead, with parent and independent review after each; stop only for a real blocker or newly required design decision. |
| D5 | Residual duplicate clusters | After the first clean refactor still left two unrelated IDE findings, the user explicitly authorized broadening the implementation to eliminate both. Reuse `uitest.TruncatedPNGHeader` from the imaging test and add one private byte-writing `littleEndianTIFF`; do not build a generic TIFF/IFD framework. |

## File Map

| File | Planned responsibility |
|------|------------------------|
| `internal/uitest/uitest.go` | Define private `wrapAPP1`; add the narrow `littleEndianTIFF` byte writer; switch the capture-date, RAW-preview, and GPS fixture writers to the shared primitives. |
| `internal/uitest/uitest_test.go` | Lock the exact APP1 framing and little-endian TIFF header/integer bytes. |
| `internal/imaging/loader_test.go` | Reuse `uitest.TruncatedPNGHeader` and remove its local duplicate. |
| `todos.md` | Move the Qodana item to Done → Internal only after inspection and tests pass. |
| `needs_refactoring.md` | Unchanged; neither duplicate cluster is an entry in this separate architectural-debt backlog. |
| `ARCHITECTURE.md` | Unchanged; no package or source responsibility moves. |

## Subagent Routing

Every `spawn_agent` call must use `fork_turns: "none"` and set both `model`
and `reasoning_effort` explicitly. Implementers must not spawn their own
subagents or reviewers.

| Role | Model | Effort | Reason |
|------|-------|--------|--------|
| Task 1 implementation | `gpt-5.6-luna` | `medium` | Two-file mechanical extraction with complete code and exact tests; the lower tier is sufficient. |
| Task 1 review | `gpt-5.6-terra` | `medium` | Byte offsets, JPEG length semantics, and behavior preservation merit a balanced reviewer. |
| Task 2 implementation | `gpt-5.6-luna` | `medium` | Inspection plus one exact backlog movement is narrow and mechanical. |
| Task 2 review | `gpt-5.6-terra` | `medium` | A balanced reviewer verifies the inspection evidence and documentation truthfulness. |
| Task 3 implementation | `gpt-5.6-luna` | `medium` | Replacing one local test helper with the existing shared fixture is mechanical. |
| Task 3 review | `gpt-5.6-terra` | `medium` | Review checks package direction, imports, and unchanged bomb-header behavior. |
| Task 4 implementation | `gpt-5.6-luna` | `medium` | The helper API and byte-level test are fully specified; migration stays inside two test-utility files. |
| Task 4 review | `gpt-5.6-terra` | `medium` | TIFF byte order, offsets, and three fixture families require balanced review judgment. |
| Task 5 implementation | `gpt-5.6-luna` | `medium` | Final inspection and exact backlog movement are mechanical after Tasks 3 and 4. |
| Task 5 review | `gpt-5.6-terra` | `medium` | Review verifies complete inspection evidence and truthful expanded Done text. |
| Final whole-change review | `gpt-5.6-sol` | `high` | The final review spans source, tests, inspection evidence, TODO accuracy, and repository constraints. |

No task needs Opus. The implementation is deliberately complete enough for
the lower-tier Luna agents; the parent retains integration decisions and uses
fresh Terra reviewers after each task.

---

### Task 1: Extract and Test the Shared APP1 Framing

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Create: `internal/uitest/uitest_test.go`
- Modify: `internal/uitest/uitest.go`
- Do not modify: `todos.md`, `ARCHITECTURE.md`, `qodana.yaml`, or any
  production package

**Interfaces:**

- Consumes: a valid JPEG byte slice beginning with its two-byte SOI marker and
  raw TIFF bytes built by an existing fixture.
- Produces: private
  `func wrapAPP1(data, tiff []byte) []byte`, used only by
  `CaptureDateJPEG` and `GPSJPEG`.

- [ ] **Step 1: Reconfirm the focused baseline and source ownership**

  Run:

  ```bash
  git status --short
  go test -timeout 2m -race -count=1 ./internal/uitest
  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  go test -timeout 2m -race -count=1 ./internal/imaging -run '^TestExport_JPEGSourceKeepsMetadataOnJPEGDest$'
  ```

  Expected: all tests pass; only the pre-existing duplicate `-lobjc` linker
  warning may appear. Record any pre-existing dirty paths and do not touch
  them.

- [ ] **Step 2: Add the failing byte-level helper test**

  Create `internal/uitest/uitest_test.go`:

  ```go
  package uitest

  import (
      "bytes"
      "encoding/binary"
      "testing"
  )

  func TestWrapAPP1_SplicesExifAfterSOI(t *testing.T) {
      jpegData := []byte{0xFF, 0xD8, 0xFF, 0xD9}
      tiff := bytes.Repeat([]byte{0xAB}, 250)

      got := wrapAPP1(jpegData, tiff)

      const (
          markerAndLengthBytes = 4
          exifID               = "Exif\x00\x00"
      )
      wantLen := len(jpegData) + markerAndLengthBytes + len(exifID) + len(tiff)
      if len(got) != wantLen {
          t.Fatalf("len(wrapAPP1()) = %d, want %d", len(got), wantLen)
      }

      if !bytes.Equal(got[:2], jpegData[:2]) {
          t.Errorf("SOI = % X, want % X", got[:2], jpegData[:2])
      }
      if !bytes.Equal(got[2:4], []byte{0xFF, 0xE1}) {
          t.Errorf("marker = % X, want FF E1", got[2:4])
      }

      gotLength := int(binary.BigEndian.Uint16(got[4:6]))
      wantLength := 2 + len(exifID) + len(tiff)
      if gotLength != wantLength {
          t.Errorf("APP1 length = %d, want %d", gotLength, wantLength)
      }

      payloadStart := 6
      tiffStart := payloadStart + len(exifID)
      tiffEnd := tiffStart + len(tiff)
      if string(got[payloadStart:tiffStart]) != exifID {
          t.Errorf("Exif identifier = %q, want %q", got[payloadStart:tiffStart], exifID)
      }
      if !bytes.Equal(got[tiffStart:tiffEnd], tiff) {
          t.Error("TIFF payload changed while wrapping APP1")
      }
      if !bytes.Equal(got[tiffEnd:], jpegData[2:]) {
          t.Errorf("JPEG remainder = % X, want % X", got[tiffEnd:], jpegData[2:])
      }
  }
  ```

  The 250-byte TIFF payload makes the APP1 length `258` (`0x0102`), so the
  test exercises both length bytes rather than only a zero high byte.

- [ ] **Step 3: Run the new test and verify RED**

  Run:

  ```bash
  go test -timeout 2m -race -count=1 ./internal/uitest -run '^TestWrapAPP1_SplicesExifAfterSOI$'
  ```

  Expected: build failure containing `undefined: wrapAPP1`. A different
  failure means the test itself must be corrected before production-helper
  code is added.

- [ ] **Step 4: Add the minimal private helper**

  Add this near the JPEG fixture builders in
  `internal/uitest/uitest.go`, before its first use:

  ```go
  // wrapAPP1 inserts a TIFF payload as an Exif APP1 segment immediately
  // after the JPEG SOI marker.
  func wrapAPP1(data, tiff []byte) []byte {
      seg := append([]byte("Exif\x00\x00"), tiff...)
      length := len(seg) + 2
      app1 := append([]byte{0xFF, 0xE1, byte(length >> 8), byte(length)}, seg...)

      out := append([]byte{}, data[:2]...)
      out = append(out, app1...)
      out = append(out, data[2:]...)

      return out
  }
  ```

  Do not add input validation or change the signature. Indexing `data[:2]`
  preserves the two callers' existing valid-JPEG precondition and failure
  behavior.

- [ ] **Step 5: Route both fixture builders through the helper**

  In `CaptureDateJPEG`, replace the complete repeated block after
  `tiff.Write(dateBytes)` with:

  ```go
  return wrapAPP1(data, tiff.Bytes())
  ```

  In `GPSJPEG`, replace the complete repeated block after
  `buf.Write(dmsRationals(lon))` with:

  ```go
  return wrapAPP1(data, buf.Bytes())
  ```

  Do not alter any TIFF constants, offsets, tag writes, coordinate encoding,
  or exported fixture signatures.

- [ ] **Step 6: Format and verify GREEN at the helper boundary**

  Run:

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/uitest/uitest.go internal/uitest/uitest_test.go
  go test -timeout 2m -race -count=1 ./internal/uitest -run '^TestWrapAPP1_SplicesExifAfterSOI$'
  ```

  Expected: PASS.

- [ ] **Step 7: Verify both semantic fixture families**

  Run:

  ```bash
  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  go test -timeout 2m -race -count=1 ./internal/imaging -run '^TestExport_JPEGSourceKeepsMetadataOnJPEGDest$'
  go test -timeout 5m -race -count=1 ./internal/uitest ./internal/filesort ./internal/imaging
  ```

  Expected: all commands pass. The targeted tests prove the two caller
  families still produce readable DateTime and GPS Exif; the package run
  catches other fixture consumers.

- [ ] **Step 8: Self-review the exact diff**

  Run:

  ```bash
  git diff --check
  git status --short
  git diff -- internal/uitest/uitest.go internal/uitest/uitest_test.go
  rg -n 'func wrapAPP1|return wrapAPP1|Exif\\x00\\x00' internal/uitest/uitest.go
  ```

  Expected source shape:

  - one private `wrapAPP1` definition;
  - one `Exif\x00\x00` framing literal inside that helper;
  - two `return wrapAPP1(...)` callers;
  - no repeated marker/length/splice tail;
  - no unrelated file change.

- [ ] **Step 9: Parent and independent review gate**

  The parent reads the full report and diff, verifies every slice boundary and
  APP1 length rule, confirms the test uses a non-zero high length byte, and
  checks that only the common tail moved. Dispatch a fresh
  `gpt-5.6-terra`/`medium` reviewer against a path-scoped review package.

  If either review finds a spec gap or Critical/Important quality issue,
  return the exact finding to the Task 1 implementer, require covering test
  output in its amended report, and dispatch a scoped re-review. Do not let
  the parent silently patch around the implementer.

- [ ] **Step 10: Suggested commit (agents and parent do not commit)**

  ```text
  deduplicate APP1 fixture framing

  Share one Exif APP1 wrapper between capture-date and GPS JPEG fixtures,
  with a byte-level test for marker placement and segment length.
  ```

---

### Task 2: Confirm the Inspection and Close the Backlog Item

> **Execution result:** the mandatory inspection correctly blocked before any
> `todos.md` edit. The APP1 duplicate was gone, but GoLand still reported an
> 11-line TIFF-writing fragment and a 30-line truncated-PNG fixture fragment.
> The user then authorized the scoped expansion in Tasks 3–5. Do not
> redispatch this original task; Task 5 replaces its backlog-closing steps.

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `todos.md`
- Inspect only: `internal/uitest/uitest.go`,
  `internal/uitest/uitest_test.go`
- Do not modify: `ARCHITECTURE.md`, `qodana.yaml`, implementation files, or
  unrelated TODO entries

**Interfaces:**

- Consumes: Task 1's reviewed private `wrapAPP1` and its two callers.
- Produces: a clean GoLand duplication inspection and a Done entry that
  accurately describes the shipped helper.

- [ ] **Step 1: Require a clean, non-timeout IDE inspection**

  Call GoLand `get_file_problems` with:

  ```json
  {
    "filePath": "internal/uitest/uitest.go",
    "errorsOnly": false,
    "timeout": 120000
  }
  ```

  Acceptance requires `timedOut` to be false or absent and no returned problem
  whose description contains `Duplicated code fragment` / `DuplicatedCode`.
  Record the complete structured response in the task report.

  If any duplicate remains, stop with `BLOCKED` and ask the user before
  broadening the refactor. Do not add suppressions, edit `qodana.yaml`, or
  close the TODO on inference.

- [ ] **Step 2: Confirm the settled source shape**

  Run:

  ```bash
  rg -n 'func wrapAPP1|return wrapAPP1|Exif\\x00\\x00' internal/uitest/uitest.go
  go test -timeout 2m -race -count=1 ./internal/uitest -run '^TestWrapAPP1_SplicesExifAfterSOI$'
  ```

  Expected: one helper, two callers, one framing literal, and a passing direct
  helper test.

- [ ] **Step 3: Move exactly this TODO to Done → Internal**

  Delete the complete section beginning:

  ```markdown
  ### Qodana: uitest.go wraps an APP1 segment twice
  ```

  through its final `4 DuplicatedCode hits.` paragraph, without changing the
  next TODO.

  Add this exact bullet under `## Done` → `### What's Changed` →
  `#### Internal`:

  ```markdown
  - Test JPEG APP1 fixtures: `wrapAPP1` in `internal/uitest/uitest.go` is the
    one Exif identifier, APP1 length, and post-SOI splice path shared by
    `CaptureDateJPEG` and `GPSJPEG`; a byte-level test locks the segment
    framing.
  ```

  Do not edit any other backlog item.

- [ ] **Step 4: Run focused and package verification**

  Run:

  ```bash
  go test -timeout 2m -race -count=1 ./internal/uitest -run '^TestWrapAPP1_SplicesExifAfterSOI$'
  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  go test -timeout 2m -race -count=1 ./internal/imaging -run '^TestExport_JPEGSourceKeepsMetadataOnJPEGDest$'
  go test -timeout 5m -race -count=1 ./internal/uitest ./internal/filesort ./internal/imaging
  ```

  Expected: PASS, with no new warning class.

- [ ] **Step 5: Self-review the documentation-only task diff**

  Run:

  ```bash
  git diff --check
  git status --short
  git diff -- todos.md
  rg -n 'Qodana: uitest.go wraps an APP1 segment twice|Test JPEG APP1 fixtures' todos.md
  ```

  Expected: the old heading is absent, the Done bullet appears exactly once,
  adjacent TODO entries are byte-for-byte unchanged, and no prohibited file
  changed during Task 2.

- [ ] **Step 6: Parent and independent review gate**

  The parent independently reruns `get_file_problems`, reads the complete
  `todos.md` diff, confirms the Done entry names the actual helper and callers,
  and verifies Task 1 source/tests did not drift. Dispatch a fresh
  `gpt-5.6-terra`/`medium` reviewer with the task brief, implementer report,
  path-scoped diff, and independent inspection result.

  Return any spec gap or Critical/Important finding to the Task 2 implementer
  and require a scoped re-review before acceptance.

- [ ] **Step 7: Suggested commit (agents and parent do not commit)**

  ```text
  close the uitest APP1 Qodana todo
  ```

---

### Task 3: Reuse the Shared Truncated-PNG Fixture

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `internal/imaging/loader_test.go`
- Inspect only: `internal/uitest/uitest.go`
- Do not modify: `internal/uitest/uitest_test.go`, `todos.md`,
  `needs_refactoring.md`, `ARCHITECTURE.md`, or production code

**Interfaces:**

- Consumes: exported `uitest.TruncatedPNGHeader(t, width, height)`.
- Produces: the same decompression-bomb fixture bytes for `TestLoadImage`,
  without the local `truncatedPNGHeader` duplicate.

- [ ] **Step 1: Reconfirm the focused baseline and residual finding**

  Run the `TestLoadImage` decompression-bomb subtest and call GoLand
  `get_file_problems` for `internal/uitest/uitest.go`. Record the complete
  inspection response; before this task it contains the 30-line finding
  anchored at `TruncatedPNGHeader` plus the separate TIFF finding.

- [ ] **Step 2: Replace the local helper with the shared fixture**

  In `internal/imaging/loader_test.go`:

  1. import `github.com/frathe/picfetch/internal/uitest` in the project-local
     import group;
  2. replace the sole `truncatedPNGHeader(t, 60000, 60000)` call with
     `uitest.TruncatedPNGHeader(t, 60000, 60000)`;
  3. delete the complete local `truncatedPNGHeader` function and its comment;
  4. remove now-unused `encoding/binary` and `hash/crc32` imports;
  5. retain `bytes`, which has unrelated callers elsewhere in the file.

  Do not move the test, change its assertions, or alter production loading.

- [ ] **Step 3: Format and verify behavior**

  Run:

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/imaging/loader_test.go
  go test -timeout 2m -race -count=1 ./internal/imaging -run 'TestLoadImage/rejects_an_absurd_header-declared_size_without_a_full_decode'
  go test -timeout 5m -race -count=1 ./internal/imaging
  ```

  Expected: PASS; only the documented duplicate `-lobjc` linker warning may
  appear.

- [ ] **Step 4: Verify the duplicate cluster disappeared**

  Call GoLand `get_file_problems` again for `internal/uitest/uitest.go` with
  `errorsOnly: false` and timeout 120 seconds. Expected: no 30-line finding at
  `TruncatedPNGHeader`; the separate TIFF construction finding may remain for
  Task 4. Record the complete structured response.

  Run:

  ```bash
  git diff --check
  git diff -- internal/imaging/loader_test.go
  rg -n 'truncatedPNGHeader|uitest.TruncatedPNGHeader|encoding/binary|hash/crc32' internal/imaging/loader_test.go
  ```

  Expected: one shared-fixture call, no local helper, and no now-unused
  imports.

- [ ] **Step 5: Parent and independent review gate**

  The parent checks the one-file diff and reruns the focused test. Dispatch a
  fresh `gpt-5.6-terra`/`medium` reviewer. Route any spec gap or
  Critical/Important finding back to the Task 3 implementer and require a
  scoped re-review.

---

### Task 4: Share the Little-Endian TIFF Byte Writer

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `internal/uitest/uitest.go`
- Modify: `internal/uitest/uitest_test.go`
- Inspect only: semantic consumers in `internal/filesort` and
  `internal/imaging`
- Do not modify: `internal/imaging/loader_test.go`, `todos.md`,
  `needs_refactoring.md`, `ARCHITECTURE.md`, or production code

**Interfaces:**

- Produces one private `littleEndianTIFF` byte writer that owns only the
  standard eight-byte TIFF prelude and little-endian integer emission.
- `CaptureDateJPEG`, `EncodeRAWPreview`, and `GPSJPEG` retain their explicit
  IFD/tag/offset/value-area layouts and consume the writer.

- [ ] **Step 1: Add the failing byte-contract test**

  Extend `internal/uitest/uitest_test.go` with
  `TestLittleEndianTIFF_WritesHeaderAndIntegers`. Construct the new writer,
  append `0x1234` through `u16`, append `0x89ABCDEF` through `u32`, and compare
  its complete bytes against:

  ```go
  []byte{
      'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00,
      0x34, 0x12, 0xEF, 0xCD, 0xAB, 0x89,
  }
  ```

  Run only the new test and record RED as `undefined: newLittleEndianTIFF`.
  A different failure must be corrected before helper code is added.

- [ ] **Step 2: Add the narrow byte writer**

  Add near `wrapAPP1` in `internal/uitest/uitest.go`:

  ```go
  const tiffHeaderSize = 8

  type littleEndianTIFF struct {
      bytes.Buffer
  }

  func newLittleEndianTIFF() *littleEndianTIFF {
      buf := new(littleEndianTIFF)
      _, _ = buf.WriteString("II")
      buf.u16(0x002A)
      buf.u32(tiffHeaderSize)

      return buf
  }

  func (b *littleEndianTIFF) u16(v uint16) {
      var data [2]byte
      binary.LittleEndian.PutUint16(data[:], v)
      _, _ = b.Write(data[:])
  }

  func (b *littleEndianTIFF) u32(v uint32) {
      var data [4]byte
      binary.LittleEndian.PutUint32(data[:], v)
      _, _ = b.Write(data[:])
  }
  ```

  The type embeds `bytes.Buffer` only to preserve explicit `Write`,
  `WriteString`, and `Bytes` calls in fixture code. Do not add a generic IFD
  entry type, callbacks, exported API, errors, validation, or configuration.

- [ ] **Step 3: Migrate exactly three fixture writers**

  In `CaptureDateJPEG`, `EncodeRAWPreview`, and `GPSJPEG`:

  - remove each local `binary.LittleEndian`, `u16`, and `u32` closure;
  - replace each standard `II`/magic/IFD0-offset prelude with
    `newLittleEndianTIFF()`;
  - replace integer-byte `Write(u16(...))` / `Write(u32(...))` calls with
    `.u16(...)` / `.u32(...)`;
  - use shared `tiffHeaderSize` in offset calculations;
  - preserve every tag, type, count, offset, comment, hemisphere rule,
    `valueArea`, embedded JPEG, and raw payload byte-for-byte.

  Do not change `dmsRationals`; it returns a payload rather than writing the
  TIFF structure.

- [ ] **Step 4: Format and verify the helper and all three fixture families**

  Run:

  ```bash
  go tool goimports -local github.com/frathe/picfetch -w internal/uitest/uitest.go internal/uitest/uitest_test.go
  go test -timeout 2m -race -count=1 ./internal/uitest -run '^TestLittleEndianTIFF_WritesHeaderAndIntegers$'
  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  go test -timeout 3m -race -count=1 ./internal/imaging -run '^(TestLoadImage_RAWPreview|TestReadMetadata_TIFFContainer|TestCaptureDate_RAWTIFF|TestExport_JPEGSourceKeepsMetadataOnJPEGDest)$'
  go test -timeout 5m -race -count=1 ./internal/uitest ./internal/filesort ./internal/imaging
  ```

  Expected: PASS, with no new warning class.

- [ ] **Step 5: Require a clean duplication inspection**

  Call GoLand `get_file_problems` for `internal/uitest/uitest.go` with
  `errorsOnly: false` and timeout 120 seconds. Acceptance requires no timeout
  and no `Duplicated code fragment` / `DuplicatedCode` result. If another
  duplicate appears, stop without suppression and report the exact fragment.

  Run:

  ```bash
  git diff --check
  git diff -- internal/uitest/uitest.go internal/uitest/uitest_test.go
  rg -n 'type littleEndianTIFF|newLittleEndianTIFF|\.u16\(|\.u32\(' internal/uitest/uitest.go internal/uitest/uitest_test.go
  ```

- [ ] **Step 6: Parent and independent review gate**

  The parent verifies the complete staged and unstaged diff, exact byte test,
  and semantic test evidence, then independently reruns the inspection.
  Dispatch a fresh `gpt-5.6-terra`/`medium` reviewer. Route real findings to
  the Task 4 implementer and require a scoped re-review.

---

### Task 5: Reinspect and Close the Expanded Backlog Item

**Subagent:** `gpt-5.6-luna` with `reasoning_effort: medium`

**Files:**

- Modify: `todos.md`
- Inspect only: `internal/uitest/uitest.go`,
  `internal/uitest/uitest_test.go`, `internal/imaging/loader_test.go`
- Do not modify: implementation/test files, `needs_refactoring.md`,
  `ARCHITECTURE.md`, `qodana.yaml`, or unrelated TODO entries

- [ ] **Step 1: Require clean, non-timeout IDE inspections**

  Call GoLand `get_file_problems` with `errorsOnly: false` and timeout 120
  seconds for both `internal/uitest/uitest.go` and
  `internal/imaging/loader_test.go`. Record both complete structured
  responses. Acceptance requires `internal/uitest/uitest.go` to have no
  `Duplicated code fragment` / `DuplicatedCode` result. For
  `internal/imaging/loader_test.go`, the scoped acceptance rule is that the
  former 30-line `truncatedPNGHeader` fragment is absent; its seven
  pre-existing 12/13-line `TestLoadImage` assertion fragments are unrelated
  to the two residual clusters the user authorized and do not block this
  task. Also run:

  ```bash
  rg -n 'func truncatedPNGHeader|uitest.TruncatedPNGHeader' internal/imaging/loader_test.go
  ```

  Acceptance requires no local helper definition and exactly one shared
  fixture call. If the uitest finding or the former 30-line PNG-construction
  finding remains, stop without editing `todos.md`.

- [ ] **Step 2: Move the original TODO to Done → Internal**

  Delete only the complete `### Qodana: uitest.go wraps an APP1 segment
  twice` section. Add this exact bullet under `## Done` → `### What's
  Changed` → `#### Internal`:

  ```markdown
  - Test image fixture duplication: `wrapAPP1` now owns Exif APP1 framing,
    `littleEndianTIFF` shares TIFF header and integer emission across the
    capture-date, RAW-preview, and GPS fixtures, and imaging's oversized-PNG
    test reuses `uitest.TruncatedPNGHeader`; byte-level tests lock both helper
    formats.
  ```

  Do not edit any other backlog or refactoring entry.

- [ ] **Step 3: Verify source shape and behavior**

  Run:

  ```bash
  go test -timeout 2m -race -count=1 ./internal/uitest
  go test -timeout 2m -race -count=1 ./internal/filesort -run '^TestOrderedFiles_SortsByCaptureDate$'
  go test -timeout 5m -race -count=1 ./internal/imaging
  git diff --check
  rg -n 'Qodana: uitest.go wraps an APP1 segment twice|Test image fixture duplication' todos.md
  ```

- [ ] **Step 4: Parent and independent review gate**

  The parent independently repeats both IDE inspections and reviews the exact
  `todos.md` diff. Dispatch a fresh `gpt-5.6-terra`/`medium` reviewer with the
  inspection evidence and all expanded-task paths. Route any real finding
  through the normal fix and scoped re-review loop.

---

## Parent Controller Protocol After Every Task

1. Record the current `HEAD` and scoped `git status --short` before dispatch.
2. Generate the task brief into the plan-specific SDD workspace and dispatch
   only one implementation agent with `fork_turns: "none"`, explicit model,
   explicit effort, no-subagent instructions, and a report-file contract.
3. When it returns, read the report, `git status --short`, `git diff --check`,
   and the complete diff for every changed source/test/document file.
4. Reject paths outside the task allowlist. Preserve `.aiignore`, the index,
   and all unrelated work.
5. Verify the task's commands and evidence. The existing duplicate `-lobjc`
   linker warning is baseline; any new warning is a finding.
6. Package the exact task diff for a fresh reviewer. Require both spec and
   quality verdicts.
7. Route real findings back to the same implementer for rounds 1-3; use a
   fresh, stronger agent only if the normal fix-loop escalation requires it.
   Every fix gets covering test output and a scoped re-review.
8. Mark the task complete only after parent review and independent review are
   clean. Continue directly to the next task unless inspection failure or a
   new design choice requires the user's answer.
9. Do not run `git commit`; retain the suggested per-task messages for final
   handoff.

## Final Whole-Change Review

After Tasks 1 and 3–5 are accepted (Task 2 is the recorded blocked gate):

1. Generate one review package covering the complete change since execution
   began: `internal/uitest/uitest.go`, `internal/uitest/uitest_test.go`,
   `internal/imaging/loader_test.go`, and `todos.md`, plus both inspection
   responses, the scoped loader-test ruling, and any deferred findings.
2. Dispatch `gpt-5.6-sol` with `reasoning_effort: high` using
   `superpowers:requesting-code-review`'s final reviewer prompt.
3. If the reviewer reports findings, dispatch one fix agent with the complete
   list, run the covering tests, and perform one scoped re-review.
4. Verify the final diff still contains only the approved APP1 helper and
   test, the little-endian TIFF writer and test, the three TIFF caller
   migrations, shared truncated-PNG use, and accurate TODO movement.

## Final Repository Verification

The parent runs from the repository root:

```bash
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

Then call GoLand `get_file_problems` one final time for
`internal/uitest/uitest.go` and `internal/imaging/loader_test.go` with
`errorsOnly: false`, timeout 120 seconds. Require the uitest file to be clean;
for loader_test, verify the former 30-line truncated-PNG fragment is absent
while recording its unrelated pre-existing `TestLoadImage` assertion
duplicates. Also verify:

```bash
git diff --check
git status --short
rg -n 'func wrapAPP1|return wrapAPP1|Exif\\x00\\x00' internal/uitest/uitest.go
rg -n 'func truncatedPNGHeader|uitest.TruncatedPNGHeader' internal/imaging/loader_test.go
rg -n 'Qodana: uitest.go wraps an APP1 segment twice|Test image fixture duplication' todos.md
```

Acceptance requires:

- all build/test/format/vet commands pass;
- GoLand does not time out, reports no `DuplicatedCode` finding in
  `internal/uitest/uitest.go`, and no longer reports the former 30-line
  truncated-PNG construction fragment;
- the direct test proves a two-byte APP1 length greater than `0x00FF`;
- the direct TIFF test proves the header, byte order, and integer widths;
- capture-date, RAW-preview, GPS, and decompression-bomb consumer tests pass;
- `todos.md` describes all three deduplication changes actually present;
- no unrelated file, failed golden image, generated binary, or dependency
  change was introduced.

Qodana CI itself still needs the repository's `QODANA_TOKEN`; the local
GoLand inspection is the immediate acceptance gate, and the next
authenticated CI run is the end-to-end confirmation.

## Out of Scope

- Refactoring TIFF IFD entry models, tag layouts, offset rules, value areas,
  or GPS hemisphere encoding beyond replacing repeated integer/header writes.
- Sharing TIFF helpers with production JPEG parsing or exposing a generic
  TIFF/IFD builder.
- Moving the decompression-bomb test or changing production image loading;
  only its duplicate fixture construction is shared.
- Generalizing APP1 to arbitrary APPn markers or supporting several inserted
  segments.
- Adding malformed-JPEG, nil-input, or 64-KiB payload validation to a helper
  whose existing callers always provide valid, small fixtures.
- Addressing the next TODO (Go doc comments), any other Qodana finding, or
  Qodana configuration.
- Updating `ARCHITECTURE.md`, translations, screenshots, dependencies, or
  release metadata.
