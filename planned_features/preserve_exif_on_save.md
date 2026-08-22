# Preserve JPEG metadata when saving (and when exporting JPEG→JPEG)

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task before dispatching the next. Do not
> start Task N+1 until Task N is reviewed and fixed. **Do not run
> `git commit`.**

**Goal:** File → Save Changes (`imaging.SaveRotated`) re-encodes a
user-rotated JPEG without stripping the metadata the original file
carried. File → Export image does the same when both the source and the
destination are JPEG.

**Architecture:** Do not add an EXIF library. After `jpeg.Encode`, splice
the original JPEG's metadata segments (COM + APPn) back in immediately
after SOI. Skip the original JFIF APP0 (the encoder writes a new one) and
skip MPF APP2 (its offsets point into the old file). On the Exif APP1,
rewrite IFD0 Orientation to 1 and unlink IFD1 (stale thumbnail) in-place
so a reload does not double-rotate; maker notes, GPS, XMP, ICC, and IPTC
stay byte-identical. Non-JPEG destinations, wallpaper PNGs, and HEIC/WebP
→ JPEG (no JPEG segments to copy) stay a plain pixel encode.

**Tech Stack:** Go 1.26.7, stdlib `image/jpeg`, existing
`internal/imaging` TIFF/APP1 parsers, Fyne test driver, `go test -race`.
No new module dependencies.

**Spec:** `todos.md` ACTIVE item
"Bug: When saving a rotated image the EXIF data is not being preserved",
plus Florian's rulings (2026-08-22): keep **all** original JPEG metadata,
copy it on **JPEG→JPEG export** as well, **unlink IFD1**. The current
code already documents the gap: `internal/imaging/save.go` says SaveRotated
is "a plain pixel round-trip, not a metadata-preserving edit", and
`internal/ui/help/manual.md` says "EXIF metadata is not carried over".

**Precedent:** `internal/imaging/exif.go` already walks JPEG APP1 and TIFF
IFDs (read-only). `halfRedHalfBlueJPEG` in `loader_test.go` already
splices `wrapAsAPP1(buildExifSegment(...))` after SOI of a stdlib JPEG.
This plan is that splice for every metadata segment, plus two in-place
IFD patches on Exif. `writeEncoded`'s temp-file-then-rename stays the
only writer.

---

## Global Constraints

Copied from `AGENTS.md`; every task's requirements implicitly include
these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the save/EXIF story
  changes (Task 7).
- Every user-visible string is `lang.L("English text")` with the same key
  in every `translations/*.json` bundle. This plan adds no new UI strings.
- Viewer-independent packages (`internal/imaging`) return errors; they do
  not call `fyne.LogError`.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do not retrofit `jpegMetadata` / `jpegEXIFOrientation` onto a shared
  walker in this plan (YAGNI). `jpegexif.go` may duplicate the APP1 walk.
- Do not implement the "Remove Metadata from file" EXIF-window button.
  Keep the new helpers unexported so that later work can reuse them.
- Verification per task, from the repository root, after the
  task's own focused tests pass:
  `gofmt -l .` (must print nothing), `go vet ./...`, `go build ./...`,
  then the focused tests named in the task. The parent runs
  `go test -race ./...` after Task 4 and Task 6.

---

## Assumptions (locked for implementers)

Florian's answers, 2026-08-22.

1. **JPEG destinations only.** Copy metadata when the file being written
   is `.jpg` / `.jpeg` / `.jpe` / `.jfif`. PNG, GIF, BMP, TIFF, AVIF,
   wallpaper PNG, and WebP/HEIC (decode-only) stay a plain re-encode.
2. **Save Changes and JPEG→JPEG Export.** `SaveRotated` always copies
   from the file it overwrites. `Export` copies only when the *source*
   URI is a JPEG (magic `FF D8`) *and* the destination extension is
   JPEG. Exporting HEIC/WebP/PNG → JPEG does **not** transcode ISOBMFF
   EXIF into APP1 in this plan. Exporting JPEG → PNG writes a PNG with
   no metadata.
3. **All original JPEG metadata segments.** Copy every COM (`0xFE`) and
   APPn (`0xE0`–`0xEF`) between SOI and SOS, **except**:
   - **APP0** (JFIF/JFXX): `jpeg.Encode` writes a fresh APP0; a second
     one is wrong.
   - **APP2 whose payload starts with `MPF\0`:** Multi-Picture offsets
     would point into the discarded original scan.
   Keep Exif APP1, XMP APP1, ICC APP2 (`ICC_PROFILE\0`), Photoshop/IPTC
   APP13, other APPn, and COM, in original order.
4. **Bake pixels, Exif orientation tag = 1.** Saved/exported pixels
   already include decode-time `ApplyOrientation` plus any view-only
   `RotateSteps`. Written Orientation is 1 when tag 0x0112 exists as a
   SHORT (missing still means 1). Never encode the user rotation as a
   new orientation tag.
5. **Drop IFD1 in-place.** Zero IFD0's next-IFD pointer so a thumbnail
   that still shows the unrotated photo is no longer linked. Do not
   rewrite offsets, do not shrink the segment, do not regenerate a
   thumbnail. Orphaned IFD1 bytes may remain inside APP1.
6. **Do not rewrite other Exif tags.** PixelXDimension / PixelYDimension /
   ImageWidth / ImageLength / MakerNote stay as in the original blob.
   SOF in the new JPEG is the authority for pixel size.
