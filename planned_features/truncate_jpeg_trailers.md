# Truncate MPF / motion-photo trailers when stripping JPEG metadata

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task before dispatching the next. Do not
> start Task N+1 until Task N is reviewed and fixed. **Do not run
> `git commit`.**

**Goal:** `StripJPEGMetadata` must drop everything after the primary JPEG
EOI (second MPF frame, Google/Samsung motion-photo video, padding), so
GPS/Exif hiding in a concatenated copy cannot survive a privacy strip.

**Architecture:** Reuse the existing unexported `jpegLength` /
`scanToEOI` pair in `internal/imaging/raw.go` (already finds the first
real EOI, stuffing `FF 00` and RST markers). Do **not** write a second
EOI scanner and do **not** extract a shared header walker. The lossless
strip path (`stripJPEGSegments`) copies the entropy-coded scan only
through that EOI. `jpegHasRemovableMetadata` treats “bytes after the
primary EOI” as removable, so `CanStripJPEGMetadata` shows the EXIF
window button for trailer-only files. Orientation 2–8 already
re-encodes via `image/jpeg.Encode` and therefore already drops trailers;
pin that with a test, do not add a second truncate.

**Tech Stack:** Go, stdlib `image/jpeg`, existing `spliceMetadataIntoJPEG`
/ GPS TIFF helpers. No new modules.

**Spec:** `todos.md` TODO “Truncate MPF / motion-photo trailers when
stripping JPEG metadata”.

**Precedent:** `planned_features/remove_metadata.md` (MPF APP2 already
dropped in the header; this plan finishes the job). `jpegLength` in
`raw.go` is the EOI oracle RAW preview extraction already trusts.

---

## Open questions (answer before dispatch)

The plan below is written against the **recommended defaults**. If Florian
answers differently, the parent patches this file before Task 1.

| # | Question | Recommended default |
|---|----------|---------------------|
| 1 | Drop **all** bytes after the primary EOI (second JPEG, MP4 motion-photo video, unknown padding), or only drop a trailer that itself starts with JPEG SOI (`FF D8`) so a still+video motion photo keeps its video? | **Drop everything after EOI.** The video can carry location/time too; keeping it after deleting the MPF/XMP index is a privacy hole. The still (primary JPEG) remains. |
| 2 | Should a JPEG whose **only** privacy issue is a trailer (header already clean) show **Remove Metadata** and rewrite? | **Yes.** `jpegHasRemovableMetadata` must be true when `jpegLength(data) > 0 && jpegLength(data) < len(data)`. Otherwise the button stays hidden and the trailer is immortal. |
| 3 | JPEG with **no EOI** (truncated): invent a cut, error, or keep today’s copy-through-EOF? | **Copy through EOF** when `jpegLength` returns 0. Do not invent an EOI. |
| 4 | Keep any “safe” trailer (ICC after EOI, zero padding)? | **No.** Nothing after EOI is needed to decode the still. Padding is unused; ICC belongs in APP2 before SOS (already kept). |

---

## Dispatch order and models

Parent: this session. One implementer at a time. After each task: parent
reviews the diff and fixes if needed, then a reviewer subagent, then the
next task.

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | Trailer detection + lossless truncate in `stripJPEGSegments` | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `gpt-5.6-sol-medium` |
| 2 | `StripJPEGMetadata` / `CanStripJPEGMetadata` file tests + comments | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `gpt-5.6-sol-medium` |
| 3 | Manuals, `ARCHITECTURE.md`, `todos.md` | `generalPurpose` · `composer-2.5-fast` | `generalPurpose` · `gpt-5.6-sol-medium` |
| Final | Whole-branch review after Task 3 | — | `generalPurpose` · `claude-opus-5-thinking-high` |

Task 1 is the only binary-format change. It stays on Sonnet because the
plan reuses `jpegLength` rather than designing a new parser. Do **not**
use Opus for implementation unless Task 1 is blocked on EOI disagreement
between `jpegLength` and `stripJPEGSegments`. Do **not** parallelize
Tasks 1–3: they share `jpegexif.go` / tests / docs.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the strip story
  changes (Task 3).
