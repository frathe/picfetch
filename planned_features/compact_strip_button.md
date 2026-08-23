# Compact and correctly hide the EXIF “Remove Metadata” button

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task before dispatching the next. Do not
> start Task N+1 until Task N is reviewed and fixed. **Do not run
> `git commit`.**

**Goal:** In the EXIF window, the **Remove Metadata** button must not occupy
layout space when `imaging.CanStripJPEGMetadata` is false, and when it is
shown it must be a compact (shrink-wrapped) control instead of a full-width
`DangerImportance` bar.

**Architecture:** Keep the existing visibility predicate
(`Window.canStrip = imaging.CanStripJPEGMetadata(data)` in `Refresh`). Do
**not** gate the button on `ReadMetadata(data).Empty()` or on the panel
showing “No EXIF metadata found in this file.” — those miss XMP, COM,
IPTC, and bytes after EOI, which are still privacy leaks. The bug that
makes a hidden button still *feel* present is Fyne’s: `Hide()` on a child
does not re-run the parent layout (the Location disclosure already
documents this). Store the north `VBox`, `Refresh` it (and the window
content) after Show/Hide, and wrap the button in `container.NewCenter` so
a `VBox` cannot stretch it to the panel width. Drop the extra
`container.NewPadded` around the button. Confirm-dialog **Remove Metadata**
stays a full-width `DangerImportance` choice.

**Tech Stack:** Go 1.26, Fyne v2.8, existing `internal/ui/exifwin` +
`internal/imaging.CanStripJPEGMetadata`. No new dependencies. No imaging
behavior change.

**Spec:** `todos.md` TODO
“hide the delete meta data button when there are no meta data to remove.
also make the button smaller”.

**Precedent:** `syncStripVisible` already Show/Hides `strip`/`stripBar`.
`toggleLocation` already `w.location.Refresh()` after Show/Hide of
`w.body`. Location header uses `widget.LowImportance`; the strip *confirm*
uses `widget.DangerImportance` (`confirm.go`). Trailer-only JPEGs must
keep showing the button (`planned_features/truncate_mpf_trailers.md` Q2,
locked).

**What is already done (do not re-implement):**

- `CanStripJPEGMetadata` / `jpegHasRemovableMetadata` (segments + trailer
  + orientation 2–8).
- `Refresh` sets `canStrip` and calls `syncStripVisible`.
- `TestStripButton_HiddenForAJPEGWithNoMetadata` (stdlib JPEG).
- `TestStripButton_ShownForAGPSJPEG`.
- `TestRequestStrip_ConfirmRemovesGPSAndCallsHost` asserts the button is
  not `Visible()` after a successful strip.
- Manual already says the button is hidden when nothing is left to remove.

This plan is the **layout collapse** and **compact chrome**, plus tests
that lock the empty-panel-but-still-removable cases so nobody “fixes”
visibility by hiding on `Metadata.Empty()`.

---

## Open questions (proposed defaults — confirm before dispatch)

Implementers treat the **Proposed** column as spec unless Florian
overrides it before Task 1 starts.

| # | Question | Proposed |
|---|----------|----------|
| 1 | Hide the button when the tag list is empty (“No EXIF metadata found in this file.”) even if `CanStripJPEGMetadata` is true (orientation-only Exif, XMP/COM/IPTC only, trailer-only)? | **No.** Keep `CanStripJPEGMetadata`. An empty *tag list* is not “nothing to remove.” Trailer-only files were locked as “button shows.” |
| 2 | How much smaller? | **Shrink-wrap width** (`container.NewCenter(strip)` so the VBox cannot stretch it) and **drop `container.NewPadded`**. Keep `widget.DangerImportance` on the panel button (destructive, matches confirm). Do not switch it to `LowImportance` like Location. Do not invent a custom height/font. |
| 3 | Confirm dialog button size? | **Unchanged.** `ChoicePanel` + `DangerImportance` stays as it is. |
| 4 | Default window height `exifH = 420`? | **Unchanged.** Smaller button only frees a few pixels of map; not worth retuning the floor. |
| 5 | Migrate `warmDone` → `completion.Signal`? | **No.** Next TODO; out of scope. |

