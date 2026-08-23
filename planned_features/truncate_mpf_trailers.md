# Truncate MPF / motion-photo trailers when stripping JPEG metadata

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task before dispatching the next. Do not
> start Task N+1 until Task N is reviewed and fixed. **Do not run
> `git commit`.**

**Goal:** `StripJPEGMetadata` must not leave a second JPEG (or motion-photo
video) after the primary EOI, because that trailer can carry its own
Exif/GPS after the header MPF APP2 index has already been dropped.

**Architecture:** At SOS, copy the entropy-coded scan only through the
primary EOI that `jpegLength` (`raw.go`) already finds, instead of through
EOF. Treat bytes after that EOI as removable, so `jpegHasRemovableMetadata`
/ `CanStripJPEGMetadata` stay true for trailer-only files and the EXIF
window button still appears. Orientation 2–8 already re-encodes via
`image/jpeg.Encode` and therefore drops trailers; add a regression test,
no extra branch.

**Tech Stack:** Go, stdlib `image/jpeg`, existing `jpegLength` /
`scanToEOI` in `internal/imaging/raw.go`. No new dependencies.

**Spec:** `todos.md` TODO “Truncate MPF / motion-photo trailers when
stripping JPEG metadata”.

**Precedent:** `stripJPEGSegments` SOS branch (`jpegexif.go`) currently
does `append(out, data[pos:]...)`. `jpegLength` already returns the
byte count through the first real EOI (stuffed `FF 00` and RST markers
handled). Header MPF APP2 is already dropped by `keepOnStrip`.

---

## Open questions (proposed defaults — confirm before dispatch)

Implementers treat the **Proposed** column as spec unless Florian
overrides it before Task 1 starts.

| # | Question | Proposed |
|---|----------|----------|
| 1 | Which trailers are safe to keep? | **None.** Drop every byte after the primary EOI (second JPEG, Google/Samsung motion-photo MP4, padding). Privacy strip: anything after EOI can hide GPS. |
| 2 | Trailer-only file (clean header, bytes after EOI)? | **Yes, strip it.** `jpegHasRemovableMetadata` is true; the Remove Metadata button shows; write truncates. |
| 3 | Motion photo becomes a still (video discarded)? | **Yes.** That is the point of this TODO. Manual stops saying the copy “may remain” and says the extra still/video is discarded. |
| 4 | No EOI (`jpegLength == 0`)? | Keep today’s copy-through-EOF. Do not invent a split. Do not treat that as removable. |
| 5 | Share `jpegLength` vs duplicate an EOI scan? | **Call `jpegLength`.** Same package. Do **not** extract a shared header walker for `jpegMetadataSegments` / `jpegEXIFOrientation`. |
| 6 | Parse CIPA MPF IFD offsets? | **No.** First EOI is the CIPA primary. Parsing IFDs is YAGNI. |

---

## Dispatch order and models

Parent: Florian’s session. One implementer at a time. After each task:
parent reviews the diff, fixes if needed, then dispatches the next.

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | Trailer detection + truncate `stripJPEGSegments` at primary EOI | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `claude-sonnet-5-thinking-high` |
| 2 | `StripJPEGMetadata` / `CanStripJPEGMetadata` file-level tests | `go-expert` · `composer-2.5-fast` | `generalPurpose` · `gpt-5.6-sol-medium` |
| 3 | Manuals, `ARCHITECTURE.md`, `todos.md` | `generalPurpose` · `composer-2.5-fast` | `generalPurpose` · `gpt-5.6-sol-medium` |
| Final | Whole-branch review after Task 3 | — | `generalPurpose` · `claude-opus-5-thinking-high` |

Task 1 is the only imaging change that can leak GPS if the EOI cut is
wrong. Do not downgrade it to the cheap model. Tasks 2–3 are transcription
from this plan.

Do **not** use Opus for Tasks 1–3. The work splits; Opus is the final
reviewer only.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the strip story
  changes (Task 3).
- Every user-visible string is `lang.L("English text")` with that exact
  key in every `translations/*.json` bundle. This plan adds **no** new
  UI strings (manual markdown only).
- Viewer-independent packages (`internal/imaging`) return errors; they do
  not call `fyne.LogError`.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do not extract a shared `walkJPEGSegments` used by `jpegMetadata`.
  Reuse `jpegLength` only.
