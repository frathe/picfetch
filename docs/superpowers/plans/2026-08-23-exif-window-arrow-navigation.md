# EXIF Window Arrow-Key Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller:** After every task, the parent agent reviews the diff, fixes issues, then starts the next task. Do **not** run `git commit` (repo `AGENTS.md` rule). Do **not** start the next task until the parent says the current one is accepted.
>
> **Do not start any task until Florian has answered the open points and explicitly said to start the subagents.**
>
> **Do not read this whole file if you are a task subagent.** Read only the task you were given, plus Global Constraints and the Interfaces block on that task.

**Goal:** When the EXIF metadata window is open and is the focused window, Left and Right change the current image the same way they do in the main window, and the panel refreshes to the new file.

**Architecture:** The EXIF window is a separate Fyne window. Keys typed there never reach `viewer.handleKeyEvent` (`internal/ui/keys.go`). Give `widgets.Singleton` an optional extra-key callback (Escape still closes). The EXIF panel uses that callback to call a new narrow `Host.StepImage(delta int)`. `viewer.StepImage` is the shared next/prev implementation used by both the main window's Left/Right (and Up/Down when not in picture-frame mode) and the EXIF panel, including wrap-around, the in-flight load guard, slideshow kick, and "don't navigate over a delete/export prompt" guards. `finishLoad` already calls `v.exif.Refresh()`, so `handleKey` must not refresh on its own.

**Tech Stack:** Go 1.26.7, Fyne v2.8 (`fyne.io/fyne/v2`), existing `internal/ui/exifwin`, `internal/ui/widgets.Singleton`, `viewer` Host pattern.

**Spec:** This plan is the spec. Source todo: `todos.md` — "When the exif window gets openend and has focus, it should be possible to change the image with left an right keys".

## Global Constraints

- Do not `git commit`. The parent reviews each task; the user commits later. Suggested commit messages in this plan are for the user, not for subagents to run.
- Every user-visible string is `lang.L("...")` with a matching key in every `translations/*.json` bundle. This feature adds **no** new UI strings.
- Feature packages talk to the app only through their own `Host`. Do not import `internal/ui` from `exifwin`. Do not pass `appState`.
- Cross-feature decisions stay in `internal/ui`. `StepImage` lives on `viewer`; `exifwin` only calls `Host.StepImage`.
- `widgets.Singleton` is shared by the manual, About, Settings, and EXIF. Extra keys must be **opt-in**. Settings/help/about must keep Escape-closes-only behavior.
- Fyne glfw (`internal/driver/glfw/window.go`) delivers `TypedKey` to `Canvas.Focused()` when it is non-nil, otherwise to `SetOnTypedKey`. A focused `widgets.ChoicePanel` (Remove Metadata confirmation) swallows Left/Right/Escape. That is load-bearing: arrows must move the confirmation ring, not the image, while the prompt is up.
- Do not use `Canvas.AddShortcut` for unmodified Left/Right. Shortcuts can fire while the confirmation panel is focused and would steal those keys.
- Do not call `viewer.Advance()` from EXIF navigation. `Advance` is slideshow-aware (shuffle via `randomOtherIndex`). Arrow keys always step `index ± 1`.
- `ShowImage` already wraps at both ends (`internal/ui/load.go`). `StepImage` must use that, not clamp.
- Same no-op conditions as the main window's arrow keys: fewer than two files, or `v.loading.Load()`.
- Manual navigation must `slides.Kick()` when picture-frame mode is on (same as `handleKeyEvent` after a successful Left/Right).
- Tests: TDD. No `time.Sleep` to guess completion. Use existing `dropAndWait` / `waitUntilLoaded` / `waitForWarm`. Fyne's test driver runs `fyne.Do` inline.
- After implementation in a task, run that task's focused tests. Do not claim the whole suite passed unless you ran it.
- Update `ARCHITECTURE.md` in the same change that changes packages/Host/Singleton behavior (Task 4). Use the real helper names: `newTestUI` / `newTestViewer`, `dropAndWait`, `waitUntilLoaded`, `testApp`, `stubHost`, `v.deletion` / `v.exportPrompt` / `v.slides`, `Canvas().SetOnTypedKey` / `OnTypedKey()`.
- English comments; match surrounding style (full sentences, "why" not "what").
- Open work belongs in `todos.md`; do not add `TODO`/`FIXME` comments to source. Do not mark the todo done until Florian accepts the feature.

## Decisions (defaults — confirm before execution)

