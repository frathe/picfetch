# JPEG header-segment walker — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** after every task the parent agent reviews the
> subagent's diff line by line, fixes it up itself, runs that task's verification
> commands, and then **stops** and hands Florian a suggested commit message. Do not
> dispatch Task N+1 until Florian has confirmed the commit landed. Do not run
> `git commit` (`AGENTS.md`). Dispatch **one** implementer at a time — these tasks
> share `internal/imaging` and will conflict if run in parallel.

**Goal:** One JPEG header-segment walker replaces the four copy-pasted SOS-stopping
marker loops so a bounds-check or no-payload-marker fix cannot silently leave the
other three wrong.

**Architecture:** Add unexported `walkJPEGSegments` next to `walkIFD`: a callback
walker that understands SOI, skips no-payload markers, stops at SOS or a malformed
length, and yields `(marker, payload)` for every payload-bearing segment. The four
callers keep their per-segment body only. `stripJPEGSegments` and `jpegLength` stay
as they are — they are not the same state machine.

**Tech Stack:** Go 1.26, package `internal/imaging`, existing `go test` / `-race`.
No new dependencies.

**Spec:** `todos.md` § "Qodana: the JPEG segment walk is copy-pasted four times".

## Global Constraints

- Do not run `git commit`. End each task with a suggested commit message for Florian.
- Do not touch the currently dirty tree (`internal/clipboard`, `filepicker`, `trash`,
  `wallpaper`, `ui/favorites`, `ui/help`, `ui/widgets`, `ui/windowmenu_notdarwin.go`,
  `update/apply_unix.go`, or the already-edited parts of `todos.md` that are unrelated
  to this item). Allowed files are listed per task.
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Unexported helpers only. Do not export the walker.
- Name and shape match `walkIFD`: `walkJPEGSegments`, not a new iteration style
  (`iter.Seq`, channels, an exported `JPEGSegment` type).
- Walker semantics are a literal extract of the four copies, **not** of
  `jpegLength` / `stripJPEGSegments`. In particular: do **not** skip fill `0xFF`
  bytes (that is `jpegLength`'s inner loop); a stray `0xFF 0xFF` is treated as a
  payload marker and typically fails the length bounds check.
- Returning `false` from the callback **stops the walk**. Returning `true`
  continues. This is the only difference from `walkIFD`'s `fn` (which has no
  early-exit).
- Comments move with the code they describe. The four-copy duplication comments
  in `jpegexif.go` are deleted once the walker exists; the walker's own comment
  is the new home of the marker rules.
- Tests: no `time.Sleep`; no new mutable package-level seams. Prefer table-driven
  tests in the `imaging` package (same style as `exif_test.go` / `jpegexif_test.go`).
- Format with `goimports -local github.com/frathe/picfetch`. `make fmt` is fine.
- `ARCHITECTURE.md` is updated in Task 4 of this plan (not a later unrelated
  PR). Task 1 may land `jpegseg.go` without the row; Task 4 must add it before
  the work is considered done. `AGENTS.md`'s "same change" rule is satisfied by
  this series, not by stuffing the row into the walker commit.
- Verification is `go test -timeout 5m -race ./internal/imaging/` unless a task
  names a narrower command. Do not run the 20-minute full-repo suite unless the
  parent asks.

## Open points (defaults used below)

Answer these before Task 1 if you disagree; otherwise the implementers follow the
defaults.

1. **Callback second argument** — **default: `payload []byte`** (the bytes after
   the 2-byte length, `data[segStart:segEnd]`), plus unexported `jpegSegmentBytes`
   to rebuild the on-disk form for `jpegMetadataSegments`. Alternative: pass the
   full on-disk `data[pos:segEnd]` and let callers use `seg[4:]`.
2. **Function name** — **default: `walkJPEGSegments`**, matching `walkIFD`. The
   todo's `forEachJPEGSegment` is rejected as a one-off name in a package that
   already has `walkIFD`.
3. **File** — **default: `internal/imaging/jpegseg.go` + `jpegseg_test.go`**,
   matching `exififd.go`. Do not drop the walker into `jpegexif.go` (save/strip
   helper) or `exif.go` (tag parsing).
4. **`stripJPEGSegments` / `jpegLength`** — **default: out of scope.** Confirmed
   below under "Not in this plan".

## File map