- Do not migrate `exifwin.warmDone`.
- Do not parse MPF IFDs or keep “safe” trailer types.
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 2.

---

## File map

| File | Role |
|------|------|
| `internal/imaging/jpegexif.go` | `jpegHasRemovableMetadata` (trailer is removable); `stripJPEGSegments` copies SOS through primary EOI |
| `internal/imaging/jpegexif_test.go` | Unit tests for trailer truncate / detection |
| `internal/imaging/save.go` | Comment on `StripJPEGMetadata` (trailer dropped) |
| `internal/imaging/save_test.go` | File-level GPS-trailer tests |
| `internal/imaging/raw.go` | Unchanged; `jpegLength` / `scanToEOI` consumed by the strip path |
| `ARCHITECTURE.md` | `jpegexif.go` row (on-disk file: `jpegexif.go`) and `save.go` row mention primary-EOI truncate |
| `internal/ui/help/manual.md` | Replace the “may keep that copy” caveat |
| `internal/ui/help/manual_de.md` | Same in German |
| `todos.md` | Move this TODO under Done |

No EXIF-window code. `CanStripJPEGMetadata` already wraps
`jpegHasRemovableMetadata`.

---

## Assumptions (locked for implementers once Florian confirms the table)

1. **Cut at `jpegLength(data)`.** That value is an index into the original
   buffer (SOI through primary EOI). At SOS, copy `data[pos:end]`, not
   `data[pos:]`.
2. **Trailer-only is removable.** A stdlib JPEG plus any bytes after EOI
   makes `jpegHasRemovableMetadata` true even with no COM/APPn to drop.
3. **`jpegLength == 0`:** copy `data[pos:]` (today’s behavior). Do not
   error. Do not report removable solely because length is 0.
4. **Do not rewrite** a file that is already a closed JPEG with nothing
   removable (`TestStripJPEGMetadata_NoRemovableSegmentsIsNoop` must
   still pass).
5. **Orientation 2–8:** no new production branch. `encodeJPEGKeepingICC`
   already emits a single JPEG. Task 2 proves a GPS trailer is gone.
6. **Fixtures are synthetic.** Concatenate two `jpeg.Encode` outputs
   (second one with GPS APP1 via existing `spliceMetadataIntoJPEG` /
   `buildGPSExifTIFF`). Do not add camera samples under `testdata/`.
7. **Existing scan-identity test stays.**
   `orientation-1 GPS JPEG is bit-identical in the scan after strip`
   uses a file with no trailer; `data[sos:]` remains the full scan.

---

### Task 1: Truncate `stripJPEGSegments` at the primary EOI

**Files:**
- Modify: `internal/imaging/jpegexif.go`
- Modify: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: `jpegLength(data []byte) int` in `raw.go` (same package);
  existing `keepOnStrip`, `errNotJPEG`, `spliceMetadataIntoJPEG`,
  `wrapAsAPP1`, `wrapAPP2`, `buildGPSExifTIFF`, `eiffelGPS`, `markedImage`.
- Produces: same signatures as today:
  - `func jpegHasRemovableMetadata(data []byte) bool`
  - `func stripJPEGSegments(data []byte) ([]byte, error)`
  Behavior change only.

- [ ] **Step 1: Write the failing tests** in `jpegexif_test.go`

Add this helper next to `spliceMetadataIntoJPEG`:

```go
func appendAfterEOI(t *testing.T, jpeg, trailer []byte) []byte {
	t.Helper()
	n := jpegLength(jpeg)
	if n == 0 || n != len(jpeg) {
		t.Fatalf("setup: primary must be a closed JPEG with no trailer, jpegLength=%d len=%d", n, len(jpeg))
	}
	out := make([]byte, 0, len(jpeg)+len(trailer))
	return append(append(out, jpeg...), trailer...)
}

func gpsTrailerJPEG(t *testing.T) []byte {
	t.Helper()
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
	return spliceMetadataIntoJPEG(t, markedImage(4, 4), [][]byte{exif})
}
```

Extend `TestJPEGHasRemovableMetadata` with:

```go
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	if jpegHasRemovableMetadata(appendAfterEOI(t, plain, gpsTrailerJPEG(t))) != true {
		t.Fatal("GPS JPEG after primary EOI is removable")
	}
	if jpegHasRemovableMetadata(appendAfterEOI(t, plain, []byte("ftypmp42fake-video"))) != true {
		t.Fatal("motion-photo bytes after primary EOI are removable")
	}
```

