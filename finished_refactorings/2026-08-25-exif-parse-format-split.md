# EXIF Parse / IFD / Format File Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the full diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Dispatch **one implementer at a time** — Tasks 1 and 2 both cut functions out of the same file.
>
> **Start gate:** Do **not** dispatch any subagent until the human partner replies to start. Open points in “Locked decisions” must be confirmed or overridden first.

**Goal:** Split `internal/imaging/exif.go` (734 lines) into three same-package files — parsers, IFD machinery, display formatting — with no behavior change.

**Architecture:** Move-only. All symbols stay unexported (except the already-exported `Metadata` / `ReadMetadata` / `Empty`, which remain in `exif.go`). No new types, no walker unification, no package split. Callers (`loader.go`, `raw.go`, `exifwin`, save/export tests) keep compiling against the same names.

**Tech Stack:** Go (see `go.mod`). Package `imaging`. Existing tests in `internal/imaging/exif_test.go` are the spec. No new dependencies.

## Status check (2026-08-25) — this todo is old but not done

Checked against the tree before writing this plan.

| `todos.md` item | Still open? | This plan |
|-----------------|-------------|-----------|
| `internal/imaging/exif.go` (~687 lines when filed; **734 now**) holds two parsers plus IFD walking plus display formatting. “A parse/format file split is cosmetic.” | **Yes.** Still one file. No `exifformat.go` / `exififd.go`. | **This plan.** |
| Menu Window / Menu Actions (listed under `## Done`) | **Already implemented.** `windowmenu.go`, `actionmenu.go`, and their tests exist. | Out of scope. |
| `finishLoad` named steps | Done (2026-08-25). | Out of scope. |

If the user wanted a different TODO, stop; do not execute this plan.

## Approaches considered

1. **Two files: `exif.go` (parse) + `exifformat.go` (display).** Literal reading of the todo. Moves ~50 lines. `exif.go` stays ~680 lines — the size complaint is almost unchanged. Rejected as the *only* split.
2. **Three files: `exif.go` + `exififd.go` + `exifformat.go` (this plan).** Matches the todo’s own inventory (“two parsers plus IFD walking plus display formatting”). Both parsers stay together in `exif.go` (they share container dispatch and the `Metadata` type). Generic IFD walking moves out. Display formatters move out. `exif.go` lands around ~530 lines. Still cosmetic (no behavior), but the files become greppable units.
3. **Four files, also splitting orientation vs metadata parsers.** `orientation.go` already exists for *pixel* transforms (`ApplyOrientation`). A second `exiforient.go` next to it is a naming trap. The two parsers share JPEG APP1 / TIFF header knowledge; splitting them forces readers to bounce. Rejected.
4. **Unify the three JPEG segment walkers** (`jpegEXIFOrientation`, `jpegMetadata`, `jpegMetadataSegments` in `jpegexif.go`) as part of this split. `jpegexif.go` already documents the duplication as intentional (different stop conditions and return types). Unifying would be a behavior-risk refactor, not a file split. Rejected.
5. **Rewrite `parseExifOrientation` to use `walkIFD`.** Different contract: orientation-in-APP1 returns `0` (“keep looking / not Exif”) and only accepts inline SHORT type 3; `tiffIFD0Orientation` already uses `walkIFD` and defaults to `1`. Unifying is a behavior change. Rejected.

## Locked decisions (confirm or override before Task 1)

These are the plan defaults. Subagents must not reopen them.

1. **Three files, same package `imaging`:**
   - `exif.go` — `readEXIFOrientation` and friends, `Metadata`, `ReadMetadata`, JPEG/TIFF/ISOBMFF/RAW-preview dispatch, `parseExifMetadata`, GPS (`parseGPSIFD`, `degreesFromDMS`, `validCoordinates`).
   - `exififd.go` — `walkIFD`, `tagComponentSize`, `asciiValue`, `uintValue`, `rationalValue`, `rationalsValue`.
   - `exifformat.go` — `formatExposureTime`, `formatFocalLength`, `formatExifDate`, `parseExifDateTime`.
