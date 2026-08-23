# Remove Metadata from the EXIF Window — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task before dispatching the next. Do not
> start Task N+1 until Task N is reviewed and fixed. **Do not run
> `git commit`.**

**Goal:** The EXIF window (`E` / “Show EXIF data”) gains a **Remove Metadata**
button that, after a keyboard-driven confirmation, strips identifying tags
from the JPEG on disk and refreshes the panel, info-overlay link, and file
size.

**Architecture:** Keep the pixel bitstream when it is safe (Exif Orientation
1 or absent): walk JPEG markers and drop removable COM/APPn segments, then
write via the existing temp-file-then-rename path. When Orientation is 2–8,
decode with the existing `DecodeLoaded` orientation bake and re-encode at
`jpegSaveQuality` with no metadata, so the photo does not go sideways.
`internal/imaging` owns the write; `internal/ui/exifwin` owns the button and
the confirmation (same `ChoicePanel` + `dialog.NewCustomWithoutButtons`
shape as Favorites, parented on the EXIF window — **not** a main-window
`ChoiceCard`). The viewer implements a new three-method `exifwin.Host`.

**Tech Stack:** Go 1.26.7, stdlib `image/jpeg`, existing `jpegexif.go`
marker walk, Fyne v2 dialogs + `widgets.ChoicePanel`, `go test -race`.
No new module dependencies.

**Spec:** `todos.md` TODO
“Feature: Button in the exit window: Remove Metadata from file”
(the EXIF window — `internal/ui/exifwin`). Confirmation required.

**Precedent:** `jpegMetadataSegments` / `writeEncoded` /
`encodeJPEGForSave` (item 6). Favorites `showConfirm` for a focused
two-choice dialog on a secondary window. Deletion’s consumer-side `Host`.
`saveRotation` for cache eviction after an in-place file rewrite (do **not**
call `ShowImage` — that would reset zoom).

---

## Open questions (locked 2026-08-23)

Florian’s answers, 2026-08-23. Implementers treat these as spec.

| # | Decision |
|---|----------|
| 1 | EXIF window (`E` / Show EXIF data), not an app-exit dialog. |
| 2 | Privacy tags: Exif + XMP + IPTC + COM + MPF. Keep JFIF APP0, Adobe APP14, ICC. |
| 3 | JPEG only. **Hide** the button for every other format (not a greyed-out control). |
| 4 | Sideways photos: re-encode once (quality 95) so they stay upright. |
| 5 | Pending `R` rotation: ignore it. |
| 6 | Keep ICC. |
| 7 | Do not migrate `warmDone` in this plan. |

---

## Dispatch order and models

Parent: this session. One implementer at a time. After each task: parent
reviews the diff, fixes if needed, then dispatches the next.

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | Lossless JPEG segment strip + “has removable metadata?” | `go-expert` · `composer-2.5-fast` (complete tests in this plan) | `generalPurpose` · `gpt-5.6-sol-medium` |
| 2 | Exported `StripJPEGMetadata`, orientation bake, atomic write | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `claude-sonnet-5-thinking-high` |
| 3 | `exifwin.Host`, `New(app, Host)`, stub + viewer wiring (no button) | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `gpt-5.6-sol-medium` |
| 4 | Confirm dialog + Remove Metadata button + call strip | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `claude-sonnet-5-thinking-high` |
| 5 | Viewer `AfterMetadataRemoved` behavior + UI integration tests | `go-expert` · `claude-sonnet-5-thinking-high` | `generalPurpose` · `claude-sonnet-5-thinking-high` |
| 6 | Translations, manuals, `ARCHITECTURE.md`, `todos.md` | `generalPurpose` · `composer-2.5-fast` | `generalPurpose` · `gpt-5.6-sol-medium` |
| Final | Whole-branch review after Task 6 | — | `generalPurpose` · `claude-opus-5-thinking-high` |

Use `go-expert` for Tasks 1–5 so package-map and Fyne/test conventions stay
in that agent’s system prompt. Task 2 is the only imaging task with a
lossy fallback — do not downgrade it to the cheap model.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the EXIF-window /
  imaging-strip story changes (Task 6).
- Every user-visible string is `lang.L("English text")` with that exact
  key in every `translations/*.json` bundle (`en.json` identity map,
  `de.json` translation). `main_test.go` enforces locale parity.
- Viewer-independent packages (`internal/imaging`) return errors; they do
  not call `fyne.LogError`. UI-boundary failures use `fyne.LogError` plus
  a toast, matching `saveRotation`.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do not extract a shared `walkJPEGSegments` used by `jpegMetadata`.
  Duplicate the marker loop in the new strip helper, same as item 6.
- Do not migrate `exifwin.warmDone` onto `internal/completion.Signal`.
- Do not import `internal/ui/favorites` from `exifwin`. Copy the small
  confirm helper into `exifwin/confirm.go`.
- `exifwin.Host.DisplayedFile` must **not** be named `CurrentFile`:
  `viewer.CurrentFile` already returns `(fyne.URI, int, bool)` for
  `deletion.Host`. A type cannot satisfy both with one method name.
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 2 and Task 5.

---

## File map