- Viewer-independent packages (`internal/imaging`) return errors; they
  do not call `fyne.LogError`.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do **not** extract a shared `walkJPEGSegments` used by
  `jpegMetadataSegments` / `jpegEXIFOrientation` / `stripJPEGSegments`.
  Reuse **only** `jpegLength` (and `scanToEOI` under it).
- Do **not** parse CIPA MPF IFDs, rewrite MP4, or re-strip the secondary
  JPEG in place. Truncate; do not multiplex.
- Do **not** migrate `exifwin.warmDone` onto `internal/completion.Signal`.
- Do **not** add real camera/phone fixtures (licensing, size). Synthesize
  with `jpeg.Encode` + `spliceMetadataIntoJPEG` + `append`.
- Do **not** change UI strings or `translations/*.json` (manuals are
  markdown, not `lang.L` keys).
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 2.

---

## File map

| File | Role |
|------|------|
| `internal/imaging/raw.go` | Existing `jpegLength` / `scanToEOI`. **Do not modify** unless a test proves they miss a real EOI (escalate to parent). |
| `internal/imaging/jpegexif.go` | `jpegHasTrailer`, `jpegHasRemovableMetadata`, `stripJPEGSegments` |
| `internal/imaging/jpegexif_test.go` | Segment-level trailer tests |
| `internal/imaging/save.go` | Comments on `StripJPEGMetadata` / `CanStripJPEGMetadata` only |
| `internal/imaging/save_test.go` | File-level strip + `CanStripJPEGMetadata` trailer case |
| `internal/ui/help/manual.md` | Replace the “may keep that copy” bullet |
| `internal/ui/help/manual_de.md` | Same in German |
| `ARCHITECTURE.md` | `jpegexif.go` / `save.go` rows: truncate at primary EOI |
| `todos.md` | Move this TODO under Done |

---

## Why this is a privacy bug

`stripJPEGSegments` walks COM/APPn until SOS, then:

```go
if marker == 0xDA {
    out = append(out, data[pos:]...) // SOS through EOF, including trailers
    return out, nil
}
```

MPF APP2 in the **header** is already dropped (`keepOnStrip` is false for
`MPF\x00`). The extra picture (or MP4) lives **after** the first `FF D9`.
`ReadMetadata` / the EXIF window only walk the primary header, so the
UI can look clean while a second JPEG’s GPS is still on disk.

`jpeg.Decode` also stops at the first EOI, so the viewer never shows the
hidden copy. Strip without truncate is therefore a silent fail.

Orientation 2–8 goes through `encodeJPEGKeepingICC` → `jpeg.Encode`,
which emits a single JPEG ending at EOI. Trailers die on that path as a
side effect. Orientation 1 is the lossless path that must start using
`jpegLength`.

---

## Locked design (implementers)

1. **Cut point:** `end := jpegLength(data)`. If `end > pos && end <= len(data)`,
   copy `data[pos:end]` at SOS. Otherwise copy `data[pos:]` (today’s
   behavior: no closed EOI).
2. **Trailer means removable:**
   `jpegHasTrailer(data)` is `n := jpegLength(data); n > 0 && n < len(data)`.
   `jpegHasRemovableMetadata` returns true if it would drop a header
   segment **or** `jpegHasTrailer(data)`.
3. **Progressive JPEGs:** `jpegLength` already, on the first SOS, scans
   to the first real EOI (later SOS markers sit in the entropy region).
   Copying `data[firstSOS:eoi]` keeps those extra scans. Do not cut at SOS.
   No progressive fixture required (stdlib `jpeg.Encode` is baseline).
4. **Do not** change `keepOnStrip`. MPF APP2 stays dropped in the header.
5. **Do not** change `encodeJPEGKeepingICC` / the orientation 2–8 branch
   except comments/tests.