Add a subtest on `TestStripJPEGSegments`:

```go
	t.Run("drops a GPS JPEG concatenated after the primary EOI", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))
		primary := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif, mpf})
		data := appendAfterEOI(t, primary, gpsTrailerJPEG(t))
		if !bytes.Contains(data, []byte("Exif\x00\x00")) {
			t.Fatal("setup: want Exif in the trailer file")
		}

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("still has removable metadata or a trailer")
		}
		if n := jpegLength(got); n != len(got) {
			t.Fatalf("stripped length %d, jpegLength %d (trailer left)", len(got), n)
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) || bytes.Contains(got, []byte("MPF\x00")) {
			t.Fatal("left identifying tags or MPF")
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped file must still decode: %v", err)
		}
		sos := bytes.Index(primary, []byte{0xFF, 0xDA})
		gotSOS := bytes.Index(got, []byte{0xFF, 0xDA})
		if sos < 0 || gotSOS < 0 {
			t.Fatal("missing SOS")
		}
		if !bytes.Equal(primary[sos:], got[gotSOS:]) {
			t.Fatal("lossless strip must copy the primary scan through EOI, not the trailer")
		}
	})

	t.Run("drops a trailer when the primary header has nothing removable", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		plain := buf.Bytes()
		data := appendAfterEOI(t, plain, gpsTrailerJPEG(t))

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatal(err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("plain primary plus dropped trailer must not look removable")
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) {
			t.Fatal("left trailer Exif")
		}
		if !bytes.Equal(got, plain) {
			t.Fatal("header-clean primary must be unchanged aside from dropping the trailer")
		}
	})

	t.Run("copy through EOF when the JPEG has no EOI", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		truncated := buf.Bytes()
		if n := jpegLength(truncated); n != len(truncated) {
			t.Fatalf("setup: stdlib JPEG should close, jpegLength=%d len=%d", n, len(truncated))
		}
		truncated = truncated[:len(truncated)-2] // drop EOI
		if jpegLength(truncated) != 0 {
			t.Fatal("setup: want jpegLength 0 after chopping EOI")
		}
		got, err := stripJPEGSegments(truncated)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if !bytes.Equal(got, truncated) {
			t.Fatal("no-EOI JPEG must still copy through EOF")
		}
	})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/imaging/ -run 'TestJPEGHasRemovableMetadata|TestStripJPEGSegments' -v`

Expected: FAIL — trailer still present / `jpegHasRemovableMetadata` false
for trailer-only.

- [ ] **Step 3: Minimal implementation** in `jpegexif.go`

Insert this check at the top of the existing `jpegHasRemovableMetadata` (after the SOI guard). Do **not** rewrite the header walk; keep its current `segLen` / `segEnd` locals:

```go
	if n := jpegLength(data); n > 0 && n < len(data) {
		return true
	}
```

Update the function comment so a trailer counts as removable. Full shape after the edit (walk unchanged):

```go
// jpegHasRemovableMetadata reports whether stripJPEGSegments would drop
// at least one COM/APPn segment or bytes after the primary EOI.
// Non-JPEG data is false.
func jpegHasRemovableMetadata(data []byte) bool {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return false
	}
	if n := jpegLength(data); n > 0 && n < len(data) {
		return true
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
		segEnd := pos + 2 + segLen
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !keepOnStrip(marker, data[pos+4:segEnd]) {
				return true
			}
		}
		pos = segEnd
	}
	return false
}
```

Update `stripJPEGSegments`’s comment and SOS branch:

```go
// stripJPEGSegments returns a copy of a JPEG with removable metadata
// segments (see keepOnStrip) omitted. DQT/DHT/SOF/DRI and the entropy-
// coded scan through the primary EOI are copied verbatim; bytes after
// that EOI (MPF extra pictures, motion-photo video) are dropped. data
// that is not a JPEG yields errNotJPEG. A JPEG with nothing removable
// returns a copy of data.
```

Replace the SOS arm:

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