| File | Role after this plan |
|------|----------------------|
| `internal/imaging/jpegseg.go` | **Create.** `walkJPEGSegments` + `jpegSegmentBytes`. |
| `internal/imaging/jpegseg_test.go` | **Create.** Walker unit tests (TDD for Task 1). |
| `internal/imaging/exif.go` | **Modify.** `jpegEXIFOrientation`, `jpegMetadata` become callback bodies. |
| `internal/imaging/jpegexif.go` | **Modify.** `jpegMetadataSegments`, `jpegHasRemovableMetadata` become callback bodies. Delete the "duplicated rather than shared" comment. |
| `internal/imaging/exif_test.go` | **Unchanged.** Characterization tests for orientation / metadata. |
| `internal/imaging/jpegexif_test.go` | **Unchanged.** Characterization tests for collect / has-removable. |
| `ARCHITECTURE.md` | **Modify.** Add a `jpegseg.go` row under `internal/imaging`. |
| `todos.md` | **Modify.** Move this Qodana item from TODO to Done → Internal. |

## Subagent assignment

| Task | Subagent | Model | Why |
|------|----------|-------|-----|
| 1 — walker + tests | `go-expert` | `cursor-grok-4.6-xhigh` | Correctness-critical extract; must match `walkIFD` style and the four copies' exact marker rules. |
| 2 — `exif.go` callers | `go-expert` | `composer-2.5-fast` | Mechanical rewrite of two functions against a locked signature. |
| 3 — `jpegexif.go` callers | `go-expert` | `composer-2.5-fast` | Same, plus deleting the duplication comment and using `jpegSegmentBytes`. |
| 4 — docs + backlog | `generalPurpose` | `composer-2.5-fast` | `ARCHITECTURE.md` + `todos.md` only. |

No task needs `claude-opus-5-thinking-high`: the work splits cleanly and Task 1's
code is fully specified. Parent review after each task uses this session's model.

Do **not** dispatch `best-of-n-runner` or parallel implementers.

---

### Task 1: `walkJPEGSegments` + `jpegSegmentBytes`

**Subagent:** `go-expert` · **Model:** `cursor-grok-4.6-xhigh`

**Files:**
- Create: `internal/imaging/jpegseg.go`
- Create: `internal/imaging/jpegseg_test.go`
- Test: `go test -timeout 2m -race -count=1 ./internal/imaging/ -run 'TestWalkJPEGSegments|TestJPEGSegmentBytes'`

**Interfaces:**
- Consumes: nothing new. Same-package helpers `wrapAsAPP1` (in `exif_test.go`) may
  be used from `jpegseg_test.go`.
- Produces:
  - `func walkJPEGSegments(data []byte, fn func(marker byte, payload []byte) bool)`
  - `func jpegSegmentBytes(marker byte, payload []byte) []byte`

- [ ] **Step 1: Write the failing tests**

Create `internal/imaging/jpegseg_test.go` with exactly these tests. Do not
implement `walkJPEGSegments` yet.