These are the recommended answers to the open points. **Do not change them in a subagent** unless the parent has recorded a different choice in this table.

| # | Question | Default |
|---|----------|---------|
| D1 | Which keys in the EXIF window change image? | **Left and Right only.** Up/Down/Home/End stay no-ops there. (Main-window Up/Down still next/prev, or slideshow interval when picture-frame is on. Home/End stay main-window only.) |
| D2 | Wrap at ends? | **Yes**, same as `ShowImage` / main window. |
| D3 | Remove Metadata confirmation showing? | **Arrows stay on the prompt** (ChoicePanel focused). Also guard `handleKey` with `w.confirm != nil` so a test (or a missed focus) cannot navigate by invoking the canvas handler directly. |
| D4 | Map / Location / strip button focused? | **Arrows still change image** after `releaseKeyboard()`: Unfocus unless a confirmation is up. Call it after `Show`, after confirmation `OnClosed`, and at the end of `toggleLocation`. Map zoom buttons inside `fyne.io/x/fyne/widget.Map` are an accepted edge case — clicking those may eat arrows until something else unfocuses. Do not wrap the map. |
| D5 | Slideshow running? | **Step `index ± 1` and `Kick()`**, same as main-window Left/Right. Never `Advance()` (shuffle). |
| D6 | Grid or delete/export prompt visible on the main window? | **No-op** if delete confirmation or export-format prompt is visible. **Still navigate** if the grid is visible (EXIF describes the current file; `ShowImage` is the source of truth). Opening EXIF while the grid is up is uncommon (`E` is swallowed by the grid), but EXIF can already be open when `G` is pressed on the main window. |
| D7 | Single file? | **No-op** (`len(files) < 2`), same as main window. |
| D8 | Load in flight? | **No-op** (`v.loading.Load()`), same as main window. |

## Subagent assignment

Run **strictly in order**. Task 2 must implement `Host.StepImage` and `viewer.StepImage` in the same change so `go test ./...` still compiles.

Cursor `Task` subagent types that exist in this repo: `go-expert`, `generalPurpose`, `code-simplifier`. There is **no** `code-reviewer` type — reviews use `generalPurpose`.

| Task | What | Subagent type | Model | Why |
|------|------|---------------|-------|-----|
| 1 | `widgets.Singleton` extra-key hook | `go-expert` | `composer-2.5-fast` | Mechanical, 1–2 files, full snippets below |
| 2 | `Host.StepImage` + `viewer.StepImage` + keys.go DRY | `go-expert` | `cursor-grok-4.6-xhigh` | Multi-file, guards, slideshow, must not call `Advance` |
| 3 | Wire EXIF window keys + Unfocus + package/integration tests | `go-expert` | `claude-sonnet-5-thinking-high` | Fyne focus vs `SetOnTypedKey` vs confirmation |
| 4 | Manual EN/DE + `ARCHITECTURE.md` | `generalPurpose` | `composer-2.5-fast` | Copy only, no logic |
| Per-task review | Spec + quality | `generalPurpose` | `claude-sonnet-5-thinking-high` | Mid-tier floor for reviewers |
| Parent fix after each task | Repair review findings | parent, or `go-expert` @ `cursor-grok-4.6-xhigh` if the parent should not edit | Integration judgment |
| Final review | Whole branch | `generalPurpose` | `claude-opus-5-thinking-high` | Last gate only — **do not use Opus to implement** |

This feature is small enough to split. **Do not dispatch Opus as an implementer.** If Task 3 is BLOCKED on Fyne focus behavior, the parent may re-dispatch Task 3 on `cursor-grok-4.6-xhigh`, not Opus.

## File map

| File | Role |
|------|------|
| `internal/ui/widgets/singleton.go` | Opt-in `SetExtraKeys`; Escape still closes |
| `internal/ui/widgets/singleton_test.go` | Hook fires for Left/Right; Escape still closes; nil hook unchanged |
| `internal/ui/exifwin/exifwin.go` | `Host.StepImage`; `handleKey`; `SetExtraKeys` in `New`; `releaseKeyboard` |
| `internal/ui/exifwin/confirm.go` | `releaseKeyboard` from confirmation `OnClosed` |
| `internal/ui/exifwin/exifwin_test.go` | `stubHost.StepImage`; Left/Right call it; confirmation does not; canvas unfocused |
| `internal/ui/exifwin/confirm_test.go` | Confirmation dismiss still leaves canvas unfocused (add one test) |
| `internal/ui/keys.go` | Left/Right (and Up/Down when not slideshow) call `StepImage` |
| `internal/ui/viewer.go` | `StepImage` implementation (Host method), next to `Advance` |
| `internal/ui/step_test.go` | **Create.** Viewer-level StepImage tests |
| `internal/ui/exif_test.go` | Integration: type Left/Right on the EXIF canvas, image + panel follow |
| `internal/ui/help/manual.md` | EN docs |
| `internal/ui/help/manual_de.md` | DE docs |
| `ARCHITECTURE.md` | Singleton extra keys; exifwin Host `StepImage`; keys.go note |