2. **Matching test files.** Move only tests whose *subject* moved. Builders (`buildExifSegment`, `buildFullExifTIFF`, `gpsJPEG`, …) stay in `exif_test.go`. `TestDegreesFromDMS` / `TestValidCoordinates` stay in `exif_test.go` (GPS conversion, not IFD).
3. **Do not touch** `jpegexif.go`, `orientation.go`, `raw.go` (`tiffOrder` stays there), `loader.go`, or any UI package.
4. **Do not unify walkers or rewrite `parseExifOrientation`.** Cut-and-keep comments. Do not “improve” while moving.
5. **No new tests whose only job is to prove a file exists.** This is a move. Existing tests are the spec. TDD of “write a failing test for a missing file” does not apply. Each task: run covering tests (green) → move with comments intact → run covering tests (still green).
6. **Do not commit.** Suggested commit message only, at the end of Task 3.
7. **Preserve `gofmt` / `goimports -local github.com/frathe/picfetch` grouping** on every new file (stdlib only here; no third-party imports in the new files).

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments to source. Open work stays in `todos.md`.
- Do not change behavior, signatures, or export surface. After the split, `go test -race ./internal/imaging/` must pass with the same assertions.
- Do not add goroutines, package-level test seams, or dependencies.
- Subagents must not start Task N+1 themselves. They stop after their task’s verification and report.
- `ARCHITECTURE.md` updates only in Task 3 (new file rows in the `internal/imaging` table). Tasks 1–2 must not edit it.
- `todos.md` updates only in Task 3.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

Implementers use `subagent_type: go-expert`. Task reviewers use `subagent_type: go-expert`. Do **not** use Opus for implementation — the work splits cleanly. Opus is reserved for the final whole-branch review (did anything besides files move?).

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `composer-2.5-fast` | Transcription: plan contains the complete new file. |
| Task 1 reviewer | `cursor-grok-4.5-high-fast` | Mid-tier floor; catch comment loss / import mistakes. |
| Task 2 implementer | `cursor-grok-4.5-high-fast` | Same move pattern, but must not scoop GPS helpers into the IFD file. |
| Task 2 reviewer | `cursor-grok-4.6-xhigh` | Boundary mistakes (moving `parseExifMetadata` / `parseGPSIFD`) would not fail tests. |
| Task 3 implementer | `composer-2.5-fast` | Docs only; locator wording is specified. |
| Task 3 reviewer | `cursor-grok-4.5-high-fast` | Check `ARCHITECTURE.md` / `todos.md` against the finished files. |
| Parent review / fix after each task | this session (do not dispatch) | User asked the parent to review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | Confirm zero behavior change across the three files vs HEAD. |

## File structure

- Create: `internal/imaging/exifformat.go`
- Create: `internal/imaging/exifformat_test.go`
- Create: `internal/imaging/exififd.go`
- Create: `internal/imaging/exififd_test.go`
- Modify: `internal/imaging/exif.go` — delete moved functions only
- Modify: `internal/imaging/exif_test.go` — delete moved tests only
- Modify: `ARCHITECTURE.md` — two new rows in the `internal/imaging` table (Task 3)
- Modify: `todos.md` — move the bullet to `## Done` (Task 3)
- Do not create: `exiforient.go`, `exifparse.go`, a new package
- Do not modify: `jpegexif.go`, `orientation.go`, `raw.go`, UI files, translations

## Target layout (after Task 2)

```
exif.go            readEXIFOrientation, jpegEXIFOrientation, tiffIFD0Orientation,
                   parseExifOrientation, Metadata, Empty, ReadMetadata,
                   jpegMetadata, isobmffMetadata, metadataFromISOBMFFExif,
                   parseExifMetadata, parseGPSIFD, degreesFromDMS, validCoordinates,
                   constants exifIFDPointer / gpsIFDPointer

exififd.go         walkIFD, tagComponentSize, asciiValue, uintValue,
                   rationalValue, rationalsValue

exifformat.go      formatExposureTime, formatFocalLength, formatExifDate,
                   parseExifDateTime
```

`parseExifDateTime` is a parser of the *display* date encoding, not an IFD parser. It lives with `formatExifDate` (they are inverses). Do not put it in `exififd.go`.

---

### Task 1: Extract display formatters