| File | Role |
|------|------|
| `internal/imaging/jpegexif.go` | Unexported `keepOnStrip`, `jpegHasRemovableMetadata`, `stripJPEGSegments` |
| `internal/imaging/jpegexif_test.go` | Tests for the walker |
| `internal/imaging/save.go` | `writeFile` extracted from `writeEncoded`; exported `StripJPEGMetadata` |
| `internal/imaging/save_test.go` | `StripJPEGMetadata` file tests (GPS gone; orientation 6 stays upright) |
| `internal/ui/exifwin/exifwin.go` | `Host`, button, `canStrip`, layout |
| `internal/ui/exifwin/confirm.go` | Two-choice confirmation on the EXIF window |
| `internal/ui/exifwin/exifwin_test.go` | `stubHost`; button/confirm/strip tests |
| `internal/ui/exifwin/confirm_test.go` | Keyboard: focus starts on Cancel; Esc cancels without closing the EXIF window |
| `internal/ui/viewer.go` | Exported `DisplayedFile`; `AfterMetadataRemoved` |
| `internal/ui/features.go` | `exifwin.New(application, view)` |
| `internal/ui/exif_test.go` | Viewer-level: strip hides EXIF link, updates overlay size, hides the button |
| `translations/en.json`, `translations/de.json` | New strings |
| `internal/ui/help/manual.md`, `manual_de.md` | Document the button, JPEG-only, orientation re-encode |
| `ARCHITECTURE.md` | `exifwin/` and `jpegexif.go` / `save.go` rows |
| `todos.md` | Move the feature under Done |

---

## Assumptions (locked for implementers)

1. **JPEG destinations only**, identified by SOI `FF D8` after
   `filepath.EvalSymlinks` (same symlink rule as `SaveRotated`). Wrong
   magic → error, no write.
2. **Removable segments:** COM (`0xFE`) and APPn (`0xE0`–`0xEF`) **except**
   APP0 (`0xE0`, JFIF/JFXX) and APP14 (`0xEE`, Adobe). ICC APP2 is kept.
   MPF APP2 is dropped (invalid once frames change; not portable).
3. **Orientation 1 (or missing tag, which `jpegEXIFOrientation` already
   reports as 1):** `stripJPEGSegments` then atomic write. Pixel bytes
   after SOS are copied verbatim.
4. **Orientation 2–8:** `DecodeLoaded(ctx, data, 0)` (already applies
   `ApplyOrientation`), then `encodeJPEGForSave` into `writeFile`. No
   segment splice. One generation of JPEG loss, quality 95.
5. **Idempotent success:** a JPEG with nothing removable returns `nil`
   without rewriting the file.
6. **UI confirmation** is required before any write. Capture the URI when
   the confirm opens; if `Refresh` sees a different current file, hide the
   dialog. The Remove Metadata button is **shown only when
   `CanStripJPEGMetadata` is true** (a JPEG with something to strip).
   Hide it for PNG/HEIC/RAW/plain JPEGs and after a successful strip.
   Do not leave a disabled button on screen.
7. **After a successful strip:** evict `imgCache` for that URI, `Stat` the
   new `currentFileSize`, set `currentHasEXIF = false`, sync the info
   overlay, `exif.Refresh()`, toast `Metadata removed`. Do not reset
   zoom or rotation. Do not `ShowImage`.
8. **Errors:** `fyne.LogError` + toast
   `could not remove metadata from %q: %v`. File unchanged
   (`writeFile` only renames after a full write).

---

### Task 1: Lossless JPEG metadata-segment strip

**Files:**
- Modify: `internal/imaging/jpegexif.go`
- Modify: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: existing `errNotJPEG`; the same marker walk as
  `jpegMetadataSegments` (duplicate the loop, do not share a walker).
- Produces:
  - `func keepOnStrip(marker byte, payload []byte) bool`
  - `func jpegHasRemovableMetadata(data []byte) bool`
  - `func stripJPEGSegments(data []byte) ([]byte, error)`

- [ ] **Step 1: Write the failing tests** in `jpegexif_test.go`

Reuse `wrapAsAPP1`, `wrapAPP2`, `buildExifSegment`, `buildGPSExifTIFF`,
`gpsFields`, `spliceMetadataIntoJPEG`, and `markedImage` (same package).
Do **not** invent a new GPS JPEG builder and do **not** use `gpsJPEG`
(that helper is SOI+APP1 only, no SOS — `jpeg.Decode` would fail).

Add a tiny Adobe APP14 helper next to `wrapAPP2`:

```go
func wrapAPP14() []byte {
	// Marker + length 14 + "Adobe" + version/flags/transform (typical APP14).
	return []byte{0xFF, 0xEE, 0x00, 0x0E, 'A', 'd', 'o', 'b', 'e', 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00}
}
```