No new package. No new translation keys.

## Why this design (for the parent, not for task subagents)

Three approaches were considered:

1. **Forward EXIF-window keys into `handleKeyEvent`** — would also fire R/S/I/zoom/G from the panel, and would still hit the grid/deletion early-returns. Too broad.
2. **`Canvas.AddShortcut` for Left/Right on the EXIF window** — shortcuts fire even when ChoicePanel is focused, stealing confirmation arrows. Rejected.
3. **Opt-in Singleton extra-key callback + `Host.StepImage` (chosen)** — Settings/help/about stay Escape-only; EXIF opts in; confirmation keeps focus; one stepper shared with the main window.

---

### Task 1: Singleton extra-key hook

**Subagent:** `go-expert` @ `composer-2.5-fast`

**Files:**
- Modify: `internal/ui/widgets/singleton.go`
- Modify: `internal/ui/widgets/singleton_test.go`

**Interfaces:**
- Consumes: existing `Singleton.Show` (builds window, installs Escape-only `SetOnTypedKey` at the `win.Canvas().SetOnTypedKey` call around line 146)
- Produces:
  ```go
  func (s *Singleton) SetExtraKeys(f func(*fyne.KeyEvent))
  ```
  `Show`'s unfocused handler: Escape always `win.Close()` and returns; any other key calls `s.extraKeys` if non-nil. Read `s.extraKeys` **inside** the handler (not copied at install time) so `SetExtraKeys` before `Show` is enough, and a later `SetExtraKeys` still takes effect on an already-open window.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/widgets/singleton_test.go` (keep the existing `newSingletonContent` helper):

```go
func TestSingleton_ExtraKeysReceiveNonEscape(t *testing.T) {
	app := test.NewApp()
	var s Singleton
	var got []fyne.KeyName
	s.SetExtraKeys(func(ev *fyne.KeyEvent) {
		got = append(got, ev.Name)
	})
	s.Show(app, "extra", fyne.NewSize(300, 200), newSingletonContent, nil)
	defer s.Window().Close()

	handler := s.Window().Canvas().OnTypedKey()
	if handler == nil {
		t.Fatal("canvas has no OnTypedKey handler")
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyRight})
	handler(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if len(got) != 2 || got[0] != fyne.KeyRight || got[1] != fyne.KeyLeft {
		t.Errorf("extra keys = %v, want [Right Left]", got)
	}
	if !s.Open() {
		t.Error("Left/Right must not close the window")
	}
}

func TestSingleton_EscapeStillClosesWhenExtraKeysSet(t *testing.T) {
	app := test.NewApp()
	var s Singleton
	var extra int
	s.SetExtraKeys(func(*fyne.KeyEvent) { extra++ })
	s.Show(app, "esc", fyne.NewSize(300, 200), newSingletonContent, nil)

	handler := s.Window().Canvas().OnTypedKey()
	handler(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if s.Open() {
		t.Error("Escape should still close the window")
	}
	if extra != 0 {
		t.Errorf("extraKeys calls = %d, want 0 (Escape must not be forwarded)", extra)
	}
}

func TestSingleton_NilExtraKeysKeepsEscapeOnly(t *testing.T) {
	app := test.NewApp()
	var s Singleton
	s.Show(app, "plain", fyne.NewSize(300, 200), newSingletonContent, nil)
	defer s.Window().Close()

	handler := s.Window().Canvas().OnTypedKey()
	handler(&fyne.KeyEvent{Name: fyne.KeyRight}) // must not panic
	if !s.Open() {
		t.Error("Right with no extraKeys must leave the window open")
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if s.Open() {
		t.Error("Escape should still close")
	}
}

func TestSingleton_ExtraKeysReadAtEventTime(t *testing.T) {
	app := test.NewApp()
	var s Singleton
	s.Show(app, "late", fyne.NewSize(300, 200), newSingletonContent, nil)
	defer s.Window().Close()

	var got fyne.KeyName
	s.SetExtraKeys(func(ev *fyne.KeyEvent) { got = ev.Name })
	s.Window().Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got != fyne.KeyRight {
		t.Errorf("got %v, want Right — extraKeys must be read inside the handler, not copied at Show", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestSingleton_ExtraKeys|TestSingleton_EscapeStillClosesWhenExtraKeys|TestSingleton_NilExtraKeys' ./internal/ui/widgets/`