**Model:** `composer-2.5-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)
**subagent_type:** `go-expert` (both)

**Files:**
- Create: `internal/imaging/exifformat.go`
- Create: `internal/imaging/exifformat_test.go`
- Modify: `internal/imaging/exif.go` — delete the four functions at the bottom of the file (today lines 684–734)
- Modify: `internal/imaging/exif_test.go` — delete `TestFormatExposureTime`, `TestFormatFocalLength`, `TestFormatExifDate`, `TestParseExifDateTime` (today lines 729–796)
- Test: `go test -race -count=1 ./internal/imaging/`

**Interfaces:**
- Consumes: nothing from later tasks. The four functions already exist, unexported, in `exif.go`. Call sites in `metadataFromISOBMFFExif` and `parseExifMetadata` stay put and keep compiling because they are the same package.
- Produces: `exifformat.go` with exactly the four functions below. `exif.go` still contains `math` (`validCoordinates`) and `time` (`Metadata.DateTakenTime`) and `fmt` (FNumber/ISO sprintf). Do not drop those imports.

- [ ] **Step 1: Confirm the split is not already done; run covering tests**

If `internal/imaging/exifformat.go` or `internal/imaging/exififd.go` already exists, stop and report **BLOCKED** (todo already done).

Run:

```bash
go test -race -count=1 ./internal/imaging/
```

Expected: PASS. If anything fails before your edit, stop and report **BLOCKED** (pre-existing).

- [ ] **Step 2: Create `internal/imaging/exifformat.go`**

Create the file with **exactly** this content (comments copied from `exif.go`, not rewritten):

```go
package imaging

import (
	"fmt"
	"math"
	"time"
)

// formatExposureTime renders a shutter speed in seconds as Exif-style
// display text: "1/200 s" for anything faster than a second (the common
// case), or "2.5 s" for a full second or slower (long exposures).
func formatExposureTime(seconds float64) string {
	if seconds >= 1 {
		return fmt.Sprintf("%.1f s", seconds)
	}

	denominator := math.Round(1 / seconds)

	return fmt.Sprintf("1/%d s", int64(denominator))
}

// formatFocalLength renders a focal length in millimeters, dropping the
// decimal point for the common whole-number case.
func formatFocalLength(mm float64) string {
	if mm == math.Trunc(mm) {
		return fmt.Sprintf("%.0f mm", mm)
	}

	return fmt.Sprintf("%.1f mm", mm)
}

// formatExifDate reformats Exif's "YYYY:MM:DD HH:MM:SS" date/time encoding
// (colons instead of dashes in the date, so it doubles as a valid bare
// filename component on every OS) into the more readable
// "YYYY-MM-DD HH:MM:SS". Anything not matching that exact shape is passed
// through unchanged rather than discarded - still useful to show even if
// this reader doesn't recognize its layout.
func formatExifDate(raw string) string {
	if len(raw) == 19 && raw[4] == ':' && raw[7] == ':' {
		return raw[:4] + "-" + raw[5:7] + "-" + raw[8:]
	}

	return raw
}