```go
func TestKeepOnStrip(t *testing.T) {
	if !keepOnStrip(0xE0, []byte("JFIF")) {
		t.Fatal("keep APP0")
	}
	if !keepOnStrip(0xEE, []byte("Adobe")) {
		t.Fatal("keep APP14")
	}
	if keepOnStrip(0xFE, []byte("hi")) {
		t.Fatal("drop COM")
	}
	if keepOnStrip(0xE1, []byte("Exif\x00\x00")) {
		t.Fatal("drop Exif APP1")
	}
	icc := []byte("ICC_PROFILE\x00\x01\x01x")
	if !keepOnStrip(0xE2, icc) {
		t.Fatal("keep ICC APP2")
	}
	if keepOnStrip(0xE2, []byte("MPF\x00x")) {
		t.Fatal("drop MPF APP2")
	}
	if keepOnStrip(0xED, []byte("Photoshop")) {
		t.Fatal("drop IPTC/APP13")
	}
}

func TestJPEGHasRemovableMetadata(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	if jpegHasRemovableMetadata(plain) {
		t.Fatal("stdlib JPEG is only JFIF APP0; nothing removable")
	}
	exif := wrapAsAPP1(buildExifSegment(t, 1, false))
	withExif := append([]byte{}, plain[:2]...)
	withExif = append(withExif, exif...)
	withExif = append(withExif, plain[2:]...)
	if !jpegHasRemovableMetadata(withExif) {
		t.Fatal("Exif APP1 is removable")
	}
	if jpegHasRemovableMetadata([]byte("\x89PNG")) {
		t.Fatal("non-JPEG is not removable metadata")
	}
}

func eiffelGPS() gpsFields {
	return gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	}
}

func TestStripJPEGSegments(t *testing.T) {
	t.Run("drops Exif XMP COM MPF IPTC; keeps APP0 APP14 ICC and the scan", func(t *testing.T) {
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		exif := wrapAsAPP1(buildExifSegment(t, 1, false))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))
		iptc := []byte{0xFF, 0xED, 0x00, 0x0C, 'P', 'h', 'o', 't', 'o', 's', 'h', 'o', 'p', 0x00}
		adobe := wrapAPP14()

		data := spliceMetadataIntoJPEG(t, markedImage(4, 4), [][]byte{
			com, xmp, exif, icc, mpf, iptc, adobe,
		})

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("still has removable segments")
		}
		if !bytes.Contains(got, []byte("ICC_PROFILE")) {
			t.Fatal("lost ICC")
		}
		if !bytes.Contains(got, []byte("Adobe")) {
			t.Fatal("lost APP14")
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) || bytes.Contains(got, []byte("xap/1.0")) || bytes.Contains(got, []byte("MPF\x00")) {
			t.Fatal("left identifying or MPF segments")
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped file must still decode: %v", err)
		}
	})

	t.Run("orientation-1 GPS JPEG is bit-identical in the scan after strip", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
		data := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})
		if !ReadMetadata(data).HasGPS {
			t.Fatal("setup: want GPS")
		}
		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatal(err)
		}
		if !ReadMetadata(got).Empty() {
			t.Fatalf("ReadMetadata after strip = %+v, want empty", ReadMetadata(got))
		}
		sos := bytes.Index(data, []byte{0xFF, 0xDA})
		if sos < 0 {
			t.Fatal("setup: no SOS")
		}
		gotSOS := bytes.Index(got, []byte{0xFF, 0xDA})
		if gotSOS < 0 {
			t.Fatal("stripped file lost SOS")
		}
		if !bytes.Equal(data[sos:], got[gotSOS:]) {
			t.Fatal("lossless strip must copy the entropy-coded scan verbatim")
		}
	})

	t.Run("non-JPEG is errNotJPEG", func(t *testing.T) {
		_, err := stripJPEGSegments([]byte("\x89PNG"))
		if !errors.Is(err, errNotJPEG) {
			t.Fatalf("err = %v, want errNotJPEG", err)
		}
	})
}
```

`injectJPEGMetadata` inserts segments immediately after SOI, before the
encoder’s JFIF APP0 — `spliceMetadataIntoJPEG` is the right fixture.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/imaging/ -run 'TestKeepOnStrip|TestJPEGHasRemovableMetadata|TestStripJPEGSegments' -v`

Expected: FAIL — `keepOnStrip` / `stripJPEGSegments` undefined.

- [ ] **Step 3: Implement**

In `jpegexif.go`, next to `skipSegment`:

```go
// keepOnStrip reports whether a COM/APPn segment should survive
// stripJPEGSegments. APP0 (JFIF/JFXX) and APP14 (Adobe color transform)
// stay; ICC APP2 stays (appearance). Everything else removable — Exif,
// XMP, IPTC, COM, MPF — is dropped.
func keepOnStrip(marker byte, payload []byte) bool {
	if marker == 0xE0 { // APP0
		return true
	}
	if marker == 0xEE { // APP14 Adobe
		return true
	}
	if marker == 0xE2 && len(payload) >= 12 && string(payload[:12]) == "ICC_PROFILE\x00" {
		return true
	}
	return false
}

// jpegHasRemovableMetadata reports whether stripJPEGSegments would drop
// at least one COM/APPn segment. Non-JPEG data is false.
func jpegHasRemovableMetadata(data []byte) bool {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return false
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

// stripJPEGSegments returns a copy of a JPEG with removable metadata
// segments (see keepOnStrip) omitted. DQT/DHT/SOF/DRI and the entropy-
// coded scan are copied verbatim. data that is not a JPEG yields
// errNotJPEG. A JPEG with nothing removable returns a copy of data.
func stripJPEGSegments(data []byte) ([]byte, error) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, errNotJPEG
	}

	out := make([]byte, 0, len(data))
	out = append(out, 0xFF, 0xD8)
	pos := 2

	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return nil, errNotJPEG
		}
		marker := data[pos+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			out = append(out, data[pos:pos+2]...)
			pos += 2
			continue
		}
		if marker == 0xDA {
			out = append(out, data[pos:]...)
			return out, nil
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			return nil, errNotJPEG
		}
		segEnd := pos + 2 + segLen
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !keepOnStrip(marker, data[pos+4:segEnd]) {
				pos = segEnd
				continue
			}
		}
		out = append(out, data[pos:segEnd]...)
		pos = segEnd
	}
	return nil, errNotJPEG
}
```

If the walk never hits SOS, return `errNotJPEG` rather than a header-only
buffer.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/imaging/ -run 'TestKeepOnStrip|TestJPEGHasRemovableMetadata|TestStripJPEGSegments|TestJPEGMetadataSegments|TestInjectJPEGMetadata|TestIsExifAPP1' -v`

Expected: PASS. Existing preserve-metadata tests still pass (this task
must not change `skipSegment` / inject / normalize).

Then: `gofmt -w internal/imaging/jpegexif.go internal/imaging/jpegexif_test.go` and the Global Constraints verification commands.

- [ ] **Step 5: Suggested commit** (do not run git commit)

```
imaging: lossless JPEG strip of identifying COM/APPn segments
```

---

### Task 2: `StripJPEGMetadata` — orientation bake + atomic write

**Files:**
- Modify: `internal/imaging/save.go`
- Modify: `internal/imaging/save_test.go`
- Test helpers: `writeTempFile` / `halfRedHalfBlueJPEG` in
  `loader_test.go` (same package — call them)

**Interfaces:**
- Consumes: `stripJPEGSegments`, `jpegHasRemovableMetadata`,
  `jpegEXIFOrientation`, `DecodeLoaded`, `encodeJPEGForSave`,
  `encoders` / `writeEncoded` internals.