---

## Dispatch order and models

Parent: this session. One implementer at a time. After each task: parent
reviews the diff, **fixes if needed**, then dispatches the next.

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | Collapse hidden `stripBar` (parent `Refresh`) + visibility tests (PNG, trailer-only, GPS→plain height 0) | `go-expert` · `claude-sonnet-5-thinking-high` | parent (this session), not a subagent |
| 2 | Compact button: `NewCenter`, no extra pad, keep `DangerImportance` + width test | `go-expert` · `composer-2.5-fast` | parent |
| 3 | `ARCHITECTURE.md`, manuals (empty-list ≠ hidden), `todos.md` | `generalPurpose` · `composer-2.5-fast` | parent |
| Final | Whole-branch review after Task 3 | — | parent; escalate to `generalPurpose` · `claude-opus-5-thinking-high` only if Task 1’s layout still leaves a gap |

Task 1 is Fyne layout (easy to get wrong; same trap as the map
disclosure). Do not downgrade it to the cheap model. Task 2 is
transcription once Task 1 stored `north`. Task 3 is docs.

Do **not** use Opus for Tasks 1–3. The work splits.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the EXIF-window strip
  control’s layout story changes (Task 3).
- Every user-visible string is `lang.L("English text")` with that exact
  key in every `translations/*.json` bundle. This plan adds **no** new
  strings.
- Viewer-independent packages (`internal/imaging`) return errors; they do
  not call `fyne.LogError`. Do not touch imaging in this plan.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do not migrate `exifwin.warmDone` onto `internal/completion.Signal`.
- Do not change `confirm.go`, `widgets.ChoicePanel`, or the confirm
  dialog’s `DangerImportance`.
- Do not gate `canStrip` on `ReadMetadata(data).Empty()` or on
  `formatExifMetadata` returning the empty-file sentence.
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 2.

---

## File map

| File | Role |
|------|------|
| `internal/ui/exifwin/exifwin.go` | `north` field; `syncStripVisible` Refresh; Task 2 wraps `strip` in `NewCenter` |
| `internal/ui/exifwin/exifwin_test.go` | Visibility, height-collapse, compact-width tests |
| `ARCHITECTURE.md` | `exifwin/` row: compact button + parent Refresh on hide |
| `internal/ui/help/manual.md`, `manual_de.md` | Empty tag list can still show the button (XMP/COM/trailer) |
| `todos.md` | Move this TODO under Done; leave `warmDone` under TODO |

No imaging files. No new packages. No new translation keys.

---

## Assumptions (locked for implementers)

1. **Predicate unchanged:** `canStrip = imaging.CanStripJPEGMetadata(data)`
   on a successful `ReadAndProbe`; `false` on read error. Same as today.
2. **Hide both `strip` and `stripBar`.** A hidden inner button inside a
   visible padded/centered bar still eats height.
3. **After Show/Hide, Refresh `north` and the window content** so Border
   gives the freed height to the map. Copy the Location comment’s rationale
   in `syncStripVisible`.
4. **Trailer-only / XMP-only / COM-only / orientation-only:** button
   **shown** even when `formatExifMetadata` is the empty-file sentence.
5. **PNG/HEIC/stdlib JPEG / post-strip:** button **hidden** and bar height
   **0** after layout.
6. **Compact:** `container.NewCenter(w.strip)` is the north stack’s second
   child. No `NewPadded`. `Importance` stays `widget.DangerImportance`.
7. **`StripButton()` still returns `w.strip`**, not the Center wrapper —
   existing tests tap `StripButton().OnTapped` and use `absolutePos` on the
   button.
8. **`exifW` / `exifH` / `mapH` unchanged.**

---