// parseExifDateTime parses raw - the same "YYYY:MM:DD HH:MM:SS" Exif
// encoding formatExifDate reformats for display - into a time.Time. ok is
// false for anything not matching that exact layout, mirroring
// formatExifDate's own tolerant fallback (pass the raw string through
// unchanged rather than erroring). Interpreted in the local zone: Exif
// carries no timezone offset in the tags this reader looks at, so that's
// the best available guess, same as most photo software assumes.
func parseExifDateTime(raw string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006:01:02 15:04:05", raw, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
```

- [ ] **Step 3: Delete the four functions from `exif.go`**

Delete from the blank line after `rationalsValue`’s closing brace through end of file, including the four `format*` / `parseExifDateTime` comments and bodies. `exif.go` must now end at `rationalsValue` (Task 2 will move that function). Do not change any remaining function. Do not run `goimports` in a way that drops `fmt`, `math`, or `time` from `exif.go` — they are still used.

- [ ] **Step 4: Create `internal/imaging/exifformat_test.go` and delete the four tests from `exif_test.go`**

Create:

```go
package imaging

import (
	"testing"
	"time"
)

func TestFormatExposureTime(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{1.0 / 200, "1/200 s"},
		{1.0 / 4000, "1/4000 s"},
		{2.5, "2.5 s"},
		{1, "1.0 s"},
	}

	for _, c := range cases {
		if got := formatExposureTime(c.seconds); got != c.want {
			t.Errorf("formatExposureTime(%v) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

func TestFormatFocalLength(t *testing.T) {
	cases := []struct {
		mm   float64
		want string
	}{
		{50, "50 mm"},
		{18.5, "18.5 mm"},
	}

	for _, c := range cases {
		if got := formatFocalLength(c.mm); got != c.want {
			t.Errorf("formatFocalLength(%v) = %q, want %q", c.mm, got, c.want)
		}
	}
}

func TestFormatExifDate(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"2024:08:12 14:33:02", "2024-08-12 14:33:02"},
		{"garbage", "garbage"},
	}

	for _, c := range cases {
		if got := formatExifDate(c.raw); got != c.want {
			t.Errorf("formatExifDate(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestParseExifDateTime(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, ok := parseExifDateTime("2024:08:12 14:33:02")
		if !ok {
			t.Fatal("ok = false, want true")
		}
		want := time.Date(2024, 8, 12, 14, 33, 2, 0, time.Local)
		if !got.Equal(want) {
			t.Errorf("parseExifDateTime() = %v, want %v", got, want)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		if _, ok := parseExifDateTime("garbage"); ok {
			t.Error("ok = true, want false for a malformed date")
		}
	})
}
```

Delete those four test functions from `exif_test.go`. Leave `TestReadEXIFOrientation` (it follows them today) in `exif_test.go`. Leave unused imports out of `exif_test.go` only if `goimports` reports them unused after the delete; `time` may become unused in `exif_test.go` — drop it there if so. Do **not** drop `time` from `exif.go`.

- [ ] **Step 5: Format and re-run covering tests**

Run:

```bash
gofmt -w internal/imaging/exif.go internal/imaging/exifformat.go internal/imaging/exif_test.go internal/imaging/exifformat_test.go
go test -race -count=1 ./internal/imaging/
```

Expected: PASS, including `TestFormatExposureTime`, `TestFormatFocalLength`, `TestFormatExifDate`, `TestParseExifDateTime` (now in the new test file).

If `exif.go` has an unused import after the cut, that is a bug in Step 3 (you cut too much) or a false alarm: `fmt`, `math`, `time`, `strings`, `bytes`, `encoding/binary`, `heic`, `avif` must all still be used. Fix the cut, do not “clean up” a still-used import.

- [ ] **Step 6: Stop. Do not commit. Do not start Task 2.**

Report: files created/modified, test command + result, any unused-import fixes.

**Parent review checklist (this session, after the implementer):**
- `exifformat.go` comments match the old `exif.go` comments word-for-word.
- `parseExifDateTime` is in `exifformat.go`, not left behind.
- `exif.go` still has `rationalsValue` and everything above it.
- No edits outside the four files listed.

---

### Task 2: Extract IFD walking

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.6-xhigh` (task reviewer)
**subagent_type:** `go-expert` (both)

**Files:**
- Create: `internal/imaging/exififd.go`
- Create: `internal/imaging/exififd_test.go`
- Modify: `internal/imaging/exif.go` — delete `walkIFD` through `rationalsValue` (after Task 1, these are the last functions in the file)
- Modify: `internal/imaging/exif_test.go` — delete `TestRationalsValue` only
- Test: `go test -race -count=1 ./internal/imaging/`

**Interfaces:**
- Consumes: Task 1’s `exifformat.go` already exists. Do not move formatters again.
- Produces: `exififd.go` with exactly the six functions below. `exif.go` still contains `parseExifMetadata`, `parseGPSIFD`, `degreesFromDMS`, `validCoordinates`, `exifIFDPointer`, `gpsIFDPointer`. Those are parsers / GPS conversion, not the generic walker.

**Do not move into `exififd.go`:**
- `parseExifMetadata`
- `parseGPSIFD`
- `degreesFromDMS`
- `validCoordinates`
- `parseExifOrientation` (has its own IFD loop on purpose)
- `tiffIFD0Orientation`
- `tiffOrder` (lives in `raw.go`)

- [ ] **Step 1: Run covering tests on the Task 1 tree**

Run:

```bash
go test -race -count=1 ./internal/imaging/
```

Expected: PASS. If not, stop and report **BLOCKED** (Task 1 incomplete).

- [ ] **Step 2: Create `internal/imaging/exififd.go`**

Create the file with **exactly** this content (copied from current `exif.go`; do not rewrite comments):

```go
package imaging

import (
	"encoding/binary"
	"strings"
)

// walkIFD calls fn once per readable entry in the IFD at ifdOffset within
// tiff. Entries with an unrecognized type, an implausible count, or a
// value/offset that doesn't fit inside tiff are silently skipped rather
// than reported - see ReadMetadata's comment on why that's the right
// failure mode here.
func walkIFD(tiff []byte, bo binary.ByteOrder, ifdOffset uint32, fn func(tag, typ uint16, val []byte)) {
	if ifdOffset+2 > uint32(len(tiff)) {
		return
	}

	numEntries := bo.Uint16(tiff[ifdOffset : ifdOffset+2])
	entriesStart := ifdOffset + 2

	for i := uint32(0); i < uint32(numEntries); i++ {
		entryOffset := entriesStart + i*12

		if entryOffset+12 > uint32(len(tiff)) {
			break
		}

		tag := bo.Uint16(tiff[entryOffset : entryOffset+2])
		typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])
		count := bo.Uint32(tiff[entryOffset+4 : entryOffset+8])

		size := tagComponentSize(typ)
		// A count this large is either a corrupt file or a hostile one -
		// either way the tags this reader looks for are all single values
		// or short strings, so anything past a generous cap is skipped
		// rather than trusted enough to compute a byte length from.
		if size == 0 || count == 0 || count > 1<<16 {
			continue
		}

		total := uint64(size) * uint64(count)
		if total > 1<<20 {
			continue
		}

		var val []byte
		if total <= 4 {
			val = tiff[entryOffset+8 : uint64(entryOffset)+8+total]
		} else {
			offset := bo.Uint32(tiff[entryOffset+8 : entryOffset+12])
			if uint64(offset)+total > uint64(len(tiff)) {
				continue
			}
			val = tiff[offset : uint64(offset)+total]
		}

		fn(tag, typ, val)
	}
}

// tagComponentSize returns the byte size of one component of Exif type typ,
// or 0 for a type this reader doesn't know how to decode.
func tagComponentSize(typ uint16) int {
	switch typ {
	case 1, 2, 6, 7: // BYTE, ASCII, SBYTE, UNDEFINED
		return 1
	case 3, 8: // SHORT, SSHORT
		return 2
	case 4, 9, 13: // LONG, SLONG, IFD (SubIFD pointers)
		return 4
	case 5, 10: // RATIONAL, SRATIONAL
		return 8
	default:
		return 0
	}
}

// asciiValue decodes val as an Exif ASCII value (type 2): NUL-terminated,
// often with trailing padding. Returns ok=false for a wrong type or a
// value that's empty once trimmed, so callers can just skip setting the
// field.
func asciiValue(typ uint16, val []byte) (string, bool) {
	if typ != 2 {
		return "", false
	}

	s := strings.TrimRight(string(val), "\x00")
	s = strings.TrimSpace(s)

	if s == "" {
		return "", false
	}

	return s, true
}

// uintValue decodes val as an unsigned integer from a SHORT or LONG entry
// (the two Exif types this reader treats as plain counts: ISO and the
// Exif SubIFD pointer).
func uintValue(bo binary.ByteOrder, typ uint16, val []byte) (uint32, bool) {
	switch typ {
	case 3: // SHORT
		if len(val) < 2 {
			return 0, false
		}
		return uint32(bo.Uint16(val[:2])), true
	case 4, 13: // LONG, IFD
		if len(val) < 4 {
			return 0, false
		}
		return bo.Uint32(val[:4]), true
	}

	return 0, false
}

// rationalValue decodes val as an unsigned RATIONAL (type 5: a numerator and
// denominator, each a LONG) - the type Exif uses for exposure time,
// aperture, and focal length. ok is false for a wrong type, a truncated
// value, or a zero denominator.
func rationalValue(bo binary.ByteOrder, typ uint16, val []byte) (float64, bool) {
	if typ != 5 || len(val) < 8 {
		return 0, false
	}

	num := bo.Uint32(val[0:4])
	den := bo.Uint32(val[4:8])

	if den == 0 {
		return 0, false
	}

	return float64(num) / float64(den), true
}

// rationalsValue decodes val as n consecutive unsigned RATIONALs - the
// shape Exif uses for a GPS coordinate's degrees/minutes/seconds triple.
// ok is false for a wrong type, a value holding fewer than n rationals, or
// any zero denominator among them.
func rationalsValue(bo binary.ByteOrder, typ uint16, val []byte, n int) ([]float64, bool) {
	if typ != 5 || len(val) < n*8 {
		return nil, false
	}

	out := make([]float64, n)

	for i := range out {
		num := bo.Uint32(val[i*8 : i*8+4])
		den := bo.Uint32(val[i*8+4 : i*8+8])

		if den == 0 {
			return nil, false
		}

		out[i] = float64(num) / float64(den)
	}

	return out, true
}
```

- [ ] **Step 3: Delete those six functions from `exif.go`**

After the delete, `exif.go` must end at `validCoordinates`. Keep `encoding/binary` (still used by `parseExifOrientation`, `tiffIFD0Orientation`, `parseExifMetadata`, `parseGPSIFD`). Keep `strings` (still used by `parseGPSIFD`). Do not drop `math`.

- [ ] **Step 4: Move `TestRationalsValue` to `internal/imaging/exififd_test.go`**

Create:

```go
package imaging

import (
	"encoding/binary"
	"testing"
)

func TestRationalsValue(t *testing.T) {
	bo := binary.LittleEndian

	rationals := func(pairs ...uint32) []byte {
		b := make([]byte, 0, len(pairs)*4)
		for _, v := range pairs {
			b = binary.LittleEndian.AppendUint32(b, v)
		}
		return b
	}

	t.Run("three rationals", func(t *testing.T) {
		got, ok := rationalsValue(bo, 5, rationals(1, 2, 3, 4, 10, 4), 3)

		if !ok || len(got) != 3 || got[0] != 0.5 || got[1] != 0.75 || got[2] != 2.5 {
			t.Errorf("rationalsValue() = %v, %v, want [0.5 0.75 2.5], true", got, ok)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 4, rationals(1, 2, 3, 4, 5, 6), 3); ok {
			t.Error("rationalsValue() ok = true for a LONG entry, want false")
		}
	})

	t.Run("truncated", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 5, rationals(1, 2, 3, 4), 3); ok {
			t.Error("rationalsValue() ok = true for two rationals, want false")
		}
	})

	t.Run("zero denominator", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 5, rationals(1, 0, 3, 4, 5, 6), 3); ok {
			t.Error("rationalsValue() ok = true for a zero denominator, want false")
		}
	})
}
```

Delete `TestRationalsValue` from `exif_test.go`. Leave `TestDegreesFromDMS` and `TestValidCoordinates` in `exif_test.go`. If `encoding/binary` is still used there (builders), keep the import.

- [ ] **Step 5: Format and re-run covering tests**

Run:

```bash
gofmt -w internal/imaging/exif.go internal/imaging/exififd.go internal/imaging/exif_test.go internal/imaging/exififd_test.go
go test -race -count=1 ./internal/imaging/
```

Expected: PASS.

- [ ] **Step 6: Stop. Do not commit. Do not start Task 3.**

**Parent review checklist:**
- `exif.go` still has `parseExifMetadata`, `parseGPSIFD`, `degreesFromDMS`, `validCoordinates`.
- `parseExifOrientation` still has its private IFD loop (not rewritten to call `walkIFD`).
- `raw.go` / `jpegexif.go` untouched.
- `walkIFD` godoc still points at `ReadMetadata`’s failure-mode comment.

---

### Task 3: Docs and todo bookkeeping

**Model:** `composer-2.5-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)
**subagent_type:** `go-expert` (both)

**Files:**
- Modify: `ARCHITECTURE.md` — `internal/imaging` table only, plus keep the two “Where to look” EXIF lines accurate
- Modify: `todos.md` — move the exif bullet from `## TODO` to `## Done`
- Test: `make fmt-check && go vet ./... && go build ./... && go test -race ./internal/imaging/ ./internal/ui/exifwin/`

**Interfaces:**
- Consumes: the three source files from Tasks 1–2.
- Produces: locator rows; a Done bullet. No Go edits unless `gofmt`/`goimports` on the new files is still dirty (then format only those files).

- [ ] **Step 1: Run the focused suite**

```bash
go test -race -count=1 ./internal/imaging/ ./internal/ui/exifwin/
```

Expected: PASS. If not, stop and report **BLOCKED**.

- [ ] **Step 2: Update `ARCHITECTURE.md`**

In the `internal/imaging` file table, replace the single `exif.go` row with:

```markdown
| `exif.go` | Orientation tags + `ReadMetadata` / `Metadata` (including GPS IFD). JPEG APP1, then TIFF IFD0, then HEIC/AVIF, then RAW preview APP1. |
| `exififd.go` | Unexported IFD walker (`walkIFD`) and tag value helpers used by `exif.go`. |
| `exifformat.go` | Unexported display formatters for exposure, focal length, and Exif dates (`formatExposureTime` / `formatExifDate` / `parseExifDateTime`). |
```

Keep the existing `orientation.go` and `jpegexif.go` rows as they are. Do not add per-function commentary beyond the one-sentence locators above.

The “Where to look” lines that name `internal/imaging/exif.go` `parseGPSIFD` and orientation stay correct (`parseGPSIFD` did not move). Do not retarget them at `exififd.go`.

- [ ] **Step 3: Update `todos.md`**

Remove the `## TODO` bullet about `exif.go`. Add under `## Done` (same one-line style as neighbouring Done entries):

```markdown
- Split `internal/imaging/exif.go` into `exif.go` (parsers + GPS), `exififd.go`
  (IFD walker / tag values), and `exifformat.go` (display formatters). Behavior
  unchanged (2026-08-25).
```

Leave `## TODO` as an empty section (or delete the section if the file’s convention after the last item is to drop it — match neighbouring style; after finishLoad the remaining item *was* this bullet, so `## TODO` may be empty). Do not touch Menu Window / Menu Actions / the Windows grid Ctrl-click note.

- [ ] **Step 4: Match CI on the touched packages, then format-check**

Run from the repository root:

```bash
make fmt-check
go vet ./...
go build ./...
go test -race ./internal/imaging/ ./internal/ui/exifwin/
```

Expected: all PASS. If `fmt-check` fails on the new files, run `make fmt` on them and re-check. Do not format unrelated files.

If the parent wants the full `go test -race ./...` before merge, that is the parent’s verification step, not this task’s (full UI suite is long; this task’s contract is imaging + exifwin + vet/build/fmt).

- [ ] **Step 5: Stop. Do not commit.**

Suggested commit message (parent offers this to the user; do not run `git commit`):

```
Split imaging EXIF parse, IFD walk, and display formatters into three files.

Keep both container parsers in exif.go; move walkIFD/tag helpers and the
exposure/focal/date formatters so the 700-line file is greppable without
changing ReadMetadata or orientation behavior.
```

**Parent review checklist:**
- `ARCHITECTURE.md` cells are one-sentence locators, not essays.
- `todos.md` does not still list the split under `## TODO`.
- No stray edits to `load.go` / other leftover worktree files.

---

## Execution notes for the parent (this session)

1. Wait for the human to confirm locked decisions (or overrides) and to say to start.
2. Dispatch Task 1 implementer only. Review the diff. Fix if needed. Then Task 1 reviewer.
3. Repeat for Task 2, then Task 3.
4. Optional: final whole-branch review with `claude-opus-5-thinking-high` against `git diff` of the six imaging files + two docs files.
5. Do not commit. Offer the suggested message from Task 3.

## Self-review (plan author)

1. **Spec coverage:** Todo asked for a parse/format split of a file that also contains IFD walking. Tasks 1 (format) + 2 (IFD) + 3 (docs) cover that. Walker unification and orientation/metadata parser split are explicitly out of scope.
2. **Placeholders:** None. New files are specified in full.
3. **Type consistency:** Function names and signatures are unchanged across tasks. `parseExifDateTime` is defined as living in `exifformat.go` in Task 1 and must not be moved in Task 2.