- Produces:
  - `func writeFile(path string, perm os.FileMode, write func(io.Writer) error) error`
  - `func StripJPEGMetadata(u fyne.URI) error`
  - `func CanStripJPEGMetadata(data []byte) bool` — exported wrapper:
    `jpegHasRemovableMetadata(data) || jpegEXIFOrientation(data) != 1`
    (and false when `data` is not a JPEG). Task 4 uses this; do not export
    `jpegHasRemovableMetadata`.

- [ ] **Step 1: Refactor `writeEncoded` to `writeFile`**

Replace the body of `writeEncoded` with a call to a new unexported
`writeFile` that takes `write func(io.Writer) error` instead of
`(encode, img)`. Behavior and comments stay the same: temp in the
destination directory, `Chmod(perm)`, write, `Sync`, `Close`, `Rename`,
`defer os.Remove`. `writeEncoded` becomes:

```go
func writeEncoded(path string, perm os.FileMode, encode func(io.Writer, image.Image) error, img image.Image) error {
	return writeFile(path, perm, func(w io.Writer) error { return encode(w, img) })
}
```

Existing `SaveRotated` / `Export` tests must keep passing with no
assertion changes.

- [ ] **Step 2: Write failing `StripJPEGMetadata` tests** in `save_test.go`

```go
func TestStripJPEGMetadata_RemovesGPSWithoutTouchingPixelsWhenOrientation1(t *testing.T) {
	// Build a temp JPEG: jpeg.Encode + GPS Exif APP1 after SOI (orientation default 1).
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	})...))
	path := writeTempFile(t, "gps.jpg", spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif}))
	u := storage.NewFileURI(path)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ReadMetadata(before).HasGPS {
		t.Fatal("setup: want GPS")
	}

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ReadMetadata(after).Empty() {
		t.Fatalf("metadata left: %+v", ReadMetadata(after))
	}
	pre, err := jpeg.Decode(bytes.NewReader(before))
	if err != nil {
		t.Fatal(err)
	}
	post, err := jpeg.Decode(bytes.NewReader(after))
	if err != nil {
		t.Fatal(err)
	}
	if pre.Bounds() != post.Bounds() {
		t.Fatalf("bounds %v vs %v", pre.Bounds(), post.Bounds())
	}
}

func TestStripJPEGMetadata_Orientation6StaysUpright(t *testing.T) {
	path := writeTempFile(t, "rotated.jpg", halfRedHalfBlueJPEG(t, 20, 10, 6))
	u := storage.NewFileURI(path)

	loadedBefore, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	b := loadedBefore.Frames[0].Bounds()
	if b.Dx() != 10 || b.Dy() != 20 {
		t.Fatalf("setup size %dx%d, want 10x20", b.Dx(), b.Dy())
	}

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	if !ReadMetadata(mustRead(t, path)).Empty() {
		t.Fatal("want no Exif after strip")
	}
	if jpegEXIFOrientation(mustRead(t, path)) != 1 {
		t.Fatal("stripped file must not carry orientation 6")
	}

	loadedAfter, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	img := loadedAfter.Frames[0]
	if img.Bounds() != b {
		t.Fatalf("size %v, want %v (upright)", img.Bounds(), b)
	}
	r, _, b2, _ := img.At(5, 5).RGBA()
	if r < b2 {
		t.Errorf("top: want red after strip+reload")
	}
	r, _, b2, _ = img.At(5, 15).RGBA()
	if b2 < r {
		t.Errorf("bottom: want blue after strip+reload")
	}
}

func TestStripJPEGMetadata_NotJPEG(t *testing.T) {
	path := writeTempFile(t, "x.png", []byte("\x89PNG\r\n"))
	err := StripJPEGMetadata(storage.NewFileURI(path))
	if !errors.Is(err, errNotJPEG) {
		t.Fatalf("err = %v, want errNotJPEG", err)
	}
}

func TestStripJPEGMetadata_NoRemovableSegmentsIsNoop(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	path := writeTempFile(t, "plain.jpg", buf.Bytes())
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()

	if err := StripJPEGMetadata(storage.NewFileURI(path)); err != nil {
		t.Fatal(err)
	}

	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(mtime) {
		t.Fatal("noop strip must not rewrite the file")
	}
}
```

`spliceMetadataIntoJPEG` / `wrapAsAPP1` / `buildGPSExifTIFF` / `markedImage`
are already in this package. `mustRead` is `os.ReadFile` + `t.Fatal`. Add a
`TestCanStripJPEGMetadata` covering: stdlib JPEG → false; GPS splice → true;
`halfRedHalfBlueJPEG(..., 6)` → true; PNG magic → false.

Also pin **symlink target** if `SaveRotated` already has that test shape:
`StripJPEGMetadata` must `EvalSymlinks` and write the target, not replace
the link. Copy the existing SaveRotated symlink test and change the call.

Also pin **permissions**: after strip, `info.Mode().Perm()` equals the
original (same as `writeEncoded`).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test -count=1 ./internal/imaging/ -run TestStripJPEGMetadata -v`

Expected: FAIL — `StripJPEGMetadata` undefined.

- [ ] **Step 4: Implement `StripJPEGMetadata`**

```go
// StripJPEGMetadata removes identifying metadata from the JPEG at u
// (Exif, XMP, IPTC, COM, MPF) in place, keeping JFIF APP0, Adobe APP14,
// and ICC. When the file's Exif Orientation is 2–8, the pixels are
// decoded with that orientation applied and re-encoded at jpegSaveQuality
// so the photo does not appear sideways after the tag is gone.
//
// A non-JPEG returns errNotJPEG and does not write. A JPEG with nothing
// removable returns nil without rewriting the file. The write is the
// same temp-file-then-rename as SaveRotated, through a symlink to the
// target, preserving permission bits.
func StripJPEGMetadata(u fyne.URI) error {
	path, err := filepath.EvalSymlinks(u.Path())
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return errNotJPEG
	}
	if !jpegHasRemovableMetadata(data) && jpegEXIFOrientation(data) == 1 {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	orient := jpegEXIFOrientation(data)
	if orient == 1 {
		stripped, err := stripJPEGSegments(data)
		if err != nil {
			return err
		}
		return writeFile(path, info.Mode().Perm(), func(w io.Writer) error {
			_, err := w.Write(stripped)
			return err
		})
	}

	loaded, err := DecodeLoaded(context.Background(), data, 0)
	if err != nil {
		return err
	}
	if len(loaded.Frames) == 0 {
		return errNotJPEG
	}
	return writeFile(path, info.Mode().Perm(), func(w io.Writer) error {
		return encodeJPEGForSave(w, loaded.Frames[0])
	})
}