6. Existing test
   `TestStripJPEGSegments` / `"orientation-1 GPS JPEG is bit-identical in the scan after strip"`
   must still pass: those fixtures have no trailer, so
   `data[sos:]` is already the scan through EOI.

---

### Task 1: Detect trailers and truncate `stripJPEGSegments`

**Files:**
- Modify: `internal/imaging/jpegexif.go`
- Modify: `internal/imaging/jpegexif_test.go`
- Do not modify: `internal/imaging/raw.go`

**Interfaces:**
- Consumes: `jpegLength([]byte) int` in `raw.go` (same package).
- Produces:
  - `func jpegHasTrailer(data []byte) bool`
  - `jpegHasRemovableMetadata` also true when `jpegHasTrailer` is true
  - `stripJPEGSegments` output ends at the primary EOI when `jpegLength`
    succeeds

- [ ] **Step 1: Write the failing tests** in `jpegexif_test.go`

Add helpers next to `spliceMetadataIntoJPEG` (same file, same package).
Reuse `wrapAsAPP1`, `buildGPSExifTIFF`, `eiffelGPS`, `markedImage`,
`wrapAPP2`, `spliceMetadataIntoJPEG`. Do **not** invent a new GPS TIFF
builder. Metadata checks use `ReadMetadata(data).HasGPS` and
`.Empty()` (same as the existing GPS strip subtest).

```go
func stdJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(4, 4), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func gpsJPEG(t *testing.T) []byte {
	t.Helper()
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
	data := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})
	if !ReadMetadata(data).HasGPS {
		t.Fatal("setup: gpsJPEG must carry GPS")
	}
	return data
}

func concatJPEG(primary, trailer []byte) []byte {
	out := make([]byte, 0, len(primary)+len(trailer))
	out = append(out, primary...)
	out = append(out, trailer...)
	return out
}
```

Append to `TestJPEGHasRemovableMetadata` (keep the existing asserts):

```go
	plain := stdJPEG(t) // or keep the existing Encode into `plain`
	if jpegHasRemovableMetadata(concatJPEG(plain, []byte("GARBAGE"))) {
		// only if jpegLength(plain) == len(plain); it must
	}
```

Better as a dedicated test so the existing function stays readable:

```go
func TestJPEGHasRemovableMetadata_Trailer(t *testing.T) {
	plain := stdJPEG(t)
	n := jpegLength(plain)
	if n == 0 || n != len(plain) {
		t.Fatalf("setup: stdlib JPEG jpegLength=%d len=%d (want n==len>0)", n, len(plain))
	}
	if jpegHasRemovableMetadata(plain) {
		t.Fatal("plain JPEG has no trailer and no removable segments")
	}
	if !jpegHasRemovableMetadata(concatJPEG(plain, []byte("GARBAGE"))) {
		t.Fatal("bytes after EOI are removable")
	}
	if !jpegHasRemovableMetadata(concatJPEG(plain, gpsJPEG(t))) {
		t.Fatal("concatenated GPS JPEG is removable")
	}
	if jpegHasRemovableMetadata(plain[:len(plain)-2]) {
		// chopped EOI: jpegLength is 0, no header Exif → not removable
		// (we cannot locate a trailer without an EOI)
	}
}
```

For the chopped-EOI assert: `plain[:len(plain)-2]` may still be a JPEG
SOI with no EOI. `jpegHasRemovableMetadata` must be **false** (no header
tags, `jpegLength` 0). If that slice panics inside `jpegLength`, that is
a `jpegLength` bug — do not paper over it; stop and tell the parent.

Add subtests on `TestStripJPEGSegments`:

```go
	t.Run("drops concatenated GPS JPEG after primary EOI", func(t *testing.T) {
		primary := stdJPEG(t)
		trailer := gpsJPEG(t)
		data := concatJPEG(primary, trailer)

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("stripped file still has removable metadata or a trailer")
		}
		if bytes.Contains(got, trailer) {
			t.Fatal("trailer JPEG still present")
		}
		if ReadMetadata(got).HasGPS {
			t.Fatal("GPS survived in the primary (trailer was the only GPS)")
		}
		if end := jpegLength(got); end != len(got) {
			t.Fatalf("stripped file still has bytes after EOI: jpegLength=%d len=%d", end, len(got))
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped primary must still decode: %v", err)
		}
		// Lossless: primary pixels/scan unchanged because primary had
		// nothing to drop in the header.
		if !bytes.Equal(got, primary) {
			t.Fatal("orientation-1 plain primary must be byte-identical after dropping only a trailer")
		}
	})

	t.Run("drops MPF APP2 and the extra frame after EOI", func(t *testing.T) {
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))
		primary := spliceMetadataIntoJPEG(t, markedImage(4, 4), [][]byte{mpf})
		trailer := gpsJPEG(t)
		data := concatJPEG(primary, trailer)

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if bytes.Contains(got, []byte("MPF\x00")) {
			t.Fatal("left MPF APP2")
		}
		if bytes.Contains(got, trailer) || ReadMetadata(got).HasGPS {
			t.Fatal("left extra frame or its GPS")
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped file must still decode: %v", err)
		}
	})

	t.Run("drops a non-JPEG motion-photo trailer", func(t *testing.T) {
		// Google Micro Video / Samsung motion photo: MP4 sits after EOI.
		// A real ftyp box is unnecessary; any post-EOI bytes must go.
		primary := stdJPEG(t)
		mp4 := []byte("\x00\x00\x00\x18ftypisomGARBAGE")
		data := concatJPEG(primary, mp4)

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if bytes.Contains(got, []byte("ftyp")) {
			t.Fatal("left motion-photo video trailer")
		}
		if !bytes.Equal(got, primary) {
			t.Fatal("plain primary must be unchanged aside from losing the trailer")
		}
	})

	t.Run("no EOI copies the scan through EOF", func(t *testing.T) {
		primary := stdJPEG(t)
		if primary[len(primary)-2] != 0xFF || primary[len(primary)-1] != 0xD9 {
			t.Fatal("setup: stdlib JPEG should end in EOI")
		}
		truncated := primary[:len(primary)-2] // strip EOI
		got, err := stripJPEGSegments(truncated)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		// Cannot locate EOI → do not invent a cut; result still has no EOI
		// and must still be a JPEG header+scan the same length as input
		// minus any removable header segments (there are none).
		if jpegLength(truncated) != 0 {
			t.Fatal("setup: jpegLength must be 0 without EOI")
		}
		if len(got) != len(truncated) {
			t.Fatalf("len(got)=%d len(in)=%d; without EOI, copy through EOF", len(got), len(truncated))
		}
	})
```

Use `ReadMetadata` / `HasGPS` / `Empty` exactly as
`TestStripJPEGSegments`'s GPS subtest already does.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test -count=1 ./internal/imaging/ -run 'TestJPEGHasRemovableMetadata_Trailer|TestStripJPEGSegments' -v
```

Expected: FAIL — trailer still present / `jpegHasRemovableMetadata`
false for `concatJPEG(plain, "GARBAGE")`. Existing subtests of
`TestStripJPEGSegments` must still pass.

- [ ] **Step 3: Implement**

In `jpegexif.go`, next to `jpegHasRemovableMetadata`:

```go
// jpegHasTrailer reports whether data is a JPEG whose primary EOI is
// followed by extra bytes (MPF extra frames, motion-photo video,
// padding). A JPEG jpegLength cannot close is false.
func jpegHasTrailer(data []byte) bool {
	n := jpegLength(data)
	return n > 0 && n < len(data)
}
```

Change `jpegHasRemovableMetadata` so the final `return false` becomes
`return jpegHasTrailer(data)`. Keep the early `return true` when a
header segment is droppable. Update the doc comment: true if a COM/APPn
segment would be dropped **or** bytes follow the primary EOI.

In `stripJPEGSegments`, replace the SOS branch:

```go
		if marker == 0xDA {
			end := jpegLength(data)
			if end > pos && end <= len(data) {
				out = append(out, data[pos:end]...)
			} else {
				out = append(out, data[pos:]...)
			}
			return out, nil
		}