```go
package imaging

import (
	"bytes"
	"testing"
)

type jpegVisit struct {
	marker  byte
	payload []byte
}

func collectJPEGSegments(data []byte) []jpegVisit {
	var got []jpegVisit
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		p := make([]byte, len(payload))
		copy(p, payload)
		got = append(got, jpegVisit{marker: marker, payload: p})
		return true
	})
	return got
}

func jpegWith(segments ...[]byte) []byte {
	data := []byte{0xFF, 0xD8}
	for _, s := range segments {
		data = append(data, s...)
	}
	data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0)
	return data
}

func TestWalkJPEGSegments(t *testing.T) {
	com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
	comPayload := []byte{'h', 'i', 0x00}

	t.Run("non-JPEG is a no-op", func(t *testing.T) {
		if got := collectJPEGSegments([]byte("\x89PNG")); len(got) != 0 {
			t.Fatalf("visits = %d, want 0", len(got))
		}
		if got := collectJPEGSegments(nil); len(got) != 0 {
			t.Fatalf("nil visits = %d, want 0", len(got))
		}
	})

	t.Run("SOI-only truncated file is a no-op", func(t *testing.T) {
		if got := collectJPEGSegments([]byte{0xFF, 0xD8}); len(got) != 0 {
			t.Fatalf("visits = %d, want 0", len(got))
		}
	})

	t.Run("one COM then SOS", func(t *testing.T) {
		got := collectJPEGSegments(jpegWith(com))
		if len(got) != 1 || got[0].marker != 0xFE || !bytes.Equal(got[0].payload, comPayload) {
			t.Fatalf("got %+v, want one COM payload %q", got, comPayload)
		}
	})

	t.Run("skips no-payload RST and TEM; still visits neighbours", func(t *testing.T) {
		app1 := wrapAsAPP1([]byte("Exif\x00\x00xxxx"))
		rst := []byte{0xFF, 0xD0}
		tem := []byte{0xFF, 0x01}
		got := collectJPEGSegments(jpegWith(app1, rst, tem, com))
		if len(got) != 2 || got[0].marker != 0xE1 || got[1].marker != 0xFE {
			t.Fatalf("got markers %v, want E1 then FE", got)
		}
		if !bytes.Equal(got[1].payload, comPayload) {
			t.Fatalf("COM payload = %q, want %q", got[1].payload, comPayload)
		}
	})

	t.Run("does not visit segments after SOS", func(t *testing.T) {
		after := []byte{0xFF, 0xFE, 0x00, 0x05, 'n', 'o', 0x00}
		data := jpegWith(com)
		data = append(data, after...)
		got := collectJPEGSegments(data)
		if len(got) != 1 || got[0].marker != 0xFE || !bytes.Equal(got[0].payload, comPayload) {
			t.Fatalf("got %+v, want only the pre-SOS COM", got)
		}
	})

	t.Run("malformed length stops the walk", func(t *testing.T) {
		short := []byte{0xFF, 0xE1, 0x00, 0x01} // segLen 1 < 2
		got := collectJPEGSegments(jpegWith(com, short, []byte{0xFF, 0xFE, 0x00, 0x05, 'z', 'z', 0x00}))
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the bad length", got)
		}
	})

	t.Run("truncated payload stops the walk", func(t *testing.T) {
		trunc := []byte{0xFF, 0xE1, 0x00, 0x10, 1, 2, 3} // claims 14 payload bytes
		got := collectJPEGSegments(append([]byte{0xFF, 0xD8}, append(com, trunc...)...))
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the truncated APP1", got)
		}
	})

	t.Run("non-0xFF at a marker boundary stops the walk", func(t *testing.T) {
		data := append([]byte{0xFF, 0xD8}, com...)
		data = append(data, 0x00, 0x00)
		data = append(data, com...)
		data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0)
		got := collectJPEGSegments(data)
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the non-FF byte", got)
		}
	})

	t.Run("false from fn stops the walk", func(t *testing.T) {
		com2 := []byte{0xFF, 0xFE, 0x00, 0x05, 'b', 'y', 0x00}
		var n int
		walkJPEGSegments(jpegWith(com, com2), func(marker byte, payload []byte) bool {
			n++
			return false
		})
		if n != 1 {
			t.Fatalf("callbacks = %d, want 1", n)
		}
	})
}

func TestJPEGSegmentBytes(t *testing.T) {
	payload := []byte("Exif\x00\x00hi")
	got := jpegSegmentBytes(0xE1, payload)
	want := wrapAsAPP1(payload)
	if !bytes.Equal(got, want) {
		t.Fatalf("jpegSegmentBytes = %x, want %x", got, want)
	}
	// Mutating the result must not change payload.
	got[4] = 'X'
	if payload[0] != 'E' {
		t.Fatal("jpegSegmentBytes must copy the payload")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test -timeout 2m -count=1 ./internal/imaging/ -run 'TestWalkJPEGSegments|TestJPEGSegmentBytes'
```

Expected: FAIL with `undefined: walkJPEGSegments` (and `undefined: jpegSegmentBytes`).
If they pass, stop — the functions already exist and this task is done or the
tree is not the one this plan describes.

- [ ] **Step 3: Write the implementation**

Create `internal/imaging/jpegseg.go`. Copy this body; do not "improve" the
no-payload set, the `pos+4` loop condition, or the fill-byte behaviour.