// CanStripJPEGMetadata reports whether StripJPEGMetadata would rewrite
// data. False for non-JPEG. True when there is a removable COM/APPn
// segment or when Exif Orientation is 2–8 (those files must be re-encoded
// so they stay upright).
func CanStripJPEGMetadata(data []byte) bool {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return false
	}
	return jpegHasRemovableMetadata(data) || jpegEXIFOrientation(data) != 1
}
```

Add `"context"` to `save.go` imports.

**Orientation-6 vs `jpegHasRemovableMetadata`:** orientation 6 implies an
Exif APP1, so removable is true. The `&& jpegEXIFOrientation(data) == 1`
guard on the noop path is so a hypothetical orientation-only file still
bakes. Keep it.

Do **not** go through `encodeJPEGPreservingMetadata`.

- [ ] **Step 5: Run tests**

Run: `go test -count=1 ./internal/imaging/ -run 'TestStripJPEGMetadata|TestSaveRotated|TestExport' -v`

Expected: PASS.

Then Global Constraints verification. Parent additionally runs
`go test -race ./internal/imaging/`.

- [ ] **Step 6: Suggested commit**

```
imaging: StripJPEGMetadata for in-place JPEG privacy strip
```

---

### Task 3: `exifwin.Host` and `New(app, Host)`

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go`
- Modify: `internal/ui/exifwin/exifwin_test.go`
- Modify: `internal/ui/exifwin/tiles_test.go` if it calls `New`
- Modify: `internal/ui/viewer.go` (`DisplayedFile`, `AfterMetadataRemoved`)
- Modify: `internal/ui/features.go`

**Interfaces:**
- Consumes: `viewer.displayedFile`, `viewer.ShowToast`, `viewer.imgCache`,
  `viewer.currentFileSize`, `viewer.currentHasEXIF`,
  `syncInfoOverlayVisibility`, `updateInfoOverlay`.
- Produces:

```go
// Host is what the EXIF window needs from the app once it can mutate the
// file on disk. DisplayedFile is the file actually on screen (not merely
// selected during a failed load) — the same fact New used to take as a
// func. It is not named CurrentFile: viewer.CurrentFile already returns
// an index for deletion.Host.
type Host interface {
	DisplayedFile() (fyne.URI, bool)
	AfterMetadataRemoved()
	ShowToast(msg string)
}
```

`New(application fyne.App, host Host) *Window`

`Window` stores `host Host` instead of `current func()`. Every
`w.current()` becomes `w.host.DisplayedFile()`.

- [ ] **Step 1: `stubHost` + rewrite `testApp` / `gpsApp`**

```go
type stubHost struct {
	current func() (fyne.URI, bool)
	toasts  []string
	after   int
}

func (s *stubHost) DisplayedFile() (fyne.URI, bool) {
	if s.current == nil {
		return nil, false
	}
	return s.current()
}
func (s *stubHost) AfterMetadataRemoved() { s.after++ }
func (s *stubHost) ShowToast(msg string) {
	s.toasts = append(s.toasts, msg)
}
```

Change `testApp` / `gpsApp` to return `(fyne.App, *stubHost)`. Replace
every `New(app, current)` with `New(app, host)`. Tests that reassign
`shown := uri` keep a closure on the stub’s `current` field — same
pattern as `TestRefresh_LocationSectionIsHiddenForAnUnreadableFile`.

- [ ] **Step 2: Viewer methods**

```go
// DisplayedFile is the file decoded and on screen, ok=false when the
// drop zone is showing. Satisfies exifwin.Host; narrower than
// CurrentFile, which still reports a selected index during a failed load.
func (v *viewer) DisplayedFile() (fyne.URI, bool) {
	return v.displayedFile()
}

// AfterMetadataRemoved is exifwin.Host: the JPEG on disk just lost its
// identifying tags. Evict the decode cache so a later visit cannot
// revive HasEXIF, refresh the on-screen file size, hide the info-overlay
// EXIF link, and sync the open EXIF panel.
func (v *viewer) AfterMetadataRemoved() {
	u, ok := v.displayedFile()
	if !ok {
		return
	}
	v.imgCache.Remove(u.String())
	if info, err := os.Stat(u.Path()); err == nil {
		v.currentFileSize = info.Size()
	}
	v.currentHasEXIF = false
	v.syncInfoOverlayVisibility()
	v.updateInfoOverlay()
	v.exif.Refresh()
}
```

Add `"os"` to `viewer.go` imports if needed.

`features.go`:

```go
view.exif = exifwin.New(application, view)
```

- [ ] **Step 3: Run tests**

Run: `go test -count=1 ./internal/ui/exifwin/ ./internal/ui/ -run 'TestFormatExifMetadata|TestRestoreGeometry|TestShowExifWindow|TestExifLink' -v`

Expected: PASS. No new user-visible behavior yet. `AfterMetadataRemoved`
is unused by the window until Task 4 — that is OK.

If `tiles_test.go` constructs `New`, update it the same way.

Then Global Constraints verification.

- [ ] **Step 4: Suggested commit**