Expected: FAIL — `SetExtraKeys` undefined.

- [ ] **Step 3: Minimal implementation**

In `Singleton`, add this field next to `onTop`:

```go
	// extraKeys, if set, receives unfocused keys other than Escape.
	// Escape still closes this window (manual, About, Settings, EXIF).
	// The EXIF panel is the only caller today (Left/Right change image).
	extraKeys func(*fyne.KeyEvent)
```

After `KeepOnTop`:

```go
// SetExtraKeys registers a callback for unfocused keys other than Escape.
// Call before Show, or any time: the handler reads the field on each event.
// Nil means Escape-only, which is the default.
func (s *Singleton) SetExtraKeys(f func(*fyne.KeyEvent)) {
	s.extraKeys = f
}
```

Replace the `SetOnTypedKey` body in `Show` (currently Escape-only close) with:

```go
	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			win.Close()
			return
		}
		if s.extraKeys != nil {
			s.extraKeys(ev)
		}
	})
```

Do **not** change `Show`'s signature. Do **not** forward Escape to `extraKeys`. Do **not** call `SetExtraKeys` from help/about/settings.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/widgets/`

Expected: PASS. Help/About/Settings do not call `SetExtraKeys`; their tests must still pass.

- [ ] **Step 5: Parent review (no commit)**

Stop. Parent reviews. Suggested message if the user later commits this task alone: `Add optional extra-key callback to widgets.Singleton.`

---

### Task 2: `StepImage` on Host and viewer

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Depends on:** Task 1 accepted (not required to compile this task, but keep order).

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go` (`Host` only in this task — do not wire keys yet)
- Modify: `internal/ui/exifwin/exifwin_test.go` (`stubHost.StepImage`)
- Modify: `internal/ui/viewer.go` (`StepImage`, next to `Advance` around line 813)
- Modify: `internal/ui/keys.go` (the navigation `switch` starting at `case fyne.KeyRight, fyne.KeyDown` around line 269)
- Create: `internal/ui/step_test.go`

**Interfaces:**
- Consumes: `ShowImage(i int)` (wraps), `v.loading`, `v.state.files`, `v.state.index`, `v.slides.Active()` / `v.slides.Kick()`, `v.deletion.Visible()`, `v.exportPrompt.Visible()`
- Produces:
  ```go
  // on exifwin.Host:
  StepImage(delta int)

  // on viewer (satisfies Host):
  func (v *viewer) StepImage(delta int)
  ```
  `delta` is `+1` (next) or `-1` (previous). Other values still just add to index; `ShowImage` wraps.

**Behavior of `StepImage` (exact):**
1. Return if `v.deletion.Visible()` or `v.exportPrompt.Visible()`.
2. Return if `len(v.state.files) < 2` or `v.loading.Load()`.
3. `v.ShowImage(v.state.index + delta)`.
4. If `v.slides.Active()`, `v.slides.Kick()`.

Do **not** check grid visibility (D6). Do **not** call `Advance`.

**keys.go change:** Leave the picture-frame Up/Down interval block (the `if v.slides.Active()` switch on `KeyUp`/`KeyDown` *above* the `len < 2` guard) **unchanged**. Replace only the navigation `switch` that currently calls `ShowImage` for arrows:

Current (do not leave it this way):

```go
	switch ev.Name {
	case fyne.KeyRight, fyne.KeyDown:
		v.ShowImage(v.state.index + 1)
	case fyne.KeyLeft, fyne.KeyUp:
		v.ShowImage(v.state.index - 1)
	case fyne.KeyHome:
		v.ShowImage(0)
	case fyne.KeyEnd:
		v.ShowImage(len(v.state.files) - 1)
	case fyne.KeyS:
		v.toggleSort()
	default:
		return
	}

	if v.slides.Active() {
		v.slides.Kick()
	}
```

After this task:

```go
	switch ev.Name {
	case fyne.KeyRight, fyne.KeyDown:
		v.StepImage(1)
		return
	case fyne.KeyLeft, fyne.KeyUp:
		v.StepImage(-1)
		return
	case fyne.KeyHome:
		v.ShowImage(0)
	case fyne.KeyEnd:
		v.ShowImage(len(v.state.files) - 1)
	case fyne.KeyS:
		v.toggleSort()
	default:
		return
	}

	if v.slides.Active() {
		v.slides.Kick()
	}