```go
package imaging

// walkJPEGSegments calls fn for each payload-bearing marker between SOI and
// SOS. No-payload markers (TEM 0x01, RST0–RST7 0xD0–0xD7, SOI 0xD8, EOI 0xD9)
// are skipped without a callback. SOS (0xDA) and any malformed structure
// (a non-0xFF where a marker is required, a length field < 2, or a segment
// that would run past len(data)) stop the walk without a callback.
//
// payload is the bytes after the 2-byte length (data[pos+4:pos+2+segLen]),
// not including the 0xFF marker. It aliases data — callers that retain it
// must copy. Returning false from fn stops the walk; true continues.
// Non-JPEG data (missing FF D8) is a no-op.
//
// This is a literal extract of the four header walks in jpegEXIFOrientation,
// jpegMetadata, jpegMetadataSegments, and jpegHasRemovableMetadata. It does
// not skip fill 0xFF bytes and does not walk the entropy-coded scan — those
// are jpegLength / stripJPEGSegments, which stay separate.
func walkJPEGSegments(data []byte, fn func(marker byte, payload []byte) bool) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return
	}

	pos := 2

	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			break
		}

		marker := data[pos+1]

		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			pos += 2
			continue
		}

		if marker == 0xDA {
			break
		}

		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			break
		}

		segStart := pos + 4
		segEnd := pos + 2 + segLen
		if !fn(marker, data[segStart:segEnd]) {
			return
		}
		pos = segEnd
	}
}

// jpegSegmentBytes returns a standalone on-disk COM/APPn segment: 0xFF,
// marker, 2-byte big-endian length (len(payload)+2), then payload. The
// result is a copy, so the caller may mutate it.
func jpegSegmentBytes(marker byte, payload []byte) []byte {
	n := len(payload) + 2
	seg := make([]byte, 4+len(payload))
	seg[0] = 0xFF
	seg[1] = marker
	seg[2] = byte(n >> 8)
	seg[3] = byte(n)
	copy(seg[4:], payload)
	return seg
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
goimports -local github.com/frathe/picfetch -w internal/imaging/jpegseg.go internal/imaging/jpegseg_test.go
go test -timeout 2m -race -count=1 ./internal/imaging/ -run 'TestWalkJPEGSegments|TestJPEGSegmentBytes'
```

Expected: PASS, both tests.

Then run the whole imaging package once to prove the new file did not break
existing tests (callers are still the old loops):

```bash
go test -timeout 5m -race -count=1 ./internal/imaging/
```

Expected: PASS.

- [ ] **Step 5: Suggested commit (do not run git commit)**

```
extract JPEG header-segment walker from the four duplicated loops

One walkJPEGSegments owns the SOI / no-payload / SOS / bounds-check state
machine so a fix in one copy cannot leave the other three wrong. Callers
are not switched yet.
```

**Parent review focus:** the no-payload set is identical to `exif.go:47`; the
loop still requires `pos+4`; fill `0xFF` is not skipped; `fn` false returns
(does not `break` to a later segment).

---

### Task 2: Rewrite `jpegEXIFOrientation` and `jpegMetadata`

**Subagent:** `go-expert` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `internal/imaging/exif.go` (`jpegEXIFOrientation`, `jpegMetadata` only)
- Test: `go test -timeout 5m -race -count=1 ./internal/imaging/ -run 'TestReadEXIFOrientation|TestReadMetadata|TestParseExifOrientation|TestWalkJPEGSegments'`

**Interfaces:**
- Consumes: `walkJPEGSegments(data []byte, fn func(marker byte, payload []byte) bool)` from Task 1.
- Produces: same exported/unexported behaviour as today. `readEXIFOrientation` and
  `ReadMetadata` keep their signatures.

- [ ] **Step 1: Replace `jpegEXIFOrientation`**

In `internal/imaging/exif.go`, replace the function body (keep the existing
doc comment that this was split out of `readEXIFOrientation` for TIFF-container
RAW). Delete the inline walk. The function becomes:

```go
func jpegEXIFOrientation(data []byte) int {
	found := 1
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xE1 {
			if o := parseExifOrientation(payload); o != 0 {
				found = o
				return false
			}
		}
		return true
	})
	return found
}
```

- [ ] **Step 2: Replace `jpegMetadata`**

Replace `jpegMetadata` the same way. Keep `ReadMetadata`'s doc comment. Delete
the duplicated SOI check and loop:

```go
func jpegMetadata(data []byte) Metadata {
	var found Metadata
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker != 0xE1 {
			return true
		}
		if len(payload) >= 8 && string(payload[:6]) == "Exif\x00\x00" {
			if m := parseExifMetadata(payload[6:]); !m.Empty() {
				found = m
				return false
			}
		}
		return true
	})
	return found
}
```

Do not change `parseExifOrientation`, `parseExifMetadata`, or `ReadMetadata`'s
dispatch (JPEG / TIFF / ISOBMFF / RAW preview).

- [ ] **Step 3: Run characterization tests**

```bash
goimports -local github.com/frathe/picfetch -w internal/imaging/exif.go
go test -timeout 5m -race -count=1 ./internal/imaging/
```