7. **Failure mode.** If orig is not a JPEG, has no metadata segments, or
   an Exif APP1 is too malformed to patch, still write the encoded
   pixels (today's behavior). `SaveRotated` already fails if it cannot
   read the file it is about to replace. `Export` must **not** fail the
   write just because the source cannot be read — export the pixels
   without metadata instead.
8. **Exported API:** `Export` gains a source URI argument (may be nil).
   Extract/inject/normalize/encode-preserving helpers stay unexported.

---

## Approaches considered

| | Approach | Why not |
| --- | --- | --- |
| **A (this plan)** | Copy COM+APPn blobs; patch Exif Orientation=1; unlink IFD1 | — |
| B | Add `dsoprea/go-exif` (or similar) and rebuild the IFD | New dependency, MakerNote offsets often break, more code than the bug needs |
| C | Lossless JPEG transpose (`jpegtran`-style) | Does not match today's re-encode-the-displayed-frame save; the pixels `SaveRotated` receives are already RGBA |

---

## File map

| File | Role |
| --- | --- |
| Create `internal/imaging/jpegexif.go` | Unexported extract / inject / normalize / encode-preserving-metadata |
| Create `internal/imaging/jpegexif_test.go` | Unit tests for those helpers |
| Modify `internal/imaging/save.go` | JPEG `SaveRotated` and JPEG `Export` use `encodeJPEGPreservingMetadata`. `Export` takes `src fyne.URI`. `encoders` stay on `encodeJPEGForSave` for non-JPEG |
| Modify `internal/imaging/save_test.go` | SaveRotated/Export keep metadata; orientation 6 does not double-rotate; existing Export tests pass `nil` src |
| Modify `internal/ui/export.go` | `imaging.Export(dest, img, src)` |
| Modify `internal/ui/wallpaper.go` | `imaging.Export(dest, img, nil)` — wallpaper is PNG |
| Modify `internal/ui/save_test.go` | Viewer `saveRotation` on a GPS JPEG still has metadata on disk |
| Modify `internal/ui/export_test.go` | JPEG→JPEG export keeps GPS; JPEG→PNG does not invent JPEG segments |
| Modify `ARCHITECTURE.md` | `save.go` / rotation / export index rows |
| Modify `internal/ui/help/manual.md` and `manual_de.md` | Stop saying metadata is discarded |
| Modify `todos.md` | Move the bug into Done |

`internal/ui/save.go` (`saveRotation`) does not change: it already
passes `v.img.Image` and the current URI. The imaging layer re-reads
the file.

---

## Subagent dispatch

Run **sequentially**. Tasks 1–5 share `jpegexif.go` / `save.go`;
parallel dispatch will conflict.

After each task: parent reviews the diff, runs the task's tests, and
fixes anything the subagent got wrong before launching the next.

| Task | Subagent | Model | Why |
| --- | --- | --- | --- |
| 1 extract | `go-expert` | `claude-sonnet-5-thinking-high` | Segment keep/skip rules (APP0, MPF vs ICC) are easy to get wrong |
| 2 inject | `go-expert` | `cursor-grok-4.6-xhigh` | Symmetric byte splice; same file as Task 1 |
| 3 normalize | `go-expert` | `claude-sonnet-5-thinking-high` | In-place IFD writes; easy to corrupt endianness or the next-IFD pointer |
| 4 SaveRotated | `go-expert` | `cursor-grok-4.6-xhigh` | Wiring + imaging tests; depends on 1–3 |
| 5 Export API | `go-expert` | `cursor-grok-4.6-xhigh` | Signature change + JPEG→JPEG; wallpaper passes nil |
| 6 UI tests | `go-expert` | `cursor-grok-4.6-xhigh` | `newTestViewer` / `dropAndWait` / chooser stubs |
| 7 docs | `generalPurpose` | `composer-2.5-fast` | Manual / ARCHITECTURE / todos only |

If Task 1, 3, or 4 fails parent review twice, redispatches that **one**
task with `claude-opus-5-thinking-high` (`go-expert`). Do not use Opus
for Tasks 2, 5, 6, or 7.

Each implementer prompt must include: this plan path, the task heading,
**Interfaces** from earlier tasks (copy them; the subagent has no chat
history), Global Constraints, and "do not git commit".

---

## Task 1: Extract JPEG metadata segments

**Files:**
- Create: `internal/imaging/jpegexif.go`
- Test: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: existing test helpers in the same package —
  `buildExifSegment`, `wrapAsAPP1` (`exif_test.go`),
  `halfRedHalfBlueJPEG` (`loader_test.go`).
- Produces:

```go
// jpegMetadataSegments returns a copy of every COM and APPn segment
// between SOI and SOS, in file order, excluding APP0 (JFIF/JFXX) and
// APP2 segments whose payload starts with "MPF\x00". Each slice includes
// the 0xFF marker, the 2-byte length, and the payload. data that is not
// a JPEG yields nil.
func jpegMetadataSegments(data []byte) [][]byte

// isExifAPP1 reports whether seg is a full APP1 whose payload starts
// with "Exif\x00\x00".
func isExifAPP1(seg []byte) bool
```

Walk JPEG markers the same way `jpegEXIFOrientation` does (`exif.go`
~31–74): start at offset 2, skip standalone markers (`0xD8`, `0x01`,
`0xD0`–`0xD9`), stop at SOS (`0xDA`) or a truncated length. On COM
(`0xFE`) or APPn (`0xE0`–`0xEF`), copy `data[pos:pos+2+segLen]` unless
the skip rules apply.

Skip APP0: `marker == 0xE0`.

Skip MPF: `marker == 0xE2` and `len(payload) >= 4` and
`string(payload[:4]) == "MPF\x00"`. ICC APP2 (`ICC_PROFILE\x00`) is
kept.

Each returned slice is a copy so later normalize can mutate it.

- [ ] **Step 1: Write the failing tests** in `jpegexif_test.go`

```go
package imaging

import (
	"bytes"
	"image/jpeg"
	"testing"
)

func TestIsExifAPP1(t *testing.T) {
	exif := wrapAsAPP1(buildExifSegment(t, 6, false))
	if !isExifAPP1(exif) {
		t.Fatal("isExifAPP1: want true for an Exif APP1")
	}
	xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta/>"))
	if isExifAPP1(xmp) {
		t.Fatal("isExifAPP1: want false for XMP")
	}
	if isExifAPP1(nil) || isExifAPP1([]byte{0xFF, 0xE1}) {
		t.Fatal("isExifAPP1: want false for truncated input")
	}
}

func TestJPEGMetadataSegments(t *testing.T) {
	t.Run("keeps Exif APP1, XMP APP1, ICC APP2, COM; skips JFIF APP0 and MPF", func(t *testing.T) {
		jfif := []byte{0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 1, 1, 0, 0, 1, 0, 1, 0, 0}
		com := []byte{0xFF, 0xFE, 0x00, 0x07, 'h', 'i', 0x00}
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		exif := wrapAsAPP1(buildExifSegment(t, 6, false))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))

		var data []byte
		data = append(data, 0xFF, 0xD8)
		data = append(data, jfif...)
		data = append(data, com...)
		data = append(data, xmp...)
		data = append(data, exif...)
		data = append(data, icc...)
		data = append(data, mpf...)
		data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0) // SOS: walk must stop

		got := jpegMetadataSegments(data)
		if len(got) != 4 {
			t.Fatalf("got %d segments, want 4 (COM, XMP, Exif, ICC)", len(got))
		}
		if !bytes.Equal(got[0], com) || !bytes.Equal(got[1], xmp) || !bytes.Equal(got[2], exif) || !bytes.Equal(got[3], icc) {
			t.Fatalf("segments = %x, want COM, XMP, Exif, ICC in that order", got)
		}
		got[2][4] = 'X'
		if !isExifAPP1(exif) {
			t.Fatal("jpegMetadataSegments must return copies")
		}
	})

	t.Run("finds Exif after jpeg.Encode's JFIF, the way real files look", func(t *testing.T) {
		data := halfRedHalfBlueJPEG(t, 8, 8, 6)
		var exif []byte
		for _, s := range jpegMetadataSegments(data) {
			if isExifAPP1(s) {
				exif = s
			}
		}
		if exif == nil {
			t.Fatal("expected an Exif APP1 in halfRedHalfBlueJPEG")
		}
		if parseExifOrientation(exif[4:]) != 6 {
			t.Errorf("orientation = %d, want 6", parseExifOrientation(exif[4:]))
		}
	})

	t.Run("nil for a JPEG with no metadata and for non-JPEG", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(buf.Bytes()); len(segs) != 0 {
			t.Errorf("stdlib JPEG metadata = %d segments, want 0 (JFIF APP0 skipped)", len(segs))
		}
		if jpegMetadataSegments([]byte("\x89PNG")) != nil {
			t.Error("want nil for PNG magic")
		}
		if jpegMetadataSegments(nil) != nil {
			t.Error("want nil for nil")
		}
	})
}

func wrapAPP2(payload []byte) []byte {
	length := len(payload) + 2
	return append([]byte{0xFF, 0xE2, byte(length >> 8), byte(length)}, payload...)
}
```

`parseExifOrientation` expects the APP1 *payload* (`Exif\x00\x00` +
TIFF), which is `seg[4:]` after skipping `FF E1 len_hi len_lo`.

SOS in the synthetic file only needs to stop the walker; it does not
need to be a valid scan.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments' ./internal/imaging/`

Expected: FAIL, functions undefined.

- [ ] **Step 3: Write the minimal implementation** in `jpegexif.go`

Do not call `jpegEXIFOrientation`. Do not change `exif.go`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments' ./internal/imaging/`

Expected: PASS.

- [ ] **Step 5: Format and focused vet**

```
gofmt -w internal/imaging/jpegexif.go internal/imaging/jpegexif_test.go
go vet ./internal/imaging/
go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments' ./internal/imaging/
```

- [ ] **Step 6: Suggested commit** (do not run git commit)

```
imaging: extract JPEG COM/APPn metadata segments, skipping JFIF and MPF
```

---

## Task 2: Inject metadata segments after SOI

**Files:**
- Modify: `internal/imaging/jpegexif.go`
- Modify: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: `jpegMetadataSegments`, `isExifAPP1` from Task 1.
- Produces:

```go
// injectJPEGMetadata returns a JPEG that is encoded with segs inserted
// immediately after the SOI marker, in slice order. encoded must start
// with FF D8. Empty segs returns a copy of encoded. Each seg must be a
// COM or APPn segment starting with 0xFF.
func injectJPEGMetadata(encoded []byte, segs [][]byte) ([]byte, error)
```

Define a small error value in `jpegexif.go`:

```go
var errNotJPEG = errors.New("not a JPEG")
```

Return `errNotJPEG` when `encoded` is shorter than 2 bytes or does not
start with `0xFF 0xD8`, or when a non-empty `seg` is shorter than 2
bytes, does not start with `0xFF`, or has a marker other than `0xFE` or
`0xE0`–`0xEF`.

- [ ] **Step 1: Write the failing tests** (append to `jpegexif_test.go`)

```go
func TestInjectJPEGMetadata(t *testing.T) {
	t.Run("inserts immediately after SOI, before JFIF APP0", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		encoded := buf.Bytes()
		com := []byte{0xFF, 0xFE, 0x00, 0x07, 'h', 'i', 0x00}
		exif := wrapAsAPP1(buildExifSegment(t, 8, false))

		out, err := injectJPEGMetadata(encoded, [][]byte{com, exif})
		if err != nil {
			t.Fatalf("injectJPEGMetadata: %v", err)
		}
		if out[0] != 0xFF || out[1] != 0xD8 {
			t.Fatal("output lost SOI")
		}
		if !bytes.Equal(out[2:2+len(com)], com) {
			t.Fatal("COM was not placed immediately after SOI")
		}
		if !bytes.Equal(out[2+len(com):2+len(com)+len(exif)], exif) {
			t.Fatal("Exif APP1 was not placed after COM")
		}
		if !bytes.Equal(out[2+len(com)+len(exif):], encoded[2:]) {
			t.Fatal("bytes after SOI were not preserved")
		}

		got := jpegMetadataSegments(out)
		if len(got) != 2 || !bytes.Equal(got[0], com) || !isExifAPP1(got[1]) {
			t.Fatal("extract did not round-trip the injected segments")
		}
	})

	t.Run("empty segs returns a copy of encoded", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		encoded := buf.Bytes()
		out, err := injectJPEGMetadata(encoded, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, encoded) {
			t.Fatal("empty segs must preserve the encoded JPEG")
		}
		out[0] = 0
		if encoded[0] != 0xFF {
			t.Fatal("must not alias encoded")
		}
	})

	t.Run("rejects a non-JPEG encoded buffer", func(t *testing.T) {
		exif := wrapAsAPP1(buildExifSegment(t, 1, false))
		if _, err := injectJPEGMetadata([]byte("\x89PNG"), [][]byte{exif}); err == nil {
			t.Fatal("want error")
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run TestInjectJPEGMetadata ./internal/imaging/`

Expected: FAIL, `injectJPEGMetadata` undefined.

- [ ] **Step 3: Write the minimal implementation**

```go
func injectJPEGMetadata(encoded []byte, segs [][]byte) ([]byte, error) {
	if len(encoded) < 2 || encoded[0] != 0xFF || encoded[1] != 0xD8 {
		return nil, errNotJPEG
	}
	extra := 0
	for _, s := range segs {
		if len(s) < 2 || s[0] != 0xFF {
			return nil, errNotJPEG
		}
		m := s[1]
		if m != 0xFE && (m < 0xE0 || m > 0xEF) {
			return nil, errNotJPEG
		}
		extra += len(s)
	}
	out := make([]byte, 0, 2+extra+len(encoded)-2)
	out = append(out, 0xFF, 0xD8)
	for _, s := range segs {
		out = append(out, s...)
	}
	out = append(out, encoded[2:]...)
	return out, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments|TestInjectJPEGMetadata' ./internal/imaging/`

Expected: PASS.

- [ ] **Step 5: Format**

`gofmt -w internal/imaging/jpegexif.go internal/imaging/jpegexif_test.go`

- [ ] **Step 6: Suggested commit** (do not run git commit)

```
imaging: splice JPEG metadata segments back in immediately after SOI
```

---

## Task 3: Normalize saved Exif (Orientation=1, unlink IFD1)

**Files:**
- Modify: `internal/imaging/jpegexif.go`
- Modify: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: `isExifAPP1`, `parseExifOrientation`, `tiffOrder` (`raw.go`).
- Produces:

```go
// normalizeSavedExif returns a copy of app1 (a full FF E1 Exif segment)
// with IFD0 Orientation (tag 0x0112) set to 1 when that tag is present
// as a SHORT, and with IFD0's next-IFD pointer zeroed so a thumbnail
// IFD1 is no longer linked. If app1 is not a well-formed Exif APP1,
// it is returned copied and unchanged.
func normalizeSavedExif(app1 []byte) []byte
```

Patch rules (all in-place on the copy):

1. If `!isExifAPP1(app1)` or `len(app1) < 10`, return a copy unchanged.
2. TIFF starts at `app1[10:]`. Use `tiffOrder(tiff)` for endianness.
   IFD0 offset is `bo.Uint32(tiff[4:8])`.
3. **Orientation:** walk IFD0 entries (12 bytes each). For tag `0x0112`
   with type SHORT (3) and count 1, write `1` as a 16-bit value at
   `entryOffset+8` using `bo.PutUint16`. Leave the two padding bytes
   after the SHORT as they were. If the tag is absent or not a SHORT,
   do not add or rewrite it.
4. **IFD1:** at `ifd0Offset+2+numEntries*12`, if those 4 bytes fit in
   `tiff`, write `bo.PutUint32(..., 0)`. Do not compact. Do not follow
   the old pointer.

Do not use `walkIFD` for the write: the next-IFD pointer is not an IFD
entry. Write a dedicated unexported helper, e.g. `patchSavedTIFF(tiff []byte)`,
called on `out[10:]` after cloning `app1`. Bounds-check every offset. On
any failure to find IFD0, return the clone unchanged (do not panic).

Helper for tests (in `jpegexif_test.go` only): `buildExifWithThumbnailIFD`
— little-endian TIFF, IFD0 with one Orientation=6 entry, next-IFD
pointing at IFD1, IFD1 with one dummy tag (e.g. ImageWidth 0x0100 = 16)
and next 0. Keep the layout explicit with named offsets like
`buildGPSExifTIFF`.

Indexing: APP1 = `FF E1` + 2-byte length + payload. Payload =
`"Exif\x00\x00"` + TIFF. TIFF = `app1[10:]`.

- [ ] **Step 1: Write the failing tests**

```go
func TestNormalizeSavedExif(t *testing.T) {
	t.Run("sets orientation 6 to 1 and leaves the rest of the payload intact", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifSegment(t, 6, false))
		got := normalizeSavedExif(app1)
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
		if parseExifOrientation(app1[4:]) != 6 {
			t.Fatal("normalizeSavedExif mutated the input segment")
		}
	})

	t.Run("big-endian orientation 8 becomes 1", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifSegment(t, 8, true))
		got := normalizeSavedExif(app1)
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
	})

	t.Run("zeros IFD0 next-IFD so IFD1 is unlinked", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifWithThumbnailIFD(t))
		tiff := app1[10:]
		le := binary.LittleEndian
		ifd0 := le.Uint32(tiff[4:8])
		num := le.Uint16(tiff[ifd0 : ifd0+2])
		nextOff := ifd0 + 2 + uint32(num)*12
		if le.Uint32(tiff[nextOff:nextOff+4]) == 0 {
			t.Fatal("fixture: IFD1 pointer should be non-zero before normalize")
		}

		got := normalizeSavedExif(app1)
		gtiff := got[10:]
		gnum := le.Uint16(gtiff[ifd0 : ifd0+2])
		gnext := ifd0 + 2 + uint32(gnum)*12
		if le.Uint32(gtiff[gnext:gnext+4]) != 0 {
			t.Errorf("next IFD = %d, want 0", le.Uint32(gtiff[gnext:gnext+4]))
		}
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
	})

	t.Run("XMP APP1 is returned copied, not rewritten as Exif", func(t *testing.T) {
		in := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		got := normalizeSavedExif(in)
		if !bytes.Equal(got, in) {
			t.Fatalf("got %x, want copy of XMP", got)
		}
		got[4] = 'x'
		if in[4] == 'x' {
			t.Fatal("must not alias the input")
		}
	})
}
```

Import `"encoding/binary"` in `jpegexif_test.go`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -run TestNormalizeSavedExif ./internal/imaging/`

Expected: FAIL, `normalizeSavedExif` undefined.

- [ ] **Step 3: Write the implementation**

- [ ] **Step 4: Run tests**

Run: `go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments|TestInjectJPEGMetadata|TestNormalizeSavedExif' ./internal/imaging/`

Expected: PASS.

- [ ] **Step 5: Format**

`gofmt -w internal/imaging/jpegexif.go internal/imaging/jpegexif_test.go`

- [ ] **Step 6: Suggested commit** (do not run git commit)

```
imaging: set saved JPEG Exif orientation to 1 and unlink a stale thumbnail IFD
```

---

## Task 4: Wire JPEG SaveRotated to preserve metadata

**Files:**
- Modify: `internal/imaging/jpegexif.go` (add `encodeJPEGPreservingMetadata`)
- Modify: `internal/imaging/save.go`
- Modify: `internal/imaging/save_test.go`
- Modify: `internal/imaging/jpegexif_test.go`

**Interfaces:**
- Consumes: `jpegMetadataSegments`, `isExifAPP1`, `injectJPEGMetadata`,
  `normalizeSavedExif`, `jpegSaveQuality`, `writeEncoded`.
- Produces:

```go
// encodeJPEGPreservingMetadata encodes img at jpegSaveQuality, then
// splices orig's metadata segments after SOI. Exif APP1 segments are
// passed through normalizeSavedExif first. A non-JPEG orig, or a JPEG
// with no metadata, encodes exactly as encodeJPEGForSave would.
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte) error
```

Body:

```go
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte) error {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegSaveQuality}); err != nil {
		return err
	}
	encoded := buf.Bytes()
	segs := jpegMetadataSegments(orig)
	if len(segs) == 0 {
		_, err := w.Write(encoded)
		return err
	}
	for i, s := range segs {
		if isExifAPP1(s) {
			segs[i] = normalizeSavedExif(s)
		}
	}
	out, err := injectJPEGMetadata(encoded, segs)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
```

`SaveRotated` in `save.go` (keep symlink resolution, `os.Stat`,
`writeEncoded`):

- After resolving `path` and looking up `encode`, if
  `strings.ToLower(ext)` is one of `.jpg`, `.jpeg`, `.jpe`, `.jfif`:
  `orig, err := os.ReadFile(path)`; on read error return it; then

```go
encode = func(w io.Writer, img image.Image) error {
	return encodeJPEGPreservingMetadata(w, img, orig)
}
```

- Do **not** change `Export` in this task (Task 5).
- The `encoders` table stays on `encodeJPEGForSave`.
- Update the `SaveRotated` doc comment: JPEG SaveRotated now copies the
  original metadata segments with Exif Orientation reset to 1; other
  formats still do not carry metadata.

- [ ] **Step 1: Write failing tests**

In `jpegexif_test.go`:

```go
func TestEncodeJPEGPreservingMetadata(t *testing.T) {
	t.Run("copies GPS, XMP, ICC, and COM from orig", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
			latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
			lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
		})...))
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		com := []byte{0xFF, 0xFE, 0x00, 0x07, 'h', 'i', 0x00}
		orig := spliceMetadataIntoJPEG(t, markedImage(4, 3), [][]byte{com, xmp, exif, icc})

		var out bytes.Buffer
		if err := encodeJPEGPreservingMetadata(&out, markedImage(3, 2), orig); err != nil {
			t.Fatal(err)
		}
		m := ReadMetadata(out.Bytes())
		if !m.HasGPS {
			t.Fatal("saved JPEG lost GPS")
		}
		if jpegEXIFOrientation(out.Bytes()) != 1 {
			t.Errorf("saved orientation = %d, want 1", jpegEXIFOrientation(out.Bytes()))
		}
		got := jpegMetadataSegments(out.Bytes())
		if len(got) != 4 {
			t.Fatalf("saved metadata segments = %d, want 4", len(got))
		}
		if !bytes.Equal(got[0], com) {
			t.Fatal("lost COM")
		}
		if !bytes.Equal(got[1], xmp) {
			t.Fatal("lost XMP")
		}
		if !bytes.Contains(got[3], []byte("ICC_PROFILE")) {
			t.Fatal("lost ICC")
		}
	})

	t.Run("orig without metadata encodes a JPEG without extra APPn", func(t *testing.T) {
		var origBuf bytes.Buffer
		if err := jpeg.Encode(&origBuf, markedImage(2, 2), nil); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := encodeJPEGPreservingMetadata(&out, markedImage(2, 2), origBuf.Bytes()); err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(out.Bytes()); len(segs) != 0 {
			t.Errorf("invented %d metadata segments", len(segs))
		}
	})
}

func spliceMetadataIntoJPEG(t *testing.T, img image.Image, segs [][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	out, err := injectJPEGMetadata(buf.Bytes(), segs)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
```

Use the same `gpsFields` literal `TestReadMetadata_GPS` already uses
(`exif_test.go`). Do not invent a new GPS builder.

In `save_test.go`, add a subtest on `TestSaveRotated`:

```go
t.Run("JPEG keeps Exif and does not double-apply orientation on reload", func(t *testing.T) {
	orig := halfRedHalfBlueJPEG(t, 20, 10, 6)
	path := writeTempFile(t, "rotated.jpg", orig)
	u := storage.NewFileURI(path)

	loaded, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("load original: %v", err)
	}
	oriented := loaded.Frames[0]
	if b := oriented.Bounds(); b.Dx() != 10 || b.Dy() != 20 {
		t.Fatalf("oriented bounds = %v, want 10x20", b)
	}

	if err := SaveRotated(u, oriented); err != nil {
		t.Fatalf("SaveRotated: %v", err)
	}

	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if jpegEXIFOrientation(saved) != 1 {
		t.Errorf("saved orientation tag = %d, want 1", jpegEXIFOrientation(saved))
	}

	reloaded, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	b := reloaded.Frames[0].Bounds()
	if b.Dx() != 10 || b.Dy() != 20 {
		t.Errorf("reloaded bounds = %v, want 10x20 (must not apply orientation 6 again)", b)
	}
})
```

Keep the existing PNG lossless pixel test unchanged.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestEncodeJPEGPreservingMetadata|TestSaveRotated' ./internal/imaging/`

Expected: FAIL on the new cases.

- [ ] **Step 3: Implement `encodeJPEGPreservingMetadata` and the
  `SaveRotated` JPEG branch**

Update `SaveRotated`'s doc comment in the same edit.

- [ ] **Step 4: Run imaging tests**

```
go test -run 'TestIsExifAPP1|TestJPEGMetadataSegments|TestInjectJPEGMetadata|TestNormalizeSavedExif|TestEncodeJPEGPreservingMetadata|TestSaveRotated|TestExport|TestCanEncode' ./internal/imaging/
```

Expected: PASS. `TestExport` still uses the old `Export(u, img)`
signature until Task 5 — do not change Export yet, so this command
still compiles.

- [ ] **Step 5: Format, vet, race on imaging**

```
gofmt -w internal/imaging/jpegexif.go internal/imaging/jpegexif_test.go internal/imaging/save.go internal/imaging/save_test.go
go vet ./internal/imaging/
go test -race ./internal/imaging/
```

Parent additionally runs `go test -race ./...` before Task 5.

- [ ] **Step 6: Suggested commit** (do not run git commit)

```
imaging: copy JPEG metadata onto SaveRotated output with orientation reset to 1
```

---

## Task 5: Export copies JPEG metadata when source and dest are JPEG

**Files:**
- Modify: `internal/imaging/save.go` (`Export` signature and JPEG branch)
- Modify: `internal/imaging/save_test.go` (every `Export(` call)
- Modify: `internal/ui/export.go` (`imaging.Export(dest, img, src)`)
- Modify: `internal/ui/wallpaper.go` (`imaging.Export(..., nil)`)

**Interfaces:**
- Consumes: `encodeJPEGPreservingMetadata` from Task 4,
  `jpegMetadataSegments`, `CanEncodeExt`.
- Produces:

```go
// Export writes img to dest, encoded in dest's format. src is the file
// the pixels came from and may be nil. When dest is JPEG and src is a
// readable JPEG, dest receives a normalized copy of src's metadata
// segments (same rules as SaveRotated). A read failure on src does not
// fail the export: pixels are written without metadata.
func Export(dest fyne.URI, img image.Image, src fyne.URI) error
```

Implementation sketch:

```go
func Export(dest fyne.URI, img image.Image, src fyne.URI) error {
	ext := dest.Extension()
	encode, ok := encoders[strings.ToLower(ext)]
	if !ok {
		return &UnsupportedSaveFormatError{ext: ext}
	}
	path := dest.Path()
	perm := os.FileMode(defaultExportPerm)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	if isJPEGExt(ext) && src != nil && src.Path() != "" {
		if orig, err := os.ReadFile(src.Path()); err == nil {
			encode = func(w io.Writer, img image.Image) error {
				return encodeJPEGPreservingMetadata(w, img, orig)
			}
		}
	}

	return writeEncoded(path, perm, encode, img)
}
```

Add an unexported `isJPEGExt(ext string) bool` in `save.go` (or
`jpegexif.go`) and use it from both `SaveRotated` and `Export` so the
four extensions are not listed twice.

Update `Export`'s doc comment. Update every existing `Export(u, img)`
call to `Export(u, img, nil)`: `save_test.go` (all subtests),
`wallpaper.go`. `export.go` passes the captured `src`.

`encodeJPEGPreservingMetadata` already no-ops when `orig` is not a JPEG
(e.g. a `.jpg` name wrapping PNG bytes, or a HEIC path whose magic is
not `FF D8`): `jpegMetadataSegments` returns nil.

- [ ] **Step 1: Write the failing tests** in `save_test.go`

Import `github.com/frathe/picfetch/internal/uitest` if that file does
not already. Add:

```go
func TestExport_JPEGSourceKeepsMetadataOnJPEGDest(t *testing.T) {
	srcPath := writeTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 8, 4, 48.858, 2.294))
	src := storage.NewFileURI(srcPath)

	t.Run("jpeg dest", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "copy.jpg")
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !ReadMetadata(got).HasGPS {
			t.Fatal("JPEG→JPEG export dropped GPS")
		}
	})

	t.Run("png dest stays a PNG without JPEG APPn", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "copy.png")
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) < 4 || string(got[:4]) != "\x89PNG" {
			t.Fatal("PNG export must still be a PNG")
		}
		if jpegMetadataSegments(got) != nil {
			t.Fatal("PNG export must not carry JPEG segments")
		}
	})

	t.Run("nil src still encodes", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "bare.jpg")
		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), nil); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(got); len(segs) != 0 {
			t.Errorf("nil src invented %d metadata segments", len(segs))
		}
	})
}
```

Update every existing `Export(storage.NewFileURI(...), markedImage(...))`
in `TestExport` to pass `nil` as the third argument so the package
compiles.

- [ ] **Step 2: Run tests to verify they fail** (compile error on
  `Export` arity, then the new cases)

- [ ] **Step 3: Change `Export`, `export.go`, `wallpaper.go`, and all
  test call sites**

- [ ] **Step 4: Run**

```
gofmt -w internal/imaging/save.go internal/imaging/save_test.go internal/ui/export.go internal/ui/wallpaper.go
go vet ./internal/imaging/ ./internal/ui/
go test -race -run 'TestExport|TestSaveRotated|TestEncodeJPEGPreservingMetadata' ./internal/imaging/
go test -race -run 'TestExport|TestCanExport|TestWallpaper|TestWriteWallpaper' ./internal/ui/
```

Use the actual wallpaper test names in `wallpaper_test.go` (`go test
-run Test ./internal/ui/wallpaper_test.go` is fine if names differ).
Expected: PASS, including existing export error/cancel tests.

- [ ] **Step 5: Suggested commit** (do not run git commit)

```
imaging: copy JPEG metadata on JPEG-to-JPEG export; Export takes a source URI
```

---

## Task 6: Viewer tests for Save Changes and Export

**Files:**
- Modify: `internal/ui/save_test.go`
- Modify: `internal/ui/export_test.go`

**Interfaces:**
- Consumes: `saveRotation`, `runExport` / chooser stubs, `dropAndWait`,
  `newTestViewer`, `settleToast`, `uitest.GPSJPEG`,
  `imaging.ReadMetadata`, `imaging.LoadImage`.
- Produces: two new tests; no production UI changes beyond Task 5.

- [ ] **Step 1: Write the failing tests**

`save_test.go`:

```go
func TestSaveRotation_PreservesJPEGExif(t *testing.T) {
	v := newTestViewer(t)
	data := uitest.GPSJPEG(t, 8, 4, 48.858, 2.294)
	path := uitest.WriteTempFile(t, "geo.jpg", data)
	u := storage.NewFileURI(path)
	dropAndWait(t, v, u)

	if !v.currentHasEXIF {
		t.Fatal("setup: GPSJPEG should set currentHasEXIF")
	}

	v.rotateBy(1) // 8x4 → 4x8
	v.saveRotation()
	settleToast(t, v)

	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !imaging.ReadMetadata(saved).HasGPS {
		t.Fatal("Save Changes dropped GPS Exif")
	}

	loaded, err := imaging.LoadImage(u, imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	if b := loaded.Frames[0].Bounds(); b.Dx() != 4 || b.Dy() != 8 {
		t.Errorf("saved bounds = %v, want 4x8", b)
	}
}
```

`export_test.go`: follow the existing pattern that stubs
`filepicker.ChooseSave` (see tests that already call `runExport`).
Export a GPS JPEG to a new `.jpg` and assert `ReadMetadata` still has
GPS. Export the same source to `.png` and assert PNG magic, not JPEG
APPn.

If `currentHasEXIF` is unexported, this test is in package `ui` and can
read it.

- [ ] **Step 2: Run**

```
go test -race -run 'TestSaveRotation_PreservesJPEGExif|TestSaveRotation|TestCanSaveRotation' ./internal/ui/
go test -race -run 'TestExport|TestCanExport|TestRunExport' ./internal/ui/
```

Match the real export test function names when writing the new one.
Expected: PASS.

- [ ] **Step 3: Suggested commit** (do not run git commit)

```
ui: assert Save Changes and JPEG export keep GPS Exif
```

---

## Task 7: Docs, architecture map, todos

**Files:**
- Modify: `ARCHITECTURE.md` (`internal/imaging` `save.go` row ~158, and
  the rotation / export index lines ~644–650)
- Modify: `internal/ui/help/manual.md` (Save Changes bullet ~531–536 and
  Export bullet ~537–546)
- Modify: `internal/ui/help/manual_de.md` (matching bullets ~595–613)
- Modify: `todos.md`

**Interfaces:** none. No new `lang.L` keys.

- [ ] **Step 1: Update the manuals**

English Save Changes: keep "replaces the original file, and re-encodes"
— that is still true. Replace "so EXIF metadata is not carried over"
with:

```
This replaces the original file and re-encodes it. For JPEG, PicFetch
copies the original metadata (EXIF, including camera/date/GPS, plus
XMP, ICC, and IPTC if present) into the new file and sets the
orientation tag to 1, because the pixels already include both the
camera's orientation and the rotation you just saved. A JPEG thumbnail
stored in EXIF is dropped so it cannot show the unrotated photo.
```

English Export: add that a JPEG exported from a JPEG keeps that same
metadata; a JPEG exported from another format, or a PNG export, has
none.

German: same facts, same tone as the surrounding bullets. No markdown
tables. Unicode arrows stay inside backticks only (`manual_test.go`).
Use ASCII `->` in menu paths.

- [ ] **Step 2: Update `ARCHITECTURE.md`**

The `save.go` row currently says the writer "does not preserve the
original file's Exif metadata". Change it to: JPEG `SaveRotated` and
JPEG←JPEG `Export` re-read the source, splice metadata segments
(Orientation=1, IFD1 unlinked, APP0/MPF skipped) after SOI via
`jpegexif.go`; other formats and wallpaper PNG remain a plain re-encode.
`Export` takes the source URI (nil from wallpaper).

- [ ] **Step 3: Update `todos.md`**

Move "Bug: When saving a rotated image the EXIF data is not being
preserved" under Done, in the same prose style as items 3–5. Mention
JPEG→JPEG export. Strip-metadata and `warmDone` stay TODO.

- [ ] **Step 4: Run manual tests**

Run: `go test ./internal/ui/help/`

Expected: PASS.

- [ ] **Step 5: Suggested commit** (do not run git commit)

```
docs: Save Changes and JPEG export keep JPEG metadata
```

---

## Out of scope (do not implement)

- "Remove Metadata from file" button on the EXIF window (`todos.md`).
- Transcoding HEIC/AVIF/WebP EXIF into a JPEG APP1 on export.
- PNG `eXIf` / TIFF IFD0 copy / AVIF EXIF write.
- Regenerating an EXIF thumbnail.
- Updating PixelXDimension / ImageWidth after a 90° save.
- Copying MPF APP2 or a second JFIF APP0.
- Extracting a shared `walkJPEGSegments` used by `jpegMetadata`.
- New dependencies.

---

## Self-review

1. **Spec coverage:** Preserve all JPEG metadata on rotated save —
   Tasks 1–4. JPEG→JPEG export — Task 5. Orientation not double-applied
   — Task 4 `halfRedHalfBlueJPEG`. GPS/XMP/ICC/COM survive — Task 4.
   UI — Task 6. Manual no longer lies — Task 7. Strip-metadata / HEIC
   transcode / MPF — explicitly out of scope.
2. **Placeholders:** none intended; `buildExifWithThumbnailIFD`,
   `wrapAPP2`, and `spliceMetadataIntoJPEG` are specified as test-local
   helpers.
3. **Type consistency:** `jpegMetadataSegments([]byte) [][]byte`;
   `isExifAPP1([]byte) bool`; `injectJPEGMetadata([]byte, [][]byte) ([]byte, error)`;
   `normalizeSavedExif([]byte) []byte`;
   `encodeJPEGPreservingMetadata(io.Writer, image.Image, []byte) error`;
   `Export(dest, img, src fyne.URI) error`. TIFF index: payload at
   `app1[4:]`, TIFF at `app1[10:]`.