```
exifwin: take a Host so the panel can report a disk rewrite
```

---

### Task 4: Confirmation + Remove Metadata button

**Files:**
- Create: `internal/ui/exifwin/confirm.go`
- Create: `internal/ui/exifwin/confirm_test.go`
- Modify: `internal/ui/exifwin/exifwin.go`
- Modify: `internal/ui/exifwin/exifwin_test.go`
- Modify: `translations/en.json`
- Modify: `translations/de.json`

**Interfaces:**
- Consumes: `imaging.StripJPEGMetadata`, `imaging.ReadAndProbe`,
  `jpegHasRemovableMetadata` — **wait:** `jpegHasRemovableMetadata` is
  unexported in `imaging`. Do **not** export it for the UI.

  Enablement: in `Refresh`, after a successful `ReadAndProbe`, set
  `w.canStrip = imaging.CanStripJPEGMetadata(data)` (Task 2). On the
  unreadable-file path, `canStrip = false`.

- Produces: button, confirm, `requestStrip` / `performStrip`.

**User-visible strings** (English keys, add to both JSON bundles in this
task so `main_test.go` locale parity stays green):

| Key | `de.json` |
|-----|-----------|
| `Remove Metadata` | `Metadaten entfernen` |
| `Remove Metadata?` | `Metadaten entfernen?` |
| `Remove camera, date, GPS, and other tags from %q? This cannot be undone.` | `Kamera-, Datums-, GPS- und andere Tags aus %q entfernen? Das kann nicht rückgängig gemacht werden.` |
| `Metadata removed` | `Metadaten entfernt` |
| `could not remove metadata from %q: %v` | `Metadaten von %q konnten nicht entfernt werden: %v` |

Reuse existing `Cancel` key.

- [ ] **Step 1: `confirm.go`** — copy the Favorites shape, not the package

Mirror `internal/ui/favorites/confirm.go`:

- `cancelChoice = 0`, `confirmChoice = 1`
- unexported `confirmation` struct: `title`, `message`, `action`,
  `importance`, `onConfirm`, `onCancel`, `onClosed`
- `func (w *Window) showConfirm(c confirmation) dialog.Dialog`
- Parent window: `w.win.Window()` — if nil, no-op (window closed).
- `dialog.NewCustomWithoutButtons` on **that** window.
- `widgets.NewChoicePanel(nil, Cancel, danger action)`.
- `panel.SetOnDismiss(func() { confirm.Hide() })`
- `panel.SetOnCancel(c.onCancel)`
- `w.win.Window().Canvas().Focus(panel)` **after** `Show` — load-bearing:
  `widgets.Singleton` registers `Canvas().SetOnTypedKey` Escape →
  `win.Close()`. If `Focused()` is nil, Escape closes the EXIF window
  behind the prompt. A focused `ChoicePanel` swallows Escape.

Store `w.confirm dialog.Dialog`. A second `showConfirm` hides the previous
one first (Favorites’ superseded-dialog guard).

```go
func (w *Window) hideConfirm() {
	if w.confirm != nil {
		w.confirm.Hide()
		w.confirm = nil
	}
	w.pending = nil
}
```

Nil-check `onClosed` like Favorites (`SetOnClosed` panics on nil).

- [ ] **Step 2: Confirm tests** in `confirm_test.go`

Port `TestShowConfirmGivesTheKeyboardToItsPanelStartingOnCancel` and
`TestShowConfirmEscapeRunsOnCancelAfterTheDialogCloses` from
`internal/ui/favorites/confirm_test.go`.

**Extra test (this package’s own trap):** with the EXIF window open and
the confirm showing, Escape must **not** set `w.Open()` to false. After
Escape, `w.Open()` is still true and overlays are empty.

- [ ] **Step 3: Button + layout**

Fields on `Window`:

```go
strip    *widget.Button
canStrip bool
pending  fyne.URI // file the open confirm is about; nil when none
confirm  dialog.Dialog
```

In `Show`’s `build` func, after `buildLocation` / `Refresh`:

```go
w.strip = widget.NewButton(lang.L("Remove Metadata"), w.requestStrip)
w.strip.Importance = widget.DangerImportance
w.syncStripVisible()

return container.NewBorder(
    container.NewPadded(w.text),
    container.NewPadded(w.strip),
    nil, nil,
    w.location,
)
```

`onClosed` nils `strip`, `confirm`, `pending` as well as the existing
map widgets.

`Refresh` already reads the file. After `ReadMetadata` / error paths:

```go
w.canStrip = err == nil && imaging.CanStripJPEGMetadata(data)
w.syncStripVisible()
if w.pending != nil {
    u, ok := w.host.DisplayedFile()
    if !ok || u == nil || u.String() != w.pending.String() {
        w.hideConfirm()
    }
}
```

On the unreadable-file path, `canStrip = false`.

```go
func (w *Window) syncStripVisible() {
    if w.strip == nil {
        return
    }
    if w.canStrip {
        w.strip.Show()
        return
    }
    w.strip.Hide()
}
```

```go
func (w *Window) requestStrip() {
    if !w.canStrip {
        return
    }
    u, ok := w.host.DisplayedFile()
    if !ok {
        return
    }
    w.pending = u
    w.showConfirm(confirmation{
        title:      lang.L("Remove Metadata?"),
        message:    fmt.Sprintf(lang.L("Remove camera, date, GPS, and other tags from %q? This cannot be undone."), u.Name()),
        action:     lang.L("Remove Metadata"),
        importance: widget.DangerImportance,
        onConfirm:  func() { w.performStrip(u) },
        onCancel:   func() { w.pending = nil },
        onClosed:   func() { w.pending = nil },
    })
}

func (w *Window) performStrip(u fyne.URI) {
    w.pending = nil
    if err := imaging.StripJPEGMetadata(u); err != nil {
        fyne.LogError("failed to remove metadata", err)
        w.host.ShowToast(fmt.Sprintf(lang.L("could not remove metadata from %q: %v"), u.Name(), err))
        return
    }
    w.host.AfterMetadataRemoved()
    w.host.ShowToast(lang.L("Metadata removed"))
}
```