## Task 1: Collapse the hidden strip bar

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go` (`Window` fields, `Show` build
  callback, `syncStripVisible`, close callback)
- Modify: `internal/ui/exifwin/exifwin_test.go`

**Interfaces:**
- Consumes: existing `imaging.CanStripJPEGMetadata`, `uitest.TempJPEGURI`,
  `uitest.TempGPSJPEGURI`, `uitest.EncodeJPEG`, `uitest.EncodePNG`,
  `uitest.WriteTempFile`, `stubHost.current`, `Window.Refresh`,
  `Window.StripButton`, `Window.Show`.
- Produces:
  - `Window.north *fyne.Container` — the north `VBox` (padded text +
    `stripBar`). Nil while the window is closed.
  - `syncStripVisible` Refresh of `north` and of `w.Window().Content()`
    after Show/Hide.

**Do not** change button Importance, padding, or Center wrapping in this
task. Task 2 does chrome. A full-width hidden bar that collapses to height
0 still satisfies Task 1.

- [ ] **Step 1: Write the failing tests** in `exifwin_test.go`

Append after `TestStripButton_ShownForAGPSJPEG` (today around line 576).
Same-package tests may read `w.stripBar`.

```go
func TestStripButton_HiddenForAPNG(t *testing.T) {
	app := test.NewApp()
	u := storage.NewFileURI(uitest.WriteTempFile(t, "plain.png", uitest.EncodePNG(t, 8, 8, color.White)))
	host := &stubHost{current: func() (fyne.URI, bool) { return u, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.StripButton() == nil || w.StripButton().Visible() {
		t.Fatal("want the button hidden for PNG")
	}
}

func TestStripButton_ShownForATrailerOnlyJPEG(t *testing.T) {
	app := test.NewApp()
	data := append(uitest.EncodeJPEG(t, 8, 8, color.White), []byte("ftypmp42fake-video")...)
	u := storage.NewFileURI(uitest.WriteTempFile(t, "trailer.jpg", data))
	host := &stubHost{current: func() (fyne.URI, bool) { return u, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.StripButton() == nil || !w.StripButton().Visible() {
		t.Fatal("want the button shown: bytes after EOI are removable even when the tag list is empty")
	}
	if got := w.Text().Text; got != lang.L("No EXIF metadata found in this file.") && got != "No EXIF metadata found in this file." {
		t.Fatalf("text = %q, want the empty-panel message (ReadMetadata sees no camera tags)", got)
	}
}

func TestStripButton_HiddenBarTakesNoHeightAfterNavigate(t *testing.T) {
	app := test.NewApp()
	gps := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)
	plain := uitest.TempJPEGURI(t, "plain.jpg", 8, 8, color.White)
	shown := gps
	host := &stubHost{current: func() (fyne.URI, bool) { return shown, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))

	if w.StripButton() == nil || !w.StripButton().Visible() {
		t.Fatal("setup: GPS JPEG should show the button")
	}
	if w.stripBar == nil || w.stripBar.Size().Height <= 0 {
		t.Fatal("setup: visible stripBar should have height")
	}

	shown = plain
	w.Refresh()

	if w.StripButton().Visible() {
		t.Fatal("want the button hidden after navigating to a JPEG with nothing removable")
	}
	if got := w.stripBar.Size().Height; got != 0 {
		t.Fatalf("hidden stripBar height = %v, want 0 (parent layout must run after Hide)", got)
	}
}
```

Also extend `TestRequestStrip_ConfirmRemovesGPSAndCallsHost`: after the
existing `StripButton().Visible()` check, assert:

```go
	if w.stripBar != nil && w.stripBar.Size().Height != 0 {
		t.Fatalf("stripBar height after strip = %v, want 0", w.stripBar.Size().Height)
	}
```

`TestRefresh_DismissesConfirmWhenTheFileChanges` already swaps GPS → plain;
do not duplicate its overlay assertion. The new navigate test is about
**height**, not the confirm overlay.

If `EncodePNG` / `WriteTempFile` names differ, use the existing helpers in
`internal/uitest/uitest.go` (`EncodePNG`, `WriteTempFile`, `EncodeJPEG`,
`TempJPEGURI`, `TempGPSJPEGURI`). If `storage` is not imported in the test
file, add `"fyne.io/fyne/v2/storage"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/ui/exifwin/ -run 'TestStripButton_HiddenForAPNG|TestStripButton_ShownForATrailerOnlyJPEG|TestStripButton_HiddenBarTakesNoHeightAfterNavigate|TestRequestStrip_ConfirmRemovesGPSAndCallsHost' -v`

Expected:

- `HiddenForAPNG` may **pass already** (PNG ⇒ `CanStripJPEGMetadata`
  false). Keep it; it locks the non-JPEG hide.
- `ShownForATrailerOnlyJPEG` should **pass already** if `CanStripJPEGMetadata`
  is true for trailers (it is). Keep it; it locks Q1.
- `HiddenBarTakesNoHeightAfterNavigate` should **FAIL** with
  `hidden stripBar height = …, want 0` — that is the Task 1 bug.
- The extended confirm test should **FAIL** the new height assertion
  after a successful strip.

If the height test already passes on this Fyne version, still do Step 3:
the Location comment is the project rule, and navigating without
`north.Refresh()` is a latent gap. Do not skip the Refresh because one
driver laid out by accident.

- [ ] **Step 3: Store `north` and Refresh after Show/Hide**

On `Window`, next to `stripBar`:

```go
	// north is the VBox holding the tag list and stripBar, live only while
	// the window is open. syncStripVisible Refreshes it after Show/Hide:
	// Fyne does not re-run a parent's layout when a child is hidden, so
	// without this a hidden stripBar keeps the height of the last layout
	// (the same trap toggleLocation documents for the map body).
	north *fyne.Container
```

In the `Show` build callback, replace the anonymous VBox:

```go
		w.north = container.NewVBox(
			container.NewPadded(w.text),
			w.stripBar,
		)
		w.syncStripVisible()

		return container.NewBorder(
			w.north,
			nil, nil, nil,
			w.location,
		)
```

Call `syncStripVisible` **after** `north` is assigned (today it runs
before the Border is built; either order is fine once `syncStripVisible`
nil-checks `north`).

In the close callback, add `w.north = nil` next to `w.stripBar = nil`.

Replace `syncStripVisible` with:

```go
func (w *Window) syncStripVisible() {
	if w.strip == nil {
		return
	}

	if w.canStrip {
		w.strip.Show()
		if w.stripBar != nil {
			w.stripBar.Show()
		}
	} else {
		w.strip.Hide()
		if w.stripBar != nil {
			w.stripBar.Hide()
		}
	}

	// Showing/hiding a child does not re-run its parent's layout, and a
	// hidden child is given no space only on the next layout - without
	// this a hidden stripBar keeps a full-width hole above the map.
	if w.north != nil {
		w.north.Refresh()
	}
	if win := w.Window(); win != nil {
		if c := win.Content(); c != nil {
			c.Refresh()
		}
	}
}
```

Do not change `strip.Importance` or `stripBar = container.NewPadded(...)`
in this task.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/exifwin/ -run 'TestStripButton_|TestRequestStrip_|TestRefresh_DismissesConfirm' -v`

Expected: PASS, including `HiddenBarTakesNoHeightAfterNavigate` and the
post-strip height assertion.

Then from the repository root:

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/exifwin/
```

`gofmt -l .` must print nothing. If it names a file, run `gofmt -w` on it
and re-check.

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
exifwin: collapse the Remove Metadata bar when there is nothing to strip

Hide() left the north stack at the last layout height, so a JPEG with
nothing removable still reserved a full-width hole above the map.
```

---

## Task 2: Shrink-wrap the button

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go` (`Show` build callback: how
  `stripBar` is built; field comment on `stripBar`)
- Modify: `internal/ui/exifwin/exifwin_test.go`

**Interfaces:**
- Consumes: Task 1’s `north`, `syncStripVisible`, `stripBar`.
- Produces: `stripBar` is `container.NewCenter(w.strip)` (no extra padded
  wrapper). `strip.Importance` remains `widget.DangerImportance`.
  `StripButton()` still returns `w.strip`.

- [ ] **Step 1: Write the failing test**

```go
func TestStripButton_DoesNotSpanTheWindow(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))

	btn := w.StripButton()
	if btn == nil || !btn.Visible() {
		t.Fatal("setup: GPS JPEG should show the button")
	}
	if btn.Importance != widget.DangerImportance {
		t.Errorf("Importance = %v, want widget.DangerImportance", btn.Importance)
	}
	// VBox stretches a naked/padded button to the north stack's width.
	// Center keeps the button at its label MinSize; anything near the
	// panel width means it is still a full-width bar.
	if btn.Size().Width >= exifW*0.8 {
		t.Fatalf("button width %v fills the %v-wide panel; want shrink-wrapped to the label", btn.Size().Width, exifW)
	}
}
```

`TestStripButton_SitsAboveTheMap` must still pass: `absolutePos` walks
containers, so a Center parent is fine. If that test starts failing
because Center reports a zero button size before layout, Resize as this
test does (`exifW` × `exifH`) inside `SitsAboveTheMap` — only if it
fails, do not pre-emptively churn it.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -count=1 ./internal/ui/exifwin/ -run TestStripButton_DoesNotSpanTheWindow -v`

Expected: FAIL `button width … fills the …-wide panel`.

- [ ] **Step 3: Wrap the button in Center; drop extra pad**

In the `Show` build callback, replace:

```go
		w.strip = widget.NewButton(lang.L("Remove Metadata"), w.requestStrip)
		w.strip.Importance = widget.DangerImportance
		w.stripBar = container.NewPadded(w.strip)
```

with:

```go
		w.strip = widget.NewButton(lang.L("Remove Metadata"), w.requestStrip)
		w.strip.Importance = widget.DangerImportance
		// Center, not a VBox child of the button itself: VBox would
		// stretch a DangerImportance button across the panel. No extra
		// NewPadded - the label's north pad is enough.
		w.stripBar = container.NewCenter(w.strip)
```

Update the field comment on `stripBar` (today: “padded wrapper”) to:

```go
	// stripBar is the Center wrapper around strip in the north stack,
	// hidden as a unit so a hidden button does not leave a hole.
	// Center is what keeps VBox from stretching the button to full width.
```

Do not change `confirm.go`. Do not change `requestStrip` copy.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/exifwin/ -run 'TestStripButton_|TestRequestStrip_|TestRefresh_DismissesConfirm' -v`

Expected: PASS.

Then from the repository root:

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/exifwin/
go test -count=1 ./internal/ui/ -run TestStripMetadata
```

(`TestStripMetadata_HidesExifLinkAndShrinksReportedSize` in
`internal/ui/exif_test.go` taps `StripButton()`; it must still find the
button.)

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
exifwin: shrink-wrap the Remove Metadata button

A VBox was stretching the DangerImportance control into a full-width bar.
Center keeps it at the label size; the confirm dialog stays unchanged.
```

---

## Task 3: Docs and todos

**Files:**
- Modify: `ARCHITECTURE.md` (`exifwin/` row)
- Modify: `internal/ui/help/manual.md` (EXIF section + keyboard cheat sheet)
- Modify: `internal/ui/help/manual_de.md` (same two places)
- Modify: `todos.md`

**Interfaces:** none. No code.

- [ ] **Step 1: `ARCHITECTURE.md`**

In the `exifwin/` table cell, the sentence that currently says the
Remove Metadata button is stacked above Location and hidden via
`CanStripJPEGMetadata`: keep that predicate. Add that the control is
**shrink-wrapped** (`container.NewCenter`, not a full-width
`DangerImportance` bar) and that `syncStripVisible` **Refreshes the north
stack** after Hide so the map inherits the gap (same Fyne layout trap as
the Location disclosure).

Do not mention `warmDone`. Do not rewrite the rest of the cell.

- [ ] **Step 2: Manuals**

English (`internal/ui/help/manual.md`), after the sentence “The button is
hidden for a JPEG with nothing left to remove, including after a
successful strip.” add one sentence:

> The tag list only shows camera/lens/exposure/GPS fields, so a JPEG that
> still has comments, XMP, IPTC, or bytes after the image (a second
> picture or a motion-photo video) can show “no metadata found” and still
> offer **Remove Metadata**. The button itself is a compact control, not a
> full-width bar.

In the cheat-sheet bullet that mentions the button (“hidden once nothing
is left to remove”), do not duplicate the XMP/trailer lecture; leave that
bullet as the short version.

German (`manual_de.md`): the same fact after the matching
“Die Schaltfläche fehlt, sobald nichts mehr zu entfernen ist” paragraph:

> Die Tag-Liste zeigt nur Kamera/Objektiv/Belichtung/GPS. Ein JPEG, das
> noch Kommentare, XMP, IPTC oder Bytes hinter dem Bild hat (zweites Bild
> oder Motion-Photo-Video), kann „keine Metadaten gefunden“ zeigen und
> trotzdem **Metadaten entfernen** anbieten. Die Schaltfläche ist ein
> kompaktes Bedienelement, keine volle Breitseite.

No new `lang.L` keys (manuals are markdown, not `lang.L`).

- [ ] **Step 3: `todos.md`**

Move

```
## hide the delete meta data button when there are no meta data to remove.
also make the button smaller
```

from TODO into Done, as a short past-tense note: the button was already
gated on `CanStripJPEGMetadata`; this change collapses the leftover
layout hole and shrink-wraps the control. Leave the `warmDone` item under
TODO, untouched.

- [ ] **Step 4: Verify**

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/exifwin/ ./internal/ui/help/
```

(help package if manuals are tested there; if `go test` on `help` is a
no-op besides compile, that is fine.)

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
docs: EXIF Remove Metadata is compact and can show when the tag list is empty

The button follows CanStripJPEGMetadata, not the camera-tag list, so
XMP/COM/trailers still offer a strip.
```

---

## Self-review (plan vs spec)

| Spec / TODO | Task |
|-------------|------|
| Hide button when nothing to remove | Already implemented; Task 1 makes Hide **collapse layout** + PNG/post-strip height |
| Do not hide privacy leftovers the tag list ignores | Task 1 `ShownForATrailerOnlyJPEG`; Task 3 manuals |
| Make the button smaller | Task 2 Center + drop pad; keep DangerImportance |
| Confirm dialog unchanged | Global constraint; Task 2 |
| No imaging change | File map |
| No `warmDone` migration | Global constraint; todos stay |
| ARCHITECTURE / manuals / todos | Task 3 |

No placeholders. Signatures match existing `Host` /
`CanStripJPEGMetadata` / `StripButton()`.

---

## Parent review checklist (after every task)

1. Diff is only the files in that task’s File map.
2. `canStrip` is still exactly `imaging.CanStripJPEGMetadata(data)` (or
   false on read error).
3. No `git commit`.
4. Task 1: `HiddenBarTakesNoHeightAfterNavigate` passes; trailer-only
   still **shows** the button.
5. Task 2: `DoesNotSpanTheWindow` passes; `Importance` is still
   `DangerImportance`; confirm tests still pass.
6. Task 3: `warmDone` TODO still present; this feature is under Done.

If Task 1’s height assertion still fails after `north.Refresh()` +
content `Refresh()`, try `w.north.Refresh()` then
`w.Window().Content().Refresh()` in that order (already specified). If
the test driver still reports a stale size, call
`w.Window().Resize(fyne.NewSize(exifW, exifH))` once more after
`Refresh()` in the **test** (not in production `syncStripVisible`) to
force a layout pass — document that in the test comment as a Fyne test
driver quirk, not as a sleep.