```

Home/End/S still Kick via the tail. Left/Right/Up/Down Kick inside `StepImage`. Slideshow Up/Down interval adjustment stays **above** this switch, unchanged.

- [ ] **Step 1: Write failing viewer tests**

Create `internal/ui/step_test.go`:

```go
package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestStepImage_NextAndPrevWrap(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)

	start := v.state.index
	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index after StepImage(1) = %d, want %d", v.state.index, (start+1)%2)
	}
	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("index after wrap = %d, want %d", v.state.index, start)
	}
	v.StepImage(-1)
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index after StepImage(-1) wrap = %d, want %d", v.state.index, (start+1)%2)
	}
}

func TestStepImage_NoopWithOneFile(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	dropAndWait(t, v, a)
	v.StepImage(1)
	if v.state.index != 0 {
		t.Errorf("index = %d, want 0", v.state.index)
	}
}

func TestStepImage_NoopWhileLoading(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)
	start := v.state.index
	v.loading.Store(true)
	t.Cleanup(func() { v.loading.Store(false) })
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while loading", v.state.index, start)
	}
}

func TestStepImage_NoopWhileDeleteConfirmVisible(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.deletion.Request()
	if !v.deletion.Visible() {
		t.Fatal("setup: delete confirmation should be visible")
	}
	start := v.state.index
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while delete confirm is visible", v.state.index, start)
	}
}

func TestStepImage_NoopWhileExportPromptVisible(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.exportPrompt.Show(lang.L("Export as which format?"))
	if !v.exportPrompt.Visible() {
		t.Fatal("setup: export prompt should be visible")
	}
	start := v.state.index
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while export prompt is visible", v.state.index, start)
	}
}

func TestHandleKeyEvent_LeftRightUseStepImage(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)
	start := v.state.index
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("Right via handleKeyEvent index = %d, want next", v.state.index)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("Left via handleKeyEvent index = %d, want %d", v.state.index, start)
	}
}
```

Drop the unused `fyne` import if `TestHandleKeyEvent_LeftRightUseStepImage` is the only user — it needs `fyne.KeyEvent`. Keep `lang` for the export-prompt message.

- [ ] **Step 2: Run tests — expect FAIL** (`StepImage` undefined)

Run: `go test -count=1 -run 'TestStepImage|TestHandleKeyEvent_LeftRightUseStepImage' ./internal/ui/`

- [ ] **Step 3: Implement `Host.StepImage`, `stubHost`, `viewer.StepImage`, keys.go**

`Host` in `internal/ui/exifwin/exifwin.go` becomes:

```go
type Host interface {
	DisplayedFile() (fyne.URI, bool)
	AfterMetadataRemoved(u fyne.URI)
	ShowToast(msg string)
	StepImage(delta int)
}
```

`stubHost` in `exifwin_test.go` — add a field and method (keep existing `current` / `toasts` / `after` / `afterU`):

```go
type stubHost struct {
	current func() (fyne.URI, bool)
	toasts  []string
	after   int
	afterU  fyne.URI
	steps   []int
}

func (s *stubHost) StepImage(delta int) { s.steps = append(s.steps, delta) }
```

`viewer.go`, immediately after `Advance`:

```go
// StepImage moves by delta files (typically +1 or -1), wrapping through
// ShowImage. No-op with fewer than two files, while a load is in flight,
// or while the delete / export-format prompt owns the main window.
// Picture-frame shuffle does not apply: this is what the arrow keys do.
func (v *viewer) StepImage(delta int) {
	if v.deletion.Visible() || v.exportPrompt.Visible() {
		return
	}
	if len(v.state.files) < 2 || v.loading.Load() {
		return
	}
	v.ShowImage(v.state.index + delta)
	if v.slides.Active() {
		v.slides.Kick()
	}
}
```

Wire `keys.go` as specified above. Do not change the overlay / deletion / export / grid early returns, and do not change the slideshow Up/Down interval block.

- [ ] **Step 4: Run tests**

Run:

```
go test -count=1 -run 'TestStepImage|TestHandleKeyEvent' ./internal/ui/
go test -count=1 ./internal/ui/exifwin/
```

Expected: PASS. Existing Left/Right key tests still navigate. Existing `TestHandleKeyEvent_UpDownAdjustIntervalInsteadOfNavigating` and `TestHandleKeyEvent_UpDownNavigateOutsidePictureFrameMode` still pass. `exifwin` tests compile because `stubHost` now has `StepImage`.

- [ ] **Step 5: Parent review (no commit)**

Suggested message: `Share next/prev image stepping between the main window and EXIF Host.`

---

### Task 3: Wire Left/Right on the EXIF window

**Subagent:** `go-expert` @ `claude-sonnet-5-thinking-high`

**Depends on:** Tasks 1–2 accepted.

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go`
- Modify: `internal/ui/exifwin/confirm.go`
- Modify: `internal/ui/exifwin/exifwin_test.go`
- Modify: `internal/ui/exifwin/confirm_test.go`
- Modify: `internal/ui/exif_test.go`

