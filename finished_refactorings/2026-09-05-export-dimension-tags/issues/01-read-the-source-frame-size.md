# 01: Read a JPEG's stored frame size from its SOF

**What to build:** One unexported reader in `internal/imaging/jpegseg.go`:

```go
// jpegFrameSize is the pixel size recorded in data's own frame header - the
// size the file's Exif dimension tags describe. ok is false for data that is
// not a JPEG, or whose frame header cannot be read.
func jpegFrameSize(data []byte) (w, h int, ok bool)
```

Build it on the existing `walkJPEGSegments`, which already hands every
payload-bearing header marker to a callback and stops at SOS. Do not write a
second walker, and do not touch `jpegLength` (the entropy-scan walk) or
`stripJPEGSegments`.

A start-of-frame marker is `0xC0`-`0xCF` **except** `0xC4` (DHT), `0xC8`
(reserved JPG) and `0xCC` (DAC), which are not frame headers. The payload of
one is `precision(1), height(2), width(2), components(1), ...`, both sizes
big-endian - so height comes first, which is the easy thing to get backwards.
Stop at the first usable frame header. A payload shorter than 5 bytes, or a
zero width or height (a JPEG that defines its height later, via DNL), is
not usable and must report `ok == false` rather than a guess.

Nothing calls it yet; ticket 02 does. This ticket ends with the function and
its test.

**Blocked by:** None

**Status:** done

- [x] `jpegFrameSize` reads width and height from a baseline JPEG's SOF0
- [x] It reads them from a progressive JPEG's SOF2
- [x] Height and width are not transposed, proven with an asymmetric fixture
- [x] Non-JPEG data, a truncated header, and a zero dimension all report `ok == false`
- [x] It is built on `walkJPEGSegments` rather than a second hand-rolled walk
- [x] `go test ./internal/imaging/ -run TestJPEGFrameSize -v` passes
- [x] `go vet ./internal/imaging/` is clean and `gofmt -l internal/imaging/` is empty

## Comments

2026-09-06 - Delivered by a go-expert subagent, test-first: jpegFrameSize on top of walkJPEGSegments, 13 subtests. Verified independently - transposing height and width fails three subtests, so the asymmetric fixtures are real. It stops at the first frame-header marker whether or not that payload is usable, rather than scanning on for a better one; the ticket said "first usable", and stopping is the safer reading since an unreadable header falls back to the size-limit trigger.