```

Update the `stripJPEGSegments` doc comment: the entropy-coded scan is
copied through the **primary EOI**; bytes after that EOI are omitted.
A JPEG `jpegLength` cannot close still copies through EOF.

Do **not** copy `scanToEOI`. Call `jpegLength(data)` from SOI (index 0),
not from SOS. Progressive extra SOS markers then stay inside
`data[pos:end]`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -count=1 ./internal/imaging/ -run 'TestJPEGHasRemovableMetadata|TestStripJPEGSegments|TestKeepOnStrip' -v
```

Expected: PASS.

Then from repo root:

```bash
gofmt -l .
go vet ./...
go build ./...
```

`gofmt -l .` must print nothing. If it lists files, run `gofmt -w` on
them and re-check.

- [ ] **Step 5: Suggested commit message** (do not commit)

```
imaging: truncate JPEG strip at the primary EOI

Drop MPF extra frames and motion-photo trailers so GPS hiding after
the first EOI cannot survive Remove Metadata.
```

---

### Task 2: File-level `StripJPEGMetadata` + exported `CanStrip` surface

**Files:**
- Modify: `internal/imaging/save.go` (comments only)
- Modify: `internal/imaging/save_test.go`

**Interfaces:**
- Consumes: Task 1’s `jpegHasRemovableMetadata` / `stripJPEGSegments`.
- Produces: no new exported API. `CanStripJPEGMetadata` becomes true for
  trailer-only JPEGs automatically. `StripJPEGMetadata` rewrites those
  files through the existing orientation-1 branch.

- [ ] **Step 1: Write the failing tests** in `save_test.go`

Reuse `stdJPEG` / `gpsJPEG` / `concatJPEG` **only if they already exist
in this package’s tests after Task 1** (`jpegexif_test.go` is the same
package, so they are callable). Do not duplicate them in `save_test.go`.

If Task 1 put those helpers in `jpegexif_test.go`, call them from
`save_test.go`. If that feels messy, move `stdJPEG`, `gpsJPEG`, and
`concatJPEG` to `jpegexif_test.go` in Task 1 (already specified there)
and use them here.

```go
func TestCanStripJPEGMetadata_TrailerOnly(t *testing.T) {
	plain := stdJPEG(t)
	if CanStripJPEGMetadata(plain) {
		t.Fatal("plain JPEG is not strippable")
	}
	if !CanStripJPEGMetadata(concatJPEG(plain, gpsJPEG(t))) {
		t.Fatal("trailer-only JPEG must be strippable")
	}
	if !CanStripJPEGMetadata(concatJPEG(plain, []byte("ftypisom"))) {
		t.Fatal("motion-photo trailer must be strippable")
	}
}
```

Add the trailer-only case to `TestCanStripJPEGMetadata`’s table
**instead** of a new test if that table is the one existing test — do
not fork two `CanStrip` tests. The table already lives in
`TestCanStripJPEGMetadata`; add:

```go
{"trailer-only concatenated JPEG", concatJPEG(plainBuf.Bytes(), gpsJPEG(t)), true},
{"motion-photo ftyp trailer", concatJPEG(plainBuf.Bytes(), []byte("\x00\x00\x00\x18ftypisom")), true},
```

`plainBuf` is already built in that test. Use `gpsJPEG(t)` from Task 1.