**Interfaces:**
- Consumes: `Singleton.SetExtraKeys`, `Host.StepImage`, `w.confirm`
- Produces: unfocused Left → `StepImage(-1)`, Right → `StepImage(1)`; no-op while `w.confirm != nil`; `releaseKeyboard()` after Show, after confirmation close, and after `toggleLocation`

- [ ] **Step 1: Write failing package tests**

In `exifwin_test.go` (package already imports `fyne.io/fyne/v2` and `internal/ui/widgets`):

```go
func TestWindow_ArrowKeysStepImage(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	handler := w.Window().Canvas().OnTypedKey()
	if handler == nil {
		t.Fatal("EXIF canvas has no OnTypedKey handler")
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyRight})
	handler(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if len(host.steps) != 2 || host.steps[0] != 1 || host.steps[1] != -1 {
		t.Errorf("StepImage deltas = %v, want [1, -1]", host.steps)
	}
}

func TestWindow_UpDownHomeEndDoNotStepImage(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	handler := w.Window().Canvas().OnTypedKey()
	for _, name := range []fyne.KeyName{fyne.KeyUp, fyne.KeyDown, fyne.KeyHome, fyne.KeyEnd} {
		handler(&fyne.KeyEvent{Name: name})
	}
	if len(host.steps) != 0 {
		t.Errorf("StepImage deltas = %v, want none for Up/Down/Home/End", host.steps)
	}
}

func TestWindow_ShowLeavesCanvasUnfocused(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if got := w.Window().Canvas().Focused(); got != nil {
		t.Errorf("Focused() = %T, want nil so Left/Right reach OnTypedKey", got)
	}
}

func TestWindow_ArrowKeysIgnoredWhileConfirming(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.showConfirm(confirmation{title: "Title", message: "Message", action: "Confirm"})
	if _, ok := w.Window().Canvas().Focused().(*widgets.ChoicePanel); !ok {
		t.Fatalf("focused = %v, want ChoicePanel", w.Window().Canvas().Focused())
	}

	host.steps = nil
	if h := w.Window().Canvas().OnTypedKey(); h != nil {
		h(&fyne.KeyEvent{Name: fyne.KeyRight})
		h(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	if len(host.steps) != 0 {
		t.Errorf("StepImage during confirm = %v, want none", host.steps)
	}
	if _, ok := w.Window().Canvas().Focused().(*widgets.ChoicePanel); !ok {
		t.Fatalf("focused after canvas Right = %v, want ChoicePanel still focused", w.Window().Canvas().Focused())
	}
}
```

In `confirm_test.go`, after `TestShowConfirmEscapeDoesNotCloseTheEXIFWindow`:

```go
func TestShowConfirmEscapeLeavesCanvasUnfocused(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.showConfirm(confirmation{title: "Title", message: "Message", action: "Confirm"})
	typeKey(t, w.Window(), fyne.KeyEscape)

	if got := w.Window().Canvas().Focused(); got != nil {
		t.Errorf("Focused() after cancelling confirm = %T, want nil so Left/Right reach OnTypedKey", got)
	}
}
```

`TestWindow_ShowLeavesCanvasUnfocused` may already pass on the test driver (it often does not auto-focus a button). Keep it as a regression pin. The tests that must fail before wiring are `TestWindow_ArrowKeysStepImage` and `TestWindow_ArrowKeysIgnoredWhileConfirming` (the latter fails if `handleKey` is missing the `w.confirm != nil` guard and someone wired keys without it).

- [ ] **Step 2: Write failing integration test**

In `internal/ui/exif_test.go`, next to `TestShowExifWindow_ContentAndRefreshOnNavigation`. This file already imports `image/color`, `fyne.io/fyne/v2`, and `internal/uitest`.