`AfterMetadataRemoved` already calls `exif.Refresh()`, so `performStrip`
does not need a second Refresh. If you Refresh twice it is harmless.

**Do not** toast from both `performStrip` and `AfterMetadataRemoved`.
Toast lives in `performStrip` only (success and failure). Host does not
toast.

Export for tests:

```go
func (w *Window) StripButton() *widget.Button { return w.strip }
```

- [ ] **Step 4: Window tests**

```go
func TestStripButton_HiddenForAJPEGWithNoMetadata(t *testing.T) {
    app, host := testApp(t) // plain TempJPEGURI
    w := New(app, host)
    w.Show()
    t.Cleanup(func() { w.Window().Close() })
    if w.StripButton() == nil || w.StripButton().Visible() {
        t.Fatal("want the button hidden when nothing is removable")
    }
}

func TestStripButton_ShownForAGPSJPEG(t *testing.T) {
    app, host := gpsApp(t)
    w := New(app, host)
    w.Show()
    t.Cleanup(func() { w.Window().Close() })
    if !w.StripButton().Visible() {
        t.Fatal("want the button shown for GPS JPEG")
    }
}

func TestRequestStrip_CancelLeavesTheFileUnchanged(t *testing.T) {
    app, host := gpsApp(t)
    w := New(app, host)
    w.Show()
    t.Cleanup(func() { w.Window().Close() })

    u, _ := host.DisplayedFile()
    before, _ := os.ReadFile(u.Path())

    w.StripButton().OnTapped()
    // Escape on the focused panel
    w.Window().Canvas().Focused().(*widgets.ChoicePanel).TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

    after, _ := os.ReadFile(u.Path())
    if !bytes.Equal(before, after) {
        t.Fatal("Cancel must not write")
    }
    if host.after != 0 {
        t.Fatal("AfterMetadataRemoved must not run")
    }
}

func TestRequestStrip_ConfirmRemovesGPSAndCallsHost(t *testing.T) {
    app, host := gpsApp(t)
    w := New(app, host)
    w.Show()
    t.Cleanup(func() { w.Window().Close() })

    w.StripButton().OnTapped()
    panel := w.Window().Canvas().Focused().(*widgets.ChoicePanel)
    panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
    panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

    u, _ := host.DisplayedFile()
    data, _ := os.ReadFile(u.Path())
    if !imaging.ReadMetadata(data).Empty() {
        t.Fatal("want metadata gone")
    }
    if host.after != 1 {
        t.Fatalf("AfterMetadataRemoved calls = %d, want 1", host.after)
    }
    if w.Location().Visible() {
        t.Fatal("map section should hide after strip+Refresh")
    }
    if got := w.Text().Text; got != lang.L("No EXIF metadata found in this file.") && got != "No EXIF metadata found in this file." {
        t.Fatalf("text = %q", got)
    }
    if w.StripButton().Visible() {
        t.Fatal("button should hide after a successful strip")
    }
}
```

Use `lang.L` only if the test app loaded translations; existing exifwin
tests assert the English literal `"No EXIF metadata found in this file."`
because `test.NewApp()` uses identity English. Keep asserting that
literal.

Copy `typeKey` from `internal/ui/favorites/manage_test.go` (same
package-local helper: send `TypedKey` to `win.Canvas().Focused()`).
Do not import `favorites`.

- [ ] **Step 5: Run tests**

Run: `go test -count=1 ./internal/ui/exifwin/ -v`

Expected: PASS.

Run: `go test -count=1 ./ -run 'TestTranslation'` (from repo root;
`main_test.go` locale parity).

Then Global Constraints verification.

- [ ] **Step 6: Suggested commit**

```
exifwin: Remove Metadata button with a confirmation dialog
```

---

### Task 5: Viewer integration tests

**Files:**
- Modify: `internal/ui/exif_test.go`
- Possibly: `internal/ui/harness_test.go` only if a new background
  goroutine appears — **it must not.** Strip is synchronous, like
  `saveRotation`. Do not add a `completion.Signal`. Do not touch `drain`.

**Interfaces:**
- Consumes: Task 3’s `AfterMetadataRemoved`, Task 4’s button.

- [ ] **Step 1: Write failing/passing integration tests** in `exif_test.go`

```go
func TestStripMetadata_HidesExifLinkAndShrinksReportedSize(t *testing.T) {
    v, _, _ := newTestUI(t)
    u := uitest.TempGPSJPEGURI(t, "gps.jpg", 40, 20, 48.858222, 2.2945)
    dropAndWait(t, v, u)

    beforeSize := v.currentFileSize
    v.toggleInfoOverlay()
    if !v.exifLink.Visible() {
        t.Fatal("setup: EXIF link shown")
    }

    v.exif.Show()
    v.exif.StripButton().OnTapped()
    panel := v.exif.Window().Canvas().Focused().(*widgets.ChoicePanel)
    panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
    panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

    settleToast(t, v)

    if v.currentHasEXIF {
        t.Fatal("currentHasEXIF still true")
    }
    if v.exifLink.Visible() {
        t.Fatal("EXIF link should hide")
    }
    if v.currentFileSize <= 0 || v.currentFileSize >= beforeSize {
        t.Fatalf("file size %d, want smaller than %d", v.currentFileSize, beforeSize)
    }
    if v.imgCache.Contains(u.String()) {
        t.Fatal("imgCache should have been evicted")
    }
}

func TestStripMetadata_ErrorToastsAndLeavesEXIF(t *testing.T) {
    // Optional: chmod 0444 the file after load, confirm strip, expect
    // error toast and currentHasEXIF still true. Skip on platforms where
    // the user is root and chmod does not make the write fail.
}
```