Leave the rest of the loop unchanged.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/imaging/ -run 'TestJPEGHasRemovableMetadata|TestStripJPEGSegments|TestKeepOnStrip' -v`

Expected: PASS.

Then: `gofmt -l .` (empty), `go vet ./...`, `go build ./...`.

- [ ] **Step 5: Suggested commit message** (do not commit)

```
imaging: drop JPEG trailers after the primary EOI when stripping metadata
```

---

### Task 2: File-level `StripJPEGMetadata` coverage

**Files:**
- Modify: `internal/imaging/save.go` (godoc only)
- Modify: `internal/imaging/save_test.go`

**Interfaces:**
- Consumes: Task 1’s `stripJPEGSegments` / `jpegHasRemovableMetadata`;
  existing `StripJPEGMetadata`, `CanStripJPEGMetadata`, `writeTempFile`,
  `mustRead`, `spliceMetadataIntoJPEG`, `halfRedHalfBlueJPEG`.
- Produces: unchanged exported signatures.

- [ ] **Step 1: Write the failing tests** in `save_test.go`

Reuse Task 1’s helpers if they are in `jpegexif_test.go` (same package).
Do not duplicate `appendAfterEOI` / `gpsTrailerJPEG` in `save_test.go`.

```go
func TestStripJPEGMetadata_DropsGPSTrailerAfterPrimaryEOI(t *testing.T) {
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	})...))
	primary := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})
	data := appendAfterEOI(t, primary, gpsTrailerJPEG(t))
	path := writeTempFile(t, "mpf-trailer.jpg", data)
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if !ReadMetadata(got).Empty() {
		t.Fatalf("metadata left: %+v", ReadMetadata(got))
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("trailer Exif survived the file rewrite")
	}
	if n := jpegLength(got); n != len(got) {
		t.Fatalf("file still has a trailer: jpegLength=%d len=%d", n, len(got))
	}
}

func TestStripJPEGMetadata_TrailerOnlyRewritesTheFile(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	before := appendAfterEOI(t, plain, gpsTrailerJPEG(t))
	path := writeTempFile(t, "trailer-only.jpg", before)

	if err := StripJPEGMetadata(storage.NewFileURI(path)); err != nil {
		t.Fatal(err)
	}

	got := mustRead(t, path)
	if bytes.Equal(got, before) {
		t.Fatal("trailer-only strip must rewrite the file")
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("trailer Exif survived")
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("header-clean primary must be unchanged aside from dropping the trailer")
	}
}

func TestStripJPEGMetadata_Orientation6DropsTrailer(t *testing.T) {
	primary := halfRedHalfBlueJPEG(t, 20, 10, 6)
	path := writeTempFile(t, "rotated-trailer.jpg", appendAfterEOI(t, primary, gpsTrailerJPEG(t)))
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("re-encode path left trailer Exif")
	}
	if n := jpegLength(got); n != len(got) {
		t.Fatalf("re-encode path left a trailer: jpegLength=%d len=%d", n, len(got))
	}
}
```

Add a case to `TestCanStripJPEGMetadata`:

```go
		{"GPS JPEG after EOI is removable", appendAfterEOI(t, plainBuf.Bytes(), gpsTrailerJPEG(t)), true},
```

`t` is not in scope in the table if the table is built inside the test
function — it is (`TestCanStripJPEGMetadata(t *testing.T)`). Building
the slice inside the function after `plainBuf` is filled is required so
`appendAfterEOI(t, ...)` can run.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/imaging/ -run 'TestStripJPEGMetadata_DropsGPSTrailer|TestStripJPEGMetadata_TrailerOnly|TestStripJPEGMetadata_Orientation6DropsTrailer|TestCanStripJPEGMetadata' -v`

Expected: FAIL until Task 1 is merged into this tree. If Task 1 is
already in the working tree, the unit path may already pass and only
godoc remains — then these tests should PASS at first run. If they PASS
immediately after Task 1, that is success, not a broken TDD cycle:
Task 2’s job is the exported API and the rewrite/mtime contract.

- [ ] **Step 3: Update `StripJPEGMetadata` godoc** in `save.go`

Extend the first paragraph so it mentions trailers, e.g. after the
`(Exif, XMP, IPTC, COM, MPF)` list:

```
Bytes after the primary EOI (a concatenated second JPEG, motion-photo
video) are discarded. CanStripJPEGMetadata is true when those bytes are
the only thing left to remove.
```

Do not change control flow unless a test failed because
`CanStripJPEGMetadata` / the noop guard still ignore trailers (they
must not, once Task 1 landed).