```go
func TestStripJPEGMetadata_DropsConcatenatedGPSTrailer(t *testing.T) {
	plain := stdJPEG(t)
	trailer := gpsJPEG(t)
	path := writeTempFile(t, "mpf.jpg", concatJPEG(plain, trailer))
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if bytes.Contains(got, trailer) {
		t.Fatal("concatenated GPS JPEG survived StripJPEGMetadata")
	}
	if ReadMetadata(got).HasGPS {
		t.Fatal("GPS survived")
	}
	if end := jpegLength(got); end != len(got) {
		t.Fatalf("bytes after EOI remain: jpegLength=%d len=%d", end, len(got))
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("orientation-1 primary with no header tags must be unchanged")
	}
}

func TestStripJPEGMetadata_Orientation6DropsTrailer(t *testing.T) {
	primary := halfRedHalfBlueJPEG(t, 20, 10, 6)
	data := concatJPEG(primary, gpsJPEG(t))
	path := writeTempFile(t, "rot-mpf.jpg", data)
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if bytes.Contains(got, gpsJPEG(t)) {
		t.Fatal("trailer survived orientation 2–8 re-encode")
	}
	if ReadMetadata(got).HasGPS {
		t.Fatal("GPS survived")
	}
	if end := jpegLength(got); end == 0 || end != len(got) {
		t.Fatalf("re-encode must be a single JPEG: jpegLength=%d len=%d", end, len(got))
	}
}

func TestStripJPEGMetadata_TrailerStripIsIdempotent(t *testing.T) {
	path := writeTempFile(t, "once.jpg", concatJPEG(stdJPEG(t), gpsJPEG(t)))
	u := storage.NewFileURI(path)
	if err := StripJPEGMetadata(u); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()
	if err := StripJPEGMetadata(u); err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(mtime) {
		t.Fatal("second strip must be a noop")
	}
}
```

Names: `writeTempFile`, `mustRead`, `halfRedHalfBlueJPEG`,
`storage.NewFileURI` already exist in `save_test.go`. Use those exact
helpers. GPS checks: `ReadMetadata(got).HasGPS` and `.Empty()`, same as
`TestStripJPEGMetadata_RemovesGPSWithoutTouchingPixelsWhenOrientation1`.

The orientation-6 trailer test builds `gpsJPEG(t)` twice; assign
`trailer := gpsJPEG(t)` once and `concatJPEG(primary, trailer)` /
`bytes.Contains(got, trailer)`.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test -count=1 ./internal/imaging/ -run 'TestCanStripJPEGMetadata|TestStripJPEGMetadata_DropsConcatenatedGPSTrailer|TestStripJPEGMetadata_Orientation6DropsTrailer|TestStripJPEGMetadata_TrailerStripIsIdempotent' -v
```

Expected: FAIL until Task 1 is present. If Task 1 already landed,
`CanStrip` / drop-trailer tests may already pass; the orientation-6 test
should already pass too (re-encode). If orientation-6 fails because
`bytes.Contains(got, trailer)` is the wrong check (re-encode could
coincidentally contain a substring), compare using the trailer slice
captured **before** strip, and also `jpegLength(got) == len(got)`.

- [ ] **Step 3: Update comments in `save.go`**

On `StripJPEGMetadata`, mention that extra bytes after the primary EOI
(MPF extra images, motion-photo video) are discarded. On
`CanStripJPEGMetadata`, mention trailers as a reason it returns true.

No production logic change in `save.go` unless tests prove
`jpegHasRemovableMetadata` is not consulted (it is: orientation 1 noops
when `!jpegHasRemovableMetadata && orient == 1`).

- [ ] **Step 4: Run tests**

```bash
go test -count=1 ./internal/imaging/ -run 'TestStripJPEGMetadata|TestCanStripJPEGMetadata' -v
gofmt -l .
go vet ./...
go build ./...
```

Expected: PASS, `gofmt -l .` empty.

Parent after this task: `go test -race ./...` from the repo root.

- [ ] **Step 5: Suggested commit message** (do not commit)

```
imaging: treat JPEG trailers as removable metadata

Show Remove Metadata for trailer-only files and pin that
StripJPEGMetadata drops concatenated GPS and motion-photo video.
```

---

### Task 3: Manuals, architecture, todos

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md`
- Modify: `todos.md`

**Interfaces:** none.