Expected: PASS. In particular `TestReadEXIFOrientation` still covers "Exif
after another APP marker", truncated SOI, and non-JPEG; `TestReadMetadata*`
still finds GPS / full tag sets on JPEGs.

- [ ] **Step 4: Suggested commit (do not run git commit)**

```
switch JPEG EXIF orientation and metadata reads onto walkJPEGSegments

jpegEXIFOrientation and jpegMetadata keep only their per-APP1 bodies; the
shared marker skip lives in one place.
```

**Parent review focus:** early-exit `return false` on the first usable APP1
(same as today's `return` from inside the loop). Empty / non-Exif APP1s still
continue. No behaviour change for TIFF/HEIC/RAW paths.

---

### Task 3: Rewrite `jpegMetadataSegments` and `jpegHasRemovableMetadata`

**Subagent:** `go-expert` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `internal/imaging/jpegexif.go` (`jpegMetadataSegments` and
  `jpegHasRemovableMetadata` only; do **not** edit `stripJPEGSegments`)
- Test: `go test -timeout 5m -race -count=1 ./internal/imaging/`

**Interfaces:**
- Consumes: `walkJPEGSegments`, `jpegSegmentBytes` from Task 1; existing
  `skipSegment`, `keepOnStrip`, `jpegLength`.
- Produces: same slice contents / boolean results as today.

- [ ] **Step 1: Replace `jpegMetadataSegments`**

Replace the function and its "duplicated rather than shared" comment with:

```go
// jpegMetadataSegments returns a copy of every COM and APPn segment between
// SOI and SOS, in file order, excluding APP0 (JFIF/JFXX) and APP2 segments
// whose payload starts with "MPF\x00". Each slice includes the 0xFF marker,
// the 2-byte length, and the payload. Later re-encoding can copy these
// segments verbatim into a freshly written JPEG to preserve metadata a
// bare image/jpeg.Encode call would otherwise drop. data that is not a
// JPEG yields nil.
func jpegMetadataSegments(data []byte) [][]byte {
	var segs [][]byte
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !skipSegment(marker, payload) {
				segs = append(segs, jpegSegmentBytes(marker, payload))
			}
		}
		return true
	})
	return segs
}
```

`skipSegment` is unchanged. The returned slices must remain copies (the
existing test mutates `got[2][4]` and asserts the source Exif is intact);
`jpegSegmentBytes` provides that copy.

- [ ] **Step 2: Replace `jpegHasRemovableMetadata`**

Keep the trailer check via `jpegLength` **before** the walk — a file whose
header is clean but that has bytes after the primary EOI is still removable.
Drop the now-redundant SOI check; `jpegLength` and `walkJPEGSegments` both
no-op on non-JPEG.

```go
func jpegHasRemovableMetadata(data []byte) bool {
	if n := jpegLength(data); n > 0 && n < len(data) {
		return true
	}
	found := false
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !keepOnStrip(marker, payload) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
```

Do not rewrite `stripJPEGSegments`, `injectJPEGMetadata`, `skipSegment`,
`keepOnStrip`, or `jpegLength`.

- [ ] **Step 3: Run characterization tests**

```bash
goimports -local github.com/frathe/picfetch -w internal/imaging/jpegexif.go
go test -timeout 5m -race -count=1 ./internal/imaging/
```

Expected: PASS. `TestJPEGMetadataSegments` still returns COM/XMP/Exif/ICC in
order and copies; `TestJPEGHasRemovableMetadata` still treats stdlib JFIF as
clean, Exif as removable, PNG as false, and post-EOI trailers as removable;
`TestStripJPEGSegments` still passes because this task does not touch it.

- [ ] **Step 4: Suggested commit (do not run git commit)**

```
switch JPEG metadata collect and removable-metadata probe onto the walker

jpegMetadataSegments and jpegHasRemovableMetadata now share the SOI-to-SOS
walk with the EXIF readers. stripJPEGSegments is unchanged: it copies or
errors rather than visiting.
```

**Parent review focus:** `jpegSegmentBytes` output is bit-identical to copying
`data[pos:segEnd]` (length field = `len(payload)+2`). Trailer detection still
runs before the walk. `stripJPEGSegments` diff is empty.

---

### Task 4: `ARCHITECTURE.md` and `todos.md`

**Subagent:** `generalPurpose` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `ARCHITECTURE.md` (the `internal/imaging` file table only)
- Modify: `todos.md` (move this Qodana item to Done → Internal)

**Interfaces:**
- Consumes: `jpegseg.go` exists and the four callers use it.
- Produces: docs that match the tree.

- [ ] **Step 1: Architecture row**

In `ARCHITECTURE.md`, in the `internal/imaging` file table, add this row
immediately above the `jpegexif.go` row (walkers sit next to the code that
uses them, `exififd.go` already follows that pattern):

```
| `jpegseg.go` | Unexported JPEG header-segment walker (`walkJPEGSegments`) used by `exif.go` and `jpegexif.go`. Stops at SOS; does not walk entropy-coded scans (`jpegLength` in `raw.go`) or copy/strip (`stripJPEGSegments` in `jpegexif.go`). |
```

Do not rewrite the `exif.go` or `jpegexif.go` rows beyond what is needed if
they currently claim to own the walk. Today they do not; leave those rows'
wording unless a sentence would become false (it should not).

- [ ] **Step 2: Backlog**

In `todos.md`:

1. Cut the whole `### Qodana: the JPEG segment walk is copy-pasted four times`
   section (the heading plus its paragraph) out of `## TODO`.
2. Under `## Done` → `#### Internal`, add:

```
- JPEG header-segment walk: `walkJPEGSegments` in `internal/imaging/jpegseg.go`
  is the one SOI-to-SOS marker loop; `jpegEXIFOrientation`, `jpegMetadata`,
  `jpegMetadataSegments`, and `jpegHasRemovableMetadata` keep only their
  per-segment bodies. `stripJPEGSegments` and `jpegLength` stay separate —
  they copy/error and walk entropy-coded scans, respectively.
```

Do not edit other TODO items. Do not revert unrelated dirty changes in
`todos.md`.

- [ ] **Step 3: Verify docs-only**

```bash
go test -timeout 5m -race -count=1 ./internal/imaging/
```

Expected: PASS (no code change in this task). Parent confirms the architecture
row is accurate.

- [ ] **Step 4: Suggested commit (do not run git commit)**

```
document the JPEG segment walker in ARCHITECTURE and close the todo
```

---

## Controller protocol (parent session)

1. Confirm Florian accepted the Open points defaults (or edit this plan first).
2. Dispatch Task N's implementer with **only that task's section** (via
   `task-brief`), the Global Constraints, and the produced signatures from
   earlier tasks. Do not paste the whole plan.
3. On DONE: read the diff. Fix anything that drifted from the specified code
   (wrong no-payload set, fill-byte "improvements", `stripJPEGSegments` edits,
   exported API, extra helpers). Re-run the task's test command.
4. Hand Florian the suggested commit message. **Stop.**
5. After the commit lands, dispatch Task N+1.
6. After Task 4: optional whole-branch review of `jpegseg.go` + the four
   callers, still no `git commit` from the agent.

## Not in this plan

- **`stripJPEGSegments`** (`jpegexif.go:143`). Same marker set, different
  control flow: it **copies** no-payload markers into the output, **returns
  `errNotJPEG`** on a non-`0xFF` or a bad length (the four copies `break`),
  and at SOS copies the entropy-coded scan through `jpegLength`. Forcing it
  onto `walkJPEGSegments` would drop RST/TEM in the header or swallow errors.
- **`jpegLength` / `scanToEOI`** (`raw.go:259`). Different machine: fill-`0xFF`
  skip, stuffed `0xFF 0x00` in the scan, stop at EOI. Used to find embedded
  JPEGs in RAW and to size the primary JPEG for trailer detection.
- Qodana items below this one in `todos.md` (settingswin numeric entries,
  orientation transforms, `wrapAPP1`, doc comments, mechanical fixes).
- Regenerating goldens, touching translations, or running Qodana CI.

## Risks and how each is contained

| Risk | Containment |
|------|-------------|
| Walker "fixed" to match `jpegLength` (fill bytes, scan walk) | Global constraint + Task 1 tests that SOS hides later segments; parent rejects fill-skip |
| `jpegMetadataSegments` returns aliases of `data` | `jpegSegmentBytes` copies; existing mutation test in `TestJPEGMetadataSegments` |
| Trailer-only files no longer look removable | `jpegHasRemovableMetadata` still consults `jpegLength` first; `TestJPEGHasRemovableMetadata` covers post-EOI |
| `stripJPEGSegments` accidentally rewritten | Task 3 file list + parent review that its diff is empty |
| Dirty-tree files get mixed into the change | Global constraint; parent `git diff --stat` before suggesting a commit |