```go
func TestExifWindow_LeftRightChangeImage(t *testing.T) {
	v, _, _ := newTestUI(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, a, b)

	v.exif.Show()
	start := v.state.index
	canvas := v.exif.Window().Canvas()
	if got := canvas.Focused(); got != nil {
		t.Fatalf("EXIF canvas focused %T after Show, want nil", got)
	}
	handler := canvas.OnTypedKey()
	if handler == nil {
		t.Fatal("EXIF window has no OnTypedKey handler")
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index = %d, want next file", v.state.index)
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("index = %d, want start %d", v.state.index, start)
	}
}
```

- [ ] **Step 3: Run tests — expect FAIL** (keys not wired)

Run:

```
go test -count=1 -run 'TestWindow_ArrowKeys|TestWindow_UpDownHomeEnd|TestWindow_ShowLeavesCanvasUnfocused' ./internal/ui/exifwin/
go test -count=1 -run TestExifWindow_LeftRightChangeImage ./internal/ui/
```

- [ ] **Step 4: Implement wiring**

In `New`, after `KeepOnTop()`:

```go
	w.win.SetExtraKeys(w.handleKey)
```

Add these methods on `Window` in `exifwin.go` (near `Show` is fine):

```go
func (w *Window) handleKey(ev *fyne.KeyEvent) {
	if w.confirm != nil {
		return
	}
	switch ev.Name {
	case fyne.KeyRight:
		w.host.StepImage(1)
	case fyne.KeyLeft:
		w.host.StepImage(-1)
	}
}

// releaseKeyboard returns Left/Right to Singleton's unfocused OnTypedKey
// handler. Fyne delivers TypedKey only to Canvas.Focused() when it is
// non-nil, and a Button click would otherwise swallow arrows. The Remove
// Metadata confirmation is the exception: ChoicePanel must stay focused so
// arrows move its ring, not the image.
func (w *Window) releaseKeyboard() {
	if w.confirm != nil {
		return
	}
	if win := w.win.Window(); win != nil {
		win.Canvas().Unfocus()
	}
}
```

At the **end** of `Show`, after `w.win.Show(...)` returns (including the already-open raise path), call `w.releaseKeyboard()`.

At the end of **both** branches of `toggleLocation` (expanded and collapsed), call `w.releaseKeyboard()`.

In `confirm.go`, `showConfirm`'s `SetOnClosed`:

```go
	confirm.SetOnClosed(func() {
		w.confirm = nil
		w.releaseKeyboard()
		if c.onClosed != nil {
			c.onClosed()
		}
	})
```

`hideConfirm` does not need its own Unfocus: `Hide()` fires `OnClosed`. Keep `hideConfirm` otherwise unchanged (it must still not nil `w.pending`).

Update the `showConfirm` comment that explains Escape vs focused ChoicePanel: mention that Left/Right on the unfocused handler would call `StepImage`, which is why the panel must stay focused *and* `handleKey` bails when `w.confirm != nil`.

Do **not** handle Up/Down/Home/End. Do **not** call `Refresh` from `handleKey` (`finishLoad` already does). Do **not** overwrite Singleton's `SetOnTypedKey` from `exifwin` (that would drop Escape-close unless you reimplement it). Do **not** wrap the map widget.

- [ ] **Step 5: Run tests**

Run:

```
go test -count=1 ./internal/ui/exifwin/
go test -count=1 -run 'TestExif|TestShowExif|TestStepImage|TestHandleKeyEvent_E|TestStripMetadata' ./internal/ui/
```

Expected: PASS. Existing strip confirmation tests (Right then Return confirms) still pass. `TestShowConfirmEscapeDoesNotCloseTheEXIFWindow` still passes.

- [ ] **Step 6: Parent review (no commit)**

Suggested message: `Navigate with Left/Right while the EXIF window is focused.`

---

### Task 4: Manual and architecture docs

**Subagent:** `generalPurpose` @ `composer-2.5-fast`