- [ ] **Step 4: Run tests**

Run: `go test -count=1 ./internal/imaging/ -run 'TestStripJPEGMetadata|TestCanStripJPEGMetadata' -v`

Expected: PASS, including
`TestStripJPEGMetadata_NoRemovableSegmentsIsNoop`.

Then: `gofmt -l .`, `go vet ./...`, `go build ./...`.

Parent: `go test -race ./...` from the repo root.

- [ ] **Step 5: Suggested commit message** (do not commit)

```
imaging: rewrite JPEGs whose only leftover PII is a post-EOI trailer
```

---

### Task 3: Manuals, architecture map, todos

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md`
- Modify: `todos.md`

**Interfaces:** none.

- [ ] **Step 1: English manual**

In `internal/ui/help/manual.md`, replace the bullet

```
- Some camera and phone files that append a second JPEG after the primary
  image (multi-picture / motion photos) may keep that copy and its tags.
```

with:

```
- A second JPEG or motion-photo video appended after the main image is
  discarded, so tags hidden in that extra copy are removed too. The still
  stays; the extra frame or video does not.
```

- [ ] **Step 2: German manual**

In `internal/ui/help/manual_de.md`, replace

```
- Manche Kamera- und Handydateien hängen ein zweites JPEG hinter das
  Hauptbild (Multi-Picture / Motion Photos); diese Kopie und ihre Tags
  können erhalten bleiben.
```

with:

```
- Ein zweites JPEG oder Motion-Photo-Video hinter dem Hauptbild wird
  verworfen, damit auch Tags in dieser Kopie verschwinden. Das Standbild
  bleibt; das Extra-Bild oder Video nicht.
```

- [ ] **Step 3: `ARCHITECTURE.md`**

On the `jpegexif.go` row, after the sentence about `stripJPEGSegments` /
`keepOnStrip`, add that the SOS copy stops at the primary EOI
(`jpegLength` in `raw.go`) so MPF extra pictures and motion-photo
trailers do not survive a strip.

On the `save.go` row, in the `StripJPEGMetadata` sentence, add that a
trailer after EOI counts as removable and is dropped on both the
lossless and the orientation 2–8 paths.

- [ ] **Step 4: `todos.md`**

Move “Truncate MPF / motion-photo trailers when stripping JPEG metadata”
(and its paragraph) from TODO to Done. Leave the `warmDone` item under
TODO.

- [ ] **Step 5: Verify**

Run: `gofmt -l .` (empty). No Go changes expected.
`go test -count=1 ./internal/ui/help/ -count=1` if that package has tests
for embedded manuals; otherwise skip.

- [ ] **Step 6: Suggested commit message** (do not commit)

```
docs: strip JPEG trailers; update manuals and architecture map
```

---

## Parent review checklist (after every task)

1. Diff matches this task only (no `warmDone`, no MPF IFD parser, no new
   deps).
2. Tests named in the task were run; output is in the implementer report.
3. `gofmt -l .` clean, `go vet ./...`, `go build ./...` clean.
4. No `git commit`.
5. Fix anything Important/Critical yourself or with a fix subagent
   (`go-expert` · `claude-sonnet-5-thinking-high`) before Task N+1.

After Task 3: dispatch the final whole-branch reviewer on
`claude-opus-5-thinking-high`, then run `go test -race ./...` from the
repository root before declaring the feature done.

---

## Self-review

1. **Spec coverage:** Truncate at primary EOI — Task 1. Trailer-only
   button/rewrite — Tasks 1–2. Orientation 2–8 trailer drop — Task 2
   test. Manual caveat — Task 3. Fixtures — synthetic concat in Task 1.
   “Which trailers are safe” — none, table row 1.
2. **Placeholders:** none. Helpers are `appendAfterEOI` / `gpsTrailerJPEG`.
3. **Type consistency:** `jpegLength([]byte) int` already in `raw.go`.
   `stripJPEGSegments` / `jpegHasRemovableMetadata` signatures unchanged.
4. **Compile continuity:** Task 1 is the only production-code change.
   Task 2 is tests + godoc. `go build ./...` stays green between tasks.
5. **Out of scope:** `warmDone`, sharing the header walker, rewriting
   MPF offsets to keep a second still, HEIC Live Photos, new UI strings.
