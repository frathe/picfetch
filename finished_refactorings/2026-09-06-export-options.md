# Export options: size limit + metadata omission

Route: **Deep** (4 tickets, 4 packages, user-visible strings, new byte-level
machinery). Spec: `.scratch/export-options/spec.md`; tickets 01-04 under
`.scratch/export-options/issues/`.

Deliverable: the export prompt grows an export size limit and an
"Include camera metadata (JPEG only)" checkbox, both defaulting to today's
behaviour, and a resized JPEG that keeps its metadata no longer carries
dimension tags that a resize made false.

## Correction to the spec

The spec and ticket 01 say `imaging.Export` "has two call sites; the wallpaper
path passes defaults". There are **three** production call sites:
`internal/ui/export.go:160`, `internal/ui/wallpaper.go:137`, and
`internal/ui/mosaicwin` — which holds it as a field value
(`window.go:106,140`) and re-exposes its type through
`SetExporter` (`export.go:115`), stubbed in 5 test call sites. Widening
`Export` therefore also widens `mosaicwin`'s `exporter` field and
`SetExporter` parameter. That is mechanical and the mosaic path passes
defaults, so the decision stands; the count in the spec was wrong.

## Contracts

```go
// internal/imaging
type ExportOptions struct {
    MaxEdge      int  // longest-edge ceiling; 0 = original size
    OmitMetadata bool // false = today's behaviour (source tags copied)
}
func Export(dest fyne.URI, img image.Image, src fyne.URI, opts ExportOptions) error
func FitEdge(w, h, maxEdge int) (int, int) // promoted from fitEdge; total for maxEdge <= 0
func ScaleForExport(src image.Image, maxEdge int) image.Image // CatmullRom

// internal/ui/widgets
type ExtraRows interface {
    Content() fyne.CanvasObject
    Rows() int
    Focus(row int) // -1: the button row holds the selection
    HandleKey(ev *fyne.KeyEvent)
    Reset()
}
func NewChoiceCardWithRows(repaint func(), rows ExtraRows, choices ...Choice) *ChoiceCard
func Ringed(ring *canvas.Rectangle, obj fyne.CanvasObject) *fyne.Container // widened from *widget.Button

// internal/ui
type exportRequest struct { ext string; opts imaging.ExportOptions }
func (v *viewer) runExport(src fyne.URI, img image.Image, req exportRequest)
type exportOptions struct{...} // implements widgets.ExtraRows; v.exportOptions
```

`ExportOptions`' zero value is exactly today's behaviour, so every caller that
does not care (wallpaper, mosaic) passes `imaging.ExportOptions{}`.

Card keyboard model: vertical stops are rows `0..Rows()-1` then the button row
(`Rows()`). `Show` resets to the button row, so `Cmd/Ctrl+E` + `Return` stays a
two-keystroke export. Up/Down move between stops. Escape always cancels the
whole prompt; Return is offered to the focused row first and commits only when
the row declines it (the metadata checkbox takes it, the size row does not);
every other key goes to the focused row, or to the panel when the buttons hold
the selection. The button ring mutes while the selection is up in the rows, so
exactly one mark on the card is ever at full strength.

## Tasks

### Task 1 — ticket 01: options value threaded end to end, card grows a rows slot
Owner:   T0 inline
Files:   `internal/imaging/save.go`, `internal/ui/wallpaper.go`,
         `internal/ui/mosaicwin/{window,export}.go` (+ its 5 test stubs),
         `internal/ui/widgets/{choicecard,style}.go`, `internal/ui/export.go`
Test:    a card built with rows delegates Up/Down and unhandled keys to them
         while Return still commits and Escape still cancels; a card built
         without rows behaves exactly as before; export with default options
         writes what it writes today.
Verify:  `go test ./internal/ui/widgets/ ./internal/imaging/ ./internal/ui/mosaicwin/`
         plus `go test ./internal/ui/ -run 'Export|Delete|Compare|CopySelection|Grid'`
Budget:  0 spawns · full suite: no

### Task 2 — ticket 02: export size limit
Owner:   T0 inline
Files:   `internal/imaging/{thumbnail,save}.go`, `internal/ui/exportoptions.go` (new),
         `internal/ui/{export,features,viewer}.go`, `translations/*.json`
Test:    each rung produces the expected longest edge; a photo inside the rung
         is written at its own size; the Original label states the frame's real
         longest edge (RAW: the embedded preview's); the suggested filename
         carries the applied size only when the pixels changed; the toast
         reports the size only when it changed; the prompt reopens at Original.
Verify:  `go test ./internal/imaging/ -run 'Export|FitEdge|ScaleForExport'` and
         `go test ./internal/ui/ -run 'Export'`
Budget:  0 spawns · full suite: no

### Task 3 — ticket 03: metadata omission
Owner:   T0 inline
Files:   `internal/imaging/save.go`, `internal/ui/exportoptions.go`,
         `internal/ui/export.go`, `translations/*.json`
Test:    unchecked writes a JPEG whose tags are gone when read back, ICC
         survives, an orientation != 1 source comes out upright, APP14 is not
         spliced back, the source file is untouched, PNG is unaffected, the
         toast reports omission only when the box was unchecked, and the box is
         back to checked on the next open.
Verify:  `go test ./internal/imaging/ -run 'Export'` and
         `go test ./internal/ui/ -run 'Export'`
Budget:  0 spawns · full suite: no

### Task 4 — ticket 04: drop dimension tags a resize invalidated
Owner:   T0 inline
Files:   `internal/imaging/jpegexif.go` (+ `save.go` wiring)
Test:    a resized export with metadata included has none of IFD0 0x0100/0x0101,
         SubIFD 0xA002/0xA003/0x9214/0xA214, Interop 0x1001/0x1002; camera,
         lens, exposure, date, GPS, MakerNote and DPI survive; every surviving
         entry's value still reads back (absolute offsets survived the shift);
         nothing is dropped at Original size or when the rung exceeded the
         photo; a truncated block is left unchanged; `SaveRotated` output is
         unchanged.
Verify:  `go test ./internal/imaging/ -run 'Exif|Export|Save'`
Budget:  0 spawns · full suite: no

### Task 5 — land
Docs (`ARCHITECTURE.md`, `manual.md`, `manual_de.md`, `todos.md`), qodana test
exclusions, UI shard manifest, then the final gate.
Verify:  `make check-qodana-test-exclusions`, `make check-test-shards`,
         `make fmt-check`, `go vet ./...`, `make test` (Docker linux/amd64).

## Verification note

Golden rendering is architecture-dependent: the final gate is `make test`
(Linux/amd64 Docker), never a bare native run. Iteration uses targeted native
package runs; `TestE2E_CopySelection` failing natively on darwin is the known
platform baseline, not a regression.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| 1    | 0 / 0                  | 1             | no         | hot context; test-stub updates scripted (rule S) |
| 2    | 0 / 0                  | 1             | no         | hot context |
| 3    | 0 / 0                  | 1             | no         | hot context |
| 4    | 0 / 0                  | 1             | no         | hot context; four guards negatively verified |
| gate | —                      | —             | yes        | Docker linux/amd64 |