**Depends on:** Task 3 accepted.

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md`

No new `lang.L` keys. Manual markdown is not a `lang.L` bundle. Unicode arrows in the manuals **must stay inside code spans** (`TestManualUnicodeArrowsStayInCodeSpans`).

- [ ] **Step 1: English manual**

In `internal/ui/help/manual.md`:

1. Section 7 ("Browsing multiple images"), immediately after the wrap-around paragraph ("Navigation **wraps around**…"), add:

```
While the **EXIF data window** is focused, `←` and `→` do the same next/previous
step (including wrap-around). `Esc` still closes only that window. While
**Remove Metadata** is asking for confirmation, `←`/`→` move the confirmation
choice, not the image.
```

2. In the info-overlay EXIF paragraph (the one that currently says the window updates if you navigate while it's still open), change that sentence so it also names in-window arrows. Keep the rest of the paragraph. The updated sentence:

```
The window updates if you navigate to a different image while it's still open
(from the image window, or with `←`/`→` while the EXIF window itself is focused),
and — like the manual and About windows — `Esc` closes just that window, and
pressing `E` again while it's already open brings it back to the front instead of
opening a second copy.
```

3. Keyboard list, on the **`E`** bullet, append: `While that window is focused, Left/Right change image.`

4. Cheatsheet **EXIF data window** bullet, append: `; while it is focused, `←`/`→` change image`

- [ ] **Step 2: German manual**

Same facts in `internal/ui/help/manual_de.md`. Match existing tone (Sie; existing terms: EXIF-Datenfenster, Metadaten entfernen).

1. After the wrap-around paragraph in the browsing section:

```
Solange das **EXIF-Datenfenster** den Fokus hat, blättern `←` und `→` genau
so weiter (einschließlich im Kreis). `Esc` schließt weiterhin nur dieses
Fenster. Während **Metadaten entfernen** nachfragt, bewegen `←`/`→` die
Bestätigungswahl, nicht das Bild.
```

2. Info-overlay EXIF paragraph: add that `←`/`→` im fokussierten EXIF-Fenster dasselbe tun.

3. On the **`E`** bullet, append: `Solange dieses Fenster den Fokus hat, wechseln Links/Rechts das Bild.`

4. Cheatsheet EXIF bullet, append the same fact with `←`/`→` in code spans.

- [ ] **Step 3: ARCHITECTURE.md**

1. `widgets/` / Singleton: note optional `SetExtraKeys` for unfocused non-Escape keys; only EXIF sets it; Escape still closes.
2. `exifwin/` Host list (the right-hand column of that table row): add `StepImage(delta int)` — next/prev via the viewer, not `Advance`. One sentence in the left-hand column: while the panel is focused, Left/Right call that Host method; confirmation keeps `ChoicePanel` focused so those keys stay on the prompt.
3. `keys.go` row: `StepImage` is the shared arrow-key stepper; `handleKeyEvent` Left/Right/Up/Down (when not tuning the slideshow interval) call it.
4. One sentence: EXIF window keys never reach `handleKeyEvent` because they are a different Fyne window.

Do not rewrite the whole `exifwin/` or `widgets/` rows. Add the minimum sentences.

- [ ] **Step 4: Verify manuals still embed**

Run: `go test -count=1 ./internal/ui/help/`

Expected: PASS, including `TestManualUnicodeArrowsStayInCodeSpans` and `TestManualIsEmbedded`.

- [ ] **Step 5: Parent review (no commit)**

Suggested message: `Document EXIF-window Left/Right image navigation.`

---

## Controller checklist (parent, after all tasks)

1. `make fmt` then `go vet ./...` then `go build ./...`.
2. `go test -count=1 ./internal/ui/widgets/ ./internal/ui/exifwin/ ./internal/ui/help/`
3. `go test -count=1 -run 'TestStepImage|TestHandleKeyEvent|TestExif|TestShowExif|TestStripMetadata' ./internal/ui/`
4. `go test -race -count=1 ./internal/ui/exifwin/ ./internal/ui/` (focused; full `go test -race ./...` before the user commits).
5. Dispatch final `generalPurpose` @ `claude-opus-5-thinking-high` on the whole uncommitted diff.
6. Fix Critical/Important findings with **one** fix subagent, then re-review.
7. Mark the todo in `todos.md` done only after Florian accepts.

## Suggested overall commit message (user, when they ask)

```
Navigate images with Left/Right while the EXIF window is focused.

Arrow keys on the metadata panel were ignored because it is a separate
Fyne window. Route them through Singleton's extra-key hook to the same
StepImage path the main window uses, without stealing keys from the
Remove Metadata confirmation.
```

## Out of scope

- Up/Down/Home/End in the EXIF window (unless D1 is changed).
- Forwarding the rest of `handleKeyEvent` (R, S, I, zoom, …) into the EXIF window.
- Changing `Advance` / shuffle.
- Keyboard pan of the OSM map; reclaiming keys after map zoom-button clicks (D4).
- Windows Ctrl+click grid bug (explicitly not this todo).