If GPS splice does not shrink the file enough to assert `>= beforeSize`
is wrong — GPS APP1 is a few hundred bytes; `currentFileSize` should
drop. If a platform keeps size equal, assert `ReadMetadata` empty on
disk instead of a strict size inequality, and keep the size update
`Stat` behavior.

`settleToast` already exists in `harness_test.go`.

- [ ] **Step 2: Run tests**

Run: `go test -count=1 ./internal/ui/ -run 'TestStripMetadata|TestShowExifWindow|TestExifLink' -v`

Expected: PASS.

Parent: `go test -race ./internal/ui/ ./internal/imaging/ ./internal/ui/exifwin/`

Then full Global Constraints verification.

- [ ] **Step 3: Suggested commit**

```
ui: refresh EXIF link and cache after metadata is removed
```

---

### Task 6: Docs, architecture, todos

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md`
- Modify: `todos.md`

No new translation keys (Task 4 added them).

- [ ] **Step 1: Manual (English)**

In section 6 (info overlay / EXIF window), after the paragraph about
files with no Exif data, add that the window has a **Remove Metadata**
button at the bottom. It asks for confirmation (Cancel selected by
default; `←`/`→` and `Return` / `Esc`, same keyboard rules as other
PicFetch confirmations). It rewrites the **original JPEG** in place.

State clearly:

- JPEG only. HEIC, RAW, PNG, WebP: the button is hidden.
- Removes camera, date, GPS, XMP, IPTC, and comments. Color profile
  (ICC) and the JPEG’s own color transform stay, so the picture should
  look the same.
- A photo shot sideways (Exif orientation 2–8) is re-saved once so it
  stays upright without the orientation tag; that is a normal JPEG
  re-encode (quality 95), not a lossless copy.
- A photo already upright is stripped without re-encoding the pixels.
- View-only rotation (`R`) is not written; use Save Changes first if
  that rotation should land on disk.
- Cannot be undone except from backups / Trash — this is not a Trash
  move.

In the keyboard/features summary (`EXIF data window` bullet) mention the
button.

No markdown tables. Unicode arrows only inside backticks. Menu paths use
ASCII `->`.

- [ ] **Step 2: Manual (German)**

Same facts, same tone as the surrounding EXIF section. Button name as
in `de.json` (`Metadaten entfernen`).

- [ ] **Step 3: `ARCHITECTURE.md`**

`exifwin/` row: mention the Remove Metadata control, the confirmation
on the panel’s own window (`ChoicePanel`, not main-window `ChoiceCard`),
and `Host` (`DisplayedFile` / `AfterMetadataRemoved` / `ShowToast`).

`jpegexif.go` row: add `stripJPEGSegments` / `keepOnStrip` as the inverse
of the preserve-metadata splice.

`save.go` row: add exported `StripJPEGMetadata` / `CanStripJPEGMetadata`.

- [ ] **Step 4: `todos.md`**

Move “Feature: Button in the exit window: Remove Metadata from file”
under Done, same prose style as item 6. Leave
`Migrate exifwin warmDone onto completion.Signal` as TODO.

- [ ] **Step 5: Run** `go test -count=1 ./internal/ui/help/`

Expected: PASS (`TestManualHasNoMarkdownTables`,
`TestManualUnicodeArrowsStayInCodeSpans`).

- [ ] **Step 6: Suggested commit**

```
docs: EXIF window can strip JPEG metadata in place
```

---

## Out of scope (do not implement)

- HEIC/AVIF/TIFF/RAW/PNG metadata rewrite or delete.
- Stripping ICC (color) or Adobe APP14.
- jpegtran-style lossless MCU rotate for orientation 2–8.
- Applying view-only `R` rotation as part of strip.
- Extracting Favorites `showConfirm` into `widgets`.
- Migrating `warmDone` to `completion.Signal`.
- Shared `walkJPEGSegments` for `jpegMetadata` / `jpegEXIFOrientation`.
- Regenerating or keeping an EXIF thumbnail.
- A File-menu item or shortcut (the todo is the EXIF window button).
- New dependencies.

---

## Parent review checklist (after every task)

1. Diff matches this task only (no drive-by `warmDone`, no extra exports).
2. Tests named in the task were run; output is in the implementer report.
3. `gofmt -l .` clean, `go vet` / `go build` clean.
4. No `git commit`.
5. Fix anything Important/Critical yourself or with a fix subagent
   (`go-expert` · `claude-sonnet-5-thinking-high`) before Task N+1.

After Task 6: dispatch the final whole-branch reviewer on
`claude-opus-5-thinking-high` with `scripts/review-package` against the
merge-base, then run `go test -race ./...` from the repository root
before declaring the feature done.

---

## Self-review

1. **Spec coverage:** Button on EXIF window — Task 4. Confirmation —
   Task 4. Strip EXIF (and XMP/IPTC/COM/MPF) — Tasks 1–2. JPEG-only —
   Task 2 + hidden button Task 4. Cache/info overlay — Tasks 3 and 5.
   Docs — Task 6. `warmDone` — out of scope.
2. **Placeholders:** none. Fixtures are `spliceMetadataIntoJPEG`,
   `buildGPSExifTIFF` / `gpsFields`, `halfRedHalfBlueJPEG`, `markedImage`.
   `mustRead` is a three-line test helper. `typeKey` is copied from
   `favorites/manage_test.go`.
3. **Type consistency:** `Host.DisplayedFile() (fyne.URI, bool)` —
   not `CurrentFile`. `StripJPEGMetadata(fyne.URI) error`.
   `CanStripJPEGMetadata([]byte) bool`. Toast strings only in
   `performStrip`.
4. **Compile continuity:** Task 3 updates `features.go` in the same
   change as `New`’s signature so `go build ./...` never breaks between
   tasks.