- [ ] **Step 1: English manual**

In `internal/ui/help/manual.md`, replace the bullet that currently says
multi-picture / motion photos **may keep** the extra copy. New meaning:

- Extra data after the main JPEG (a second picture from multi-picture
  files, or the short video from a motion photo) is **removed** with the
  metadata. What remains is the still photo. This cannot be undone.

Keep surrounding bullets (JPEG only, ICC stays, orientation 2–8
re-encode, `R` is view-only) unchanged.

- [ ] **Step 2: German manual**

Same change in `internal/ui/help/manual_de.md` for the matching bullet
about Multi-Picture / Motion Photos. German, same meaning: the extra
copy / video is removed; the still remains; not undoable.

- [ ] **Step 3: `ARCHITECTURE.md`**

On the `jpegexif.go` row's strip
sentence, add that `stripJPEGSegments` copies the scan through the
primary EOI via `jpegLength` (`raw.go`) and drops trailers. On the
`save.go` `StripJPEGMetadata` sentence, add the same: trailers are
removable; orientation 2–8 re-encode already emits no trailer.

Do not rewrite those rows from scratch; splice one clause into each.

- [ ] **Step 4: `todos.md`**

Move “Truncate MPF / motion-photo trailers…” from TODO to Done (one
short line). Leave the `warmDone` / `completion.Signal` item under TODO.

- [ ] **Step 5: Verify**

```bash
gofmt -l .
go test -count=1 ./internal/ui/help/ ./internal/imaging/ -run 'TestStripJPEG|TestCanStrip|TestJPEGHasRemovable|TestStripJPEGSegments' -v
```

Help tests, if any, only check embed/parse; they should still pass.

- [ ] **Step 6: Suggested commit message** (do not commit)

```
docs: Remove Metadata drops JPEG trailers and motion-photo video
```

---

## Parent review checklist (after every task)

1. Diff matches this task only (no `warmDone`, no MPF IFD parser, no
   new walker, no `raw.go` rewrite).
2. Tests named in the task were run; output is in the implementer report.
3. `gofmt -l .` clean, `go vet` / `go build` clean.
4. No `git commit`.
5. Existing `TestStripJPEGSegments` scan-bit-identical subtest still
   passes.
6. Fix anything Important/Critical yourself or with a fix subagent
   (`go-expert` · `claude-sonnet-5-thinking-high`) before Task N+1.

After Task 2: `go test -race ./...` from the repository root.

After Task 3: dispatch the final whole-branch reviewer on
`claude-opus-5-thinking-high`, then parent runs `go test -race ./...`
again before declaring the feature done.

---

## Out of scope

- Parsing or rewriting CIPA MPF IFDs / keeping a second still.
- Re-muxing Google/Samsung motion-photo MP4.
- HEIC Live Photos (not JPEG trailers).
- Migrating `exifwin.warmDone` to `internal/completion.Signal`.
- Shared `walkJPEGSegments`.
- New dependencies, UI chrome, translation JSON keys.
- Real vendor fixtures.

---

## Self-review

1. **Spec coverage:** Truncate at primary EOI — Task 1. Trailer-only
   button/`CanStrip` — Tasks 1–2. Motion-photo MP4 — Task 1 ftyp test +
   Task 2. Orientation 2–8 — Task 2. Docs — Task 3. `warmDone` — out of
   scope.
2. **Placeholders:** none. Helpers `stdJPEG` / `gpsJPEG` / `concatJPEG`
   are specified. GPS builder is `buildGPSExifTIFF` + `eiffelGPS`.
3. **Type consistency:** `jpegLength([]byte) int`,
   `jpegHasTrailer([]byte) bool`, `stripJPEGSegments([]byte) ([]byte, error)`,
   `StripJPEGMetadata(fyne.URI) error`, `CanStripJPEGMetadata([]byte) bool`.
4. **Compile continuity:** Task 1 is self-contained in `internal/imaging`.
   Task 2 only comments + tests. Task 3 is docs.
