# Folder sibling navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Do not dispatch Task 1 until Florian has confirmed the locked product decisions (or explicitly said to proceed with the defaults).

**Goal:** Opening a single image (drop, file picker, CLI / Open With, session restore) loads the other images in that file's parent directory so Left/Right (and the rest of existing navigation) walk the folder, while still showing the opened file first.

**Architecture:** Add `filescan.Siblings` — a non-recursive parent-directory listing that always keeps the opened file in the result. `handleDrop` calls it instead of `filescan.Images` when the drop is exactly one supported image file and merge mode is not adding to an existing set. `applyScannedFiles` then keeps that file on screen after the sort, instead of `ShowImage(0)`. Arrow keys stay `StepImage`; they start working because `len(v.state.files)` becomes ≥ 2. No new lifecycle, no new overlay, no new `lang.L` keys.

**Tech Stack:** Go 1.26, Fyne `storage.Parent` / `storage.List` / `storage.CanList`, existing `filescan` + `filesort` + `viewer.handleDrop` / `StepImage` / `dropAndWait`.

## Why this approach

Today `handleKeyEvent` and `StepImage` both no-op when `len(v.state.files) < 2`. Dropping `DSC_0042.jpg` therefore shows that one file and ignores arrows. Preview, Windows Photos, and feh all treat “open this file” as “open this folder, parked on this file”. PicFetch already knows how to scan, sort, and step a set — it just never gathers the siblings.

Alternatives rejected:

- **Rewrite the drop as the parent folder.** `filescan.Images` recurses into subfolders and `applyScannedFiles` starts at index 0, so the user would land on the wrong file and see nested albums they did not open.
- **Expand lazily on the first arrow.** The first Right would stall on a directory listing, `dropAndWait` would return before the set grew (flaky tests unless the harness grows a second wait), and `G` / picture-frame would still see a 1-file set until the user arrows.
- **Two-phase apply (show the file, then replace the set).** Two `scanOp` generations, two sorts, two `ShowImage`s, and every existing `dropAndWait` caller would race the second apply. Not worth it for avoiding a brief scan overlay.

One scan generation, one sort, one `ShowImage` of the opened file. `dropAndWait` keeps working.

## Locked product decisions

These are the defaults. Change them only if Florian says so before Task 1.

1. **What “selected or dropped” means.** A replace-mode drop / `Cmd+O` pick / CLI `picfetch photo.jpg` / session restore of **exactly one supported image file**. File picker and CLI already call `handleDrop`, so they come along for free. This is **not** grid selection of one thumbnail among many (that set is already loaded; arrows already work).
2. **Same folder only.** List the parent directory. Do **not** recurse into subdirectories. Dropping the folder itself still uses recursive `Images`.
3. **When to gather.** During the same `handleDrop` scan, before the first `ShowImage`. Single-file expansion uses the **background goroutine + existing scan overlay** (Parent+List of a large Pictures folder must not run on the UI thread). Multi-file drops with no directories stay on the synchronous fast path.
4. **Keep the opened file on screen.** After sort, `showFileIfPresent(droppedURI)`, not `ShowImage(0)`. Default name-sort of `c.jpg` + `a.jpg` + `b.jpg` with `b.jpg` dropped shows `b.jpg` (index 1), not `a.jpg`.
5. **Do not expand a subset.** Two or more files in one drop stay exactly those files, even if they share a parent. Merge-mode adding one file onto a non-empty set adds only that file. Deleting down to one file does not rescan the folder.
6. **Do not expand a non-image.** Dropping `notes.txt` in a photo folder still errors; it must not load the siblings. Gate: `imaging.IsSupportedImage(uris[0])`.
7. **Opened file survives the cap.** `Siblings` seeds the result with the opened file, then fills remaining slots from the listing. `maxScan` and the existing truncation toast still apply.
8. **ByDropOrder** after a sibling expand is: opened file first, then the other siblings in `storage.List` order. Other sort modes reorder as they already do. The opened file stays on screen across that reorder (decision 4).
9. **All existing navigation** (Left/Right/Up/Down, Home/End, EXIF-window arrows, picture-frame auto-advance, `G` grid) walks the expanded set. No special-case “only Left/Right”.
10. **No new `lang.L` strings.** Reuse `"Scanning... %d images"` and the existing truncation toast. Manual EN/DE + README + `ARCHITECTURE.md` get the behavior change.
11. **No golden screenshot regeneration.**
12. **Out of scope:** hide-duplicates “highest resolution” and Shift+D variant-loop todos. Do not start them.

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Every user-visible string is `lang.L("English text")` with the same key in every `translations/*.json` bundle. This feature adds none.
- `internal/filescan` stays viewer-independent (no Fyne widgets, no `internal/ui` import). `handleDrop` is the only caller of `Siblings`.
- Do not pass `appState` into feature packages. This work stays in `filescan` + `internal/ui` drop/apply.
- Every goroutine needs cancellation/staleness plus an observable done signal. Sibling listing runs under the existing `scanOp` token/context/`done` — do **not** add a second lifecycle. `drain` already waits `scanOp`.
- Drive tests with `dropAndWait` / `waitForScan` / `waitUntilLoaded`. Never `time.Sleep` to guess completion.
- `t.TempDir()` in `uitest.TempJPEGURI` is a **fresh directory per call**. Existing one-file tests will not magically pick up siblings. Tests that need siblings must write several files into **one** directory.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Update `ARCHITECTURE.md` in the same change that adds `Siblings` (Task 4 may do the doc pass; Task 1 must not leave a new exported function undocumented across the whole branch — Task 4 is required before “done”).
- Subagents must not start Task N+1 themselves. They stop after their task's verification and report.
- Do not “fix” Windows Ctrl+click. Out of scope.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.5-high-fast` | Pure `filescan` function + tests; complete code in the brief. |
| Task 2 implementer | `cursor-grok-4.6-xhigh` | `handleDrop` fast-path vs goroutine, merge/replace, keep-file after sort. If `BLOCKED` on scan/sort races, re-dispatch this task only with `claude-opus-5-thinking-high`. |
| Task 3 implementer | `cursor-grok-4.6-xhigh` | Viewer navigation + harness waits; easy to flake if `dropAndWait` is skipped. |
| Task 4 implementer | `cursor-grok-4.5-high-fast` | Manual / README / ARCHITECTURE / todos copy. |
| Task reviewer (Tasks 1, 4) | `cursor-grok-4.5-high-fast` | Mid-tier floor. |
| Task reviewer (Tasks 2, 3) | `cursor-grok-4.6-xhigh` | Scan/sort/index bugs are easy to miss. |
| Parent review / fix after each task | this session (do not dispatch) | Review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | Cross-task: lonely file still no-ops, folder drop still recurses, opened file stays on screen. |

Subagent type: `generalPurpose` for implementers and reviewers. Do not use `go-expert` to write the code (it is for design questions). Do not dispatch two implementers in parallel.

## File structure

- Modify: `internal/filescan/filescan.go` — add `Siblings`; do **not** change `Images` behavior
- Modify: `internal/filescan/filescan_test.go` — `Siblings` tests
- Modify: `internal/uitest/uitest.go` — `TempDirJPEGURIs` helper (Task 2)
- Modify: `internal/ui/drop.go` — `handleDrop` sibling branch; `applyScannedFiles` keep-opened-file
- Modify: `internal/ui/drop_test.go` — expand / don't-expand / keep-file / non-image / truncation
- Modify: `internal/ui/step_test.go` — arrows / Home / End on an expanded single-file drop; lonely file still no-ops
- Modify: `internal/ui/keys.go` — comment on the `< 2` guard (behavior of the guard stays)
- Modify: `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`
- Modify: `README.md` — Features bullet about arrows
- Modify: `ARCHITECTURE.md` — `filescan` row + drop.go row + “Where to look”
- Modify: `todos.md` — point this item at this plan (do not move it to Done until the branch is accepted)

Do not add `internal/ui/siblings.go`. Do not add a new `asyncOpUI`. Do not split `drop.go`.

## Current code the implementers must not break

`internal/ui/keys.go` (unchanged logic):

```go
if len(v.state.files) < 2 || v.loading.Load() {
    return
}
```

`internal/ui/viewer.go` `StepImage` has the same `< 2` guard. After a sibling expand the guard is simply false. After a genuinely lonely file it stays true. `TestStepImage_NoopWithOneFile` must remain green.

`handleDrop` today: no directories → synchronous `filescan.Images` + `fyne.Do(applyScanResult)`. Directories → goroutine + progress overlay. After this plan: **single supported image, not merging** also takes the goroutine, calling `Siblings` instead of `Images`. Two-or-more loose files stay synchronous.

`applyScannedFiles` today always `ShowImage(0)` on replace. That is wrong once a single file has been expanded and then name-sorted.

`uitest.TempJPEGURI` / `WriteTempFile` call `t.TempDir()` per file, and multiple `t.TempDir()` calls return **different** directories. One-file tests do not need changing to “stay lonely”.

---

### Task 1: `filescan.Siblings`

**Files:**
- Modify: `internal/filescan/filescan.go`
- Test: `internal/filescan/filescan_test.go`

**Interfaces:**
- Consumes: `realPathOf`, `imaging.IsSupportedImage`, `storage.Parent`, `storage.List`, `storage.CanList`, `DefaultMax` floor-at-1 rule, progress throttle (`n == 1`, every 10th, truncation)
- Produces: `func Siblings(ctx context.Context, file fyne.URI, max int, progress func(n int)) (images []fyne.URI, truncated bool)`
  - Non-recursive listing of `file`'s parent directory
  - If `file` is a supported image, it is **always** in `images` (seeded first) unless `ctx` is already cancelled before any work
  - When `file` is in the listing under a different URI string but the same `realPathOf`, the result contains the **caller's** `file` URI, not the listed one (so `showFileIfPresent` can match on `URI.String()`)
  - Directories among the children are skipped (not descended)
  - `max < 1` floors to 1, same as `Images`
  - `truncated` is true iff gathering stopped because `count >= max` (not because of cancel)
  - On `Parent`/`List` error, or a non-listable parent: return `[]fyne.URI{file}` if `file` is a supported image, else nil; `truncated == false`
  - `progress` nil-safe; same throttle as `Images`

Do **not** refactor `Images` into a shared walker. Duplicate the small collect-file loop. Do **not** change any existing `TestImages_*` assertion.

- [ ] **Step 1: Write the failing tests**

Append to `internal/filescan/filescan_test.go`. Helper local to this file:

```go
func writeJPEG(t *testing.T, dir, name string) fyne.URI {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, uitest.EncodeJPEG(t, 4, 4, color.White), 0o644); err != nil {
		t.Fatal(err)
	}
	return storage.NewFileURI(path)
}
```

`TestSiblings_ListsSameDirectoryOnly`:

```go
func TestSiblings_ListsSameDirectoryOnly(t *testing.T) {
	root := t.TempDir()
	opened := writeJPEG(t, root, "b.jpg")
	writeJPEG(t, root, "a.jpg")
	writeJPEG(t, root, "c.jpg")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJPEG(t, nested, "nested.jpg")

	images, truncated := Siblings(context.Background(), opened, DefaultMax, nil)
	if truncated {
		t.Fatal("truncated = true, want false")
	}
	if len(images) != 3 {
		t.Fatalf("images = %d, want 3 (a, b, c) — not notes.txt, not nested.jpg, not the sub dir", len(images))
	}
	if images[0].String() != opened.String() {
		t.Fatalf("images[0] = %q, want the opened URI identity so showFileIfPresent can find it", images[0])
	}
	names := make([]string, len(images))
	for i, u := range images {
		names[i] = u.Name()
	}
	if !slices.Contains(names, "a.jpg") || !slices.Contains(names, "c.jpg") {
		t.Fatalf("names = %v, want a.jpg and c.jpg among siblings", names)
	}
	if slices.Contains(names, "nested.jpg") || slices.Contains(names, "notes.txt") {
		t.Fatalf("names = %v, must not include nested.jpg or notes.txt", names)
	}
}
```

Do **not** call `Siblings` with `uitest.FakeURI`. Its scheme is `file` and `Path()` is `/name`; `storage.Parent` can succeed and `storage.List` can then walk the process root. Use real temp files, or a file URI whose parent does not exist (List fails; the seeded opened file is still returned).

`TestSiblings_ListFailureReturnsOpenedFile`:

```go
func TestSiblings_ListFailureReturnsOpenedFile(t *testing.T) {
	// Parent exists as a URI but List fails because the directory was never created.
	opened := storage.NewFileURI(filepath.Join(t.TempDir(), "missing-dir", "photo.jpg"))
	images, truncated := Siblings(context.Background(), opened, DefaultMax, nil)
	if truncated {
		t.Fatal("truncated = true, want false")
	}
	if len(images) != 1 || images[0].String() != opened.String() {
		t.Fatalf("images = %v, want just the opened URI after List fails", images)
	}
}
```

`TestSiblings_LonelyFile`:

```go
func TestSiblings_LonelyFile(t *testing.T) {
	opened := uitest.TempJPEGURI(t, "solo.jpg", 4, 4, color.White)
	images, truncated := Siblings(context.Background(), opened, DefaultMax, nil)
	if truncated || len(images) != 1 {
		t.Fatalf("images = %d, truncated = %v, want 1 and not truncated", len(images), truncated)
	}
	if images[0].String() != opened.String() {
		t.Fatalf("images[0] = %q, want opened URI %q", images[0], opened)
	}
}
```

`TestSiblings_CapsAtMaxKeepsOpenedFile`:

```go
func TestSiblings_CapsAtMaxKeepsOpenedFile(t *testing.T) {
	root := t.TempDir()
	opened := writeJPEG(t, root, "opened.jpg")
	for i := range 5 {
		writeJPEG(t, root, fmt.Sprintf("photo%d.jpg", i))
	}
	images, truncated := Siblings(context.Background(), opened, 3, nil)
	if !truncated || len(images) != 3 {
		t.Fatalf("images = %d, truncated = %v, want 3 and truncated", len(images), truncated)
	}
	if images[0].String() != opened.String() {
		t.Fatalf("images[0] = %q, want opened file even when the cap is hit", images[0])
	}
}
```

`TestSiblings_MaxFlooredAtOne`:

```go
func TestSiblings_MaxFlooredAtOne(t *testing.T) {
	root := t.TempDir()
	opened := writeJPEG(t, root, "opened.jpg")
	writeJPEG(t, root, "other.jpg")
	images, truncated := Siblings(context.Background(), opened, 0, nil)
	if len(images) != 1 || !truncated {
		t.Fatalf("images = %d, truncated = %v, want 1 (the opened file) and truncated", len(images), truncated)
	}
	if images[0].String() != opened.String() {
		t.Fatalf("images[0] = %q, want opened", images[0])
	}
}
```

`TestSiblings_AlreadyCancelledContext`:

```go
func TestSiblings_AlreadyCancelledContext(t *testing.T) {
	root := t.TempDir()
	opened := writeJPEG(t, root, "opened.jpg")
	writeJPEG(t, root, "other.jpg")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	images, truncated := Siblings(ctx, opened, DefaultMax, nil)
	if len(images) != 0 || truncated {
		t.Fatalf("images = %d, truncated = %v, want nothing from an already-cancelled context", len(images), truncated)
	}
}
```

`TestSiblings_ProgressThrottle`:

```go
func TestSiblings_ProgressThrottle(t *testing.T) {
	root := t.TempDir()
	opened := writeJPEG(t, root, "photo00.jpg")
	for i := 1; i < 25; i++ {
		writeJPEG(t, root, fmt.Sprintf("photo%02d.jpg", i))
	}

	t.Run("throttled to the first and every 10th call", func(t *testing.T) {
		var calls []int
		images, truncated := Siblings(context.Background(), opened, 1000, func(n int) {
			calls = append(calls, n)
		})
		if len(images) != 25 || truncated {
			t.Fatalf("images = %d, truncated = %v, want 25, not truncated", len(images), truncated)
		}
		want := []int{1, 10, 20}
		if !slices.Equal(calls, want) {
			t.Errorf("progress calls = %v, want %v", calls, want)
		}
	})

	t.Run("truncation forces a final call off the every-10th cadence", func(t *testing.T) {
		var calls []int
		images, truncated := Siblings(context.Background(), opened, 13, func(n int) {
			calls = append(calls, n)
		})
		if len(images) != 13 || !truncated {
			t.Fatalf("images = %d, truncated = %v, want 13, truncated", len(images), truncated)
		}
		want := []int{1, 10, 13}
		if !slices.Equal(calls, want) {
			t.Errorf("progress calls = %v, want %v", calls, want)
		}
	})

	t.Run("nil progress is never called and never panics", func(t *testing.T) {
		images, truncated := Siblings(context.Background(), opened, DefaultMax, nil)
		if len(images) != 25 || truncated {
			t.Fatalf("images = %d, truncated = %v, want 25, not truncated", len(images), truncated)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestSiblings_' ./internal/filescan/`

Expected: FAIL compile (`Siblings` undefined) or FAIL assertions.

- [ ] **Step 3: Write `Siblings`**

Add to `internal/filescan/filescan.go` after `Images`:

```go
// Siblings returns the supported images that share file's parent directory.
// It does not recurse into subdirectories. If file itself is a supported
// image it is always the first entry — the caller's URI, not a possibly
// different URI storage.List produced for the same path — so a caller that
// looks the opened file up by URI.String() still finds it after a sort.
// Directories among the children are skipped. max is floored at 1, the same
// as Images; truncated means the listing stopped at max rather than
// exhausting the directory. On Parent/List failure the result is just file
// when it is a supported image, otherwise empty. ctx is checked before any
// work and before each child; an already-cancelled context returns nil,
// false rather than a partial directory.
func Siblings(ctx context.Context, file fyne.URI, max int, progress func(n int)) (images []fyne.URI, truncated bool) {
	if max < 1 {
		max = 1
	}
	if ctx.Err() != nil {
		return nil, false
	}

	origin := realPathOf(file)
	seen := make(map[string]bool)
	count := 0
	add := func(u fyne.URI) {
		if truncated || ctx.Err() != nil {
			return
		}
		if canList, err := storage.CanList(u); err == nil && canList {
			return
		}
		if !imaging.IsSupportedImage(u) {
			return
		}
		pathOf := realPathOf(u)
		if seen[pathOf] {
			return
		}
		seen[pathOf] = true
		if pathOf == origin {
			u = file
		}
		images = append(images, u)
		count++
		if count >= max {
			truncated = true
		}
		if progress != nil && (count == 1 || count%10 == 0 || truncated) {
			progress(count)
		}
	}

	if imaging.IsSupportedImage(file) {
		add(file)
	}

	parent, err := storage.Parent(file)
	if err != nil {
		return images, truncated
	}
	children, err := storage.List(parent)
	if err != nil {
		return images, truncated
	}
	for _, child := range children {
		add(child)
	}
	return images, truncated
}
```

If `file` is a supported image, `add(file)` runs first so a `max == 1` result is always the opened file and `truncated` is true when other images exist. Do not call `add(file)` a second time from the listing — `seen[origin]` skips it, and if List produced a different URI the `pathOf == origin` branch would rewrite to `file` anyway; seeding first is what keeps it at index 0.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 -run 'TestSiblings_|TestImages_' ./internal/filescan/`

Expected: PASS. Every existing `TestImages_*` still passes (Images is untouched).

- [ ] **Step 5: Do not commit**

Report the test command and output. Status: DONE / DONE_WITH_CONCERNS / BLOCKED.

---

### Task 2: `handleDrop` expands a single-file replace; keep the opened file

**Files:**
- Create helper in: `internal/uitest/uitest.go` — `TempDirJPEGURIs`
- Modify: `internal/ui/drop.go` — `handleDrop`, `applyScannedFiles`
- Test: `internal/ui/drop_test.go`

**Interfaces:**
- Consumes: `filescan.Siblings` from Task 1 (`Siblings(ctx, file, max, progress)`)
- Produces: replace-mode drop of one supported image file → file set is that file's siblings (non-recursive), current index is the opened file. `dropAndWait` still waits out the one `scanOp` + `sortOp` + load.

Do not change `StepImage` in this task. Do not edit the manual.

- [ ] **Step 1: Add `TempDirJPEGURIs`**

In `internal/uitest/uitest.go`, next to `TempJPEGURI`:

```go
// TempDirJPEGURIs writes solid-color 8×8 white JPEGs named names into a
// single temp directory and returns their file URIs in the same order.
// TempJPEGURI cannot be used for sibling tests: each call uses its own
// t.TempDir(), so the files would not share a parent.
func TempDirJPEGURIs(t *testing.T, names ...string) []fyne.URI {
	t.Helper()
	dir := t.TempDir()
	out := make([]fyne.URI, len(names))
	for i, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, EncodeJPEG(t, 8, 8, color.White), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		out[i] = storage.NewFileURI(path)
	}
	return out
}
```

`EncodeJPEG` and `storage` are already imported in this file.

- [ ] **Step 2: Write the failing drop tests**

Append to `internal/ui/drop_test.go`.

`TestHandleDrop_SingleFileExpandsSiblingsAndKeepsOpened`:

```go
func TestHandleDrop_SingleFileExpandsSiblingsAndKeepsOpened(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "c.jpg", "a.jpg", "b.jpg")
	// Drop b.jpg only. Name-sort order is a, b, c — ShowImage(0) would
	// wrongly land on a.jpg.
	var opened fyne.URI
	for _, u := range files {
		if u.Name() == "b.jpg" {
			opened = u
			break
		}
	}
	dropAndWait(t, v, opened)

	if n := len(v.state.files); n != 3 {
		t.Fatalf("files = %d, want 3 siblings", n)
	}
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("showing %q at index %d, want b.jpg (the opened file, not the first name-sort entry)", v.state.files[v.state.index].Name(), v.state.index)
	}
}
```

`TestHandleDrop_SingleFileDoesNotRecurse`:

```go
func TestHandleDrop_SingleFileDoesNotRecurse(t *testing.T) {
	v := newTestViewer(t)
	root := t.TempDir()
	openedPath := filepath.Join(root, "top.jpg")
	if err := os.WriteFile(openedPath, uitest.EncodeJPEG(t, 4, 4, color.White), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "nested.jpg"), uitest.EncodeJPEG(t, 4, 4, color.White), 0o644); err != nil {
		t.Fatal(err)
	}
	dropAndWait(t, v, storage.NewFileURI(openedPath))
	if n := len(v.state.files); n != 1 {
		t.Fatalf("files = %d, want 1 (nested.jpg is in a subdirectory)", n)
	}
}
```

`TestHandleDrop_TwoFilesInSameDirDoNotExpand`:

```go
func TestHandleDrop_TwoFilesInSameDirDoNotExpand(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	dropAndWait(t, v, files[0], files[1]) // a and b, not c
	if n := len(v.state.files); n != 2 {
		t.Fatalf("files = %d, want 2 (explicit subset, not the whole folder)", n)
	}
}
```

`TestHandleDrop_MergeSingleFileDoesNotExpandSiblings`:

```go
func TestHandleDrop_MergeSingleFileDoesNotExpandSiblings(t *testing.T) {
	v := newTestViewer(t)
	existing := uitest.TempJPEGURI(t, "keep.jpg", 4, 4, color.White)
	dropAndWait(t, v, existing)
	v.SetMergeMode(true)
	sibs := uitest.TempDirJPEGURIs(t, "x.jpg", "y.jpg", "z.jpg")
	var one fyne.URI
	for _, u := range sibs {
		if u.Name() == "y.jpg" {
			one = u
			break
		}
	}
	dropAndWait(t, v, one)
	if n := len(v.state.files); n != 2 {
		t.Fatalf("files = %d, want 2 (existing + merged y.jpg), not the whole sibling folder", n)
	}
}
```

`TestHandleDrop_UnsupportedSingleFileDoesNotExpandFolder`:

```go
func TestHandleDrop_UnsupportedSingleFileDoesNotExpandFolder(t *testing.T) {
	v := newTestViewer(t)
	dir := t.TempDir()
	txt := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txt, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), uitest.EncodeJPEG(t, 4, 4, color.White), 0o644); err != nil {
		t.Fatal(err)
	}
	dropAndWaitScan(t, v, storage.NewFileURI(txt))
	if n := len(v.state.files); n != 0 {
		t.Fatalf("files = %d, want 0 — dropping a non-image must not load sibling photos", n)
	}
}
```

`TestHandleDrop_SiblingScanTruncationToast`:

```go
func TestHandleDrop_SiblingScanTruncationToast(t *testing.T) {
	v := newTestViewer(t)
	v.settings.maxScan = 2
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	dropAndWait(t, v, files[0])
	if n := len(v.state.files); n != 2 {
		t.Fatalf("files = %d, want 2 (maxScan)", n)
	}
	if !v.toast.card.Visible() {
		t.Fatal("want a toast warning that the scan was truncated")
	}
	if !strings.Contains(v.toast.text.Text, "2") {
		t.Errorf("toast text = %q, want it to mention the cap (2)", v.toast.text.Text)
	}
	settleToast(t, v)
}
```

Existing `TestHandleDrop_RecursesIntoNestedDirectories` must stay green (folder drop still uses `Images`).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestHandleDrop_SingleFile|TestHandleDrop_TwoFilesInSameDir|TestHandleDrop_MergeSingleFile|TestHandleDrop_UnsupportedSingleFile|TestHandleDrop_SiblingScan' ./internal/ui/`

Expected: FAIL (`files = 1, want 3` on the expand test).

- [ ] **Step 4: Wire `handleDrop` and `applyScannedFiles`**

In `drop.go`, add the imaging import:

```go
"github.com/frathe/picfetch/internal/imaging"
```

Replace the `hasDirs` / fast-path / goroutine block (from `hasDirs := false` through the goroutine's closing `}()`) with:

```go
	hasDirs := false
	for _, u := range uris {
		if canList, err := storage.CanList(u); err == nil && canList {
			hasDirs = true
			break
		}
	}

	expandSiblings := !merging && !hasDirs && len(uris) == 1 && imaging.IsSupportedImage(uris[0])

	scan := func(progress func(int)) (images []fyne.URI, truncated bool) {
		if expandSiblings {
			return filescan.Siblings(token.context(), uris[0], maxScan, progress)
		}
		return filescan.Images(token.context(), uris, maxScan, progress)
	}

	if !hasDirs && !expandSiblings {
		// nil progress: this path is synchronous and instantaneous, so
		// there's nothing to show, and it avoids calling fyne.Do from the
		// UI goroutine. Multi-file loose drops and merge-mode single
		// files stay here; sibling expansion of a possibly large
		// directory does not — that listing belongs on the goroutine
		// below, same as a folder drop.
		images, truncated := scan(nil)
		fyne.Do(func() {
			v.applyScanResult(token, merging, uris, images, truncated, maxScan, scanDone)
		})
		return
	}

	go func() {
		images, truncated := scan(func(n int) {
			fyne.Do(func() {
				if !token.current() {
					return
				}
				v.scanOp.label.SetText(fmt.Sprintf(lang.L("Scanning... %d images"), n))
			})
		})

		fyne.Do(func() {
			v.applyScanResult(token, merging, uris, images, truncated, maxScan, scanDone)
		})
	}()
```

Update `applyScannedFiles`'s `startSort` onDone replace branch. Current:

```go
		if merging {
			if !v.showFileIfPresent(images[0]) {
				v.ShowImage(0)
			}
		} else {
			v.ShowImage(0)
		}
```

Change to:

```go
		if merging {
			if !v.showFileIfPresent(images[0]) {
				v.ShowImage(0)
			}
			return
		}
		// A single-file replace that was expanded to the parent directory
		// must stay parked on the file the user opened, not jump to the
		// first name-sorted sibling.
		if len(uris) == 1 && imaging.IsSupportedImage(uris[0]) && v.showFileIfPresent(uris[0]) {
			return
		}
		v.ShowImage(0)
```

Thread `uris` into `applyScannedFiles`. Today the signature is `applyScannedFiles(merging bool, images []fyne.URI)` and the only caller is `applyScanResult`, which already has `uris`. Change to:

```go
func (v *viewer) applyScannedFiles(merging bool, images, dropped []fyne.URI) {
```

and pass `uris` from `applyScanResult`:

```go
	v.applyScannedFiles(merging, images, uris)
```

Use `dropped` (not a captured `uris` from handleDrop) inside onDone. Update the keep-file check to read `dropped` instead of `uris`.

Extend `applyScannedFiles`'s doc comment: on a non-merge drop of one supported image, show that image after the reorder rather than index 0.

A folder drop has `dropped[0]` as a directory (`IsSupportedImage` false) → `ShowImage(0)` unchanged.

- [ ] **Step 5: Run the new tests and the existing drop/step suite**

Run:

```
go test -count=1 -run 'TestHandleDrop_|TestStepImage_NoopWithOneFile' ./internal/ui/
```

Expected: PASS. `TestStepImage_NoopWithOneFile` still no-ops (lonely `TempJPEGURI`). Folder recursion test still sees nested photos.

- [ ] **Step 6: Do not commit**

Report commands and output.

---

### Task 3: Arrow keys walk the expanded folder

**Files:**
- Modify: `internal/ui/step_test.go`
- Modify: `internal/ui/keys.go` — comments only (the `< 2` guard stays)

**Interfaces:**
- Consumes: Task 2's `handleDrop` sibling expand + `uitest.TempDirJPEGURIs`
- Produces: Left/Right (via `handleKeyEvent` and `StepImage`) move to the previous/next sibling in the current sort order and wrap. Home/End jump to first/last. A genuinely single-file directory still no-ops. Picture-frame `Advance` walks siblings once the set has 2+.

- [ ] **Step 1: Write the failing tests**

In `internal/ui/step_test.go`, add:

```go
func TestStepImage_SingleFileDropWalksFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "c.jpg", "a.jpg", "b.jpg")
	var opened fyne.URI
	for _, u := range files {
		if u.Name() == "b.jpg" {
			opened = u
			break
		}
	}
	dropAndWait(t, v, opened)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("setup: showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("after StepImage(1) showing %q, want c.jpg (name-sort a,b,c)", v.state.files[v.state.index].Name())
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("after wrap showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}

	v.StepImage(-1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("after StepImage(-1) showing %q, want c.jpg", v.state.files[v.state.index].Name())
	}
}

func TestHandleKeyEvent_LeftRightWalkFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
	dropAndWait(t, v, files[0])
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("Right showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("Left showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}
}

func TestHandleKeyEvent_HomeEndOnFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	var opened fyne.URI
	for _, u := range files {
		if u.Name() == "b.jpg" {
			opened = u
			break
		}
	}
	dropAndWait(t, v, opened)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("End showing %q, want c.jpg", v.state.files[v.state.index].Name())
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyHome})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("Home showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}
}
```

Keep `TestStepImage_NoopWithOneFile` as-is; it is the lonely-file regression.

`TestAdvance_SingleFileDropWalksSiblings` — `Advance` is the same next-file path picture-frame uses, and must walk siblings once the set has 2+ files:

```go
func TestAdvance_SingleFileDropWalksSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
	dropAndWait(t, v, files[0])
	v.Advance()
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("Advance showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestStepImage_SingleFileDrop|TestHandleKeyEvent_LeftRightWalkFolder|TestHandleKeyEvent_HomeEndOnFolder|TestAdvance_SingleFileDrop' ./internal/ui/`

If Task 2 is done they may already **pass** without further production code — that is OK. If they fail, the production gap is in Task 2 (keep-file / expand), not a new `StepImage` API. Do **not** weaken `StepImage`'s `< 2` guard.

- [ ] **Step 3: Comment-only update in `keys.go`**

The navigation guard comment currently reads as if two dropped files are required forever. Update the comment above `len(v.state.files) < 2` (handleKeyEvent) to:

```go
	// Ignore repeat events fired while the previous image is still
	// decoding/rendering, instead of piling up decodes for images the
	// user has already navigated past. A single-file drop that found
	// siblings in the same folder has already expanded the set (see
	// handleDrop); a genuinely lonely file still no-ops here.
	if len(v.state.files) < 2 || v.loading.Load() {
		return
	}
```

Same idea on `StepImage`'s doc comment in `viewer.go`: mention that a single-file drop may already have been expanded to the parent directory by `handleDrop`, so this guard is “fewer than two files in the current set”, not “the user dropped only one path”.

- [ ] **Step 4: Run the step + drop tests**

Run:

```
go test -count=1 -run 'TestStepImage_|TestHandleKeyEvent_LeftRight|TestHandleKeyEvent_HomeEnd|TestAdvance_SingleFileDrop' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Do not commit**

---

### Task 4: Docs, architecture map, todos pointer

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `README.md`
- Modify: `ARCHITECTURE.md`
- Modify: `todos.md`

**Interfaces:**
- Consumes: behavior from Tasks 1–3
- Produces: user-facing copy that no longer says arrows require dropping two files; `ARCHITECTURE.md` names `Siblings`; `todos.md` points at this plan and stays under TODO until Florian accepts the branch

No new translation keys. `main_test.go` locale parity is therefore unchanged — do not add strings.

- [ ] **Step 1: Manual EN**

In `internal/ui/help/manual.md`, replace the note (around “The arrow keys only do something when you dropped **two or more** images”) with:

```markdown
- The arrow keys walk the current set and wrap around. If you opened
  **one image file** (a drop, the file picker, or `picfetch photo.jpg`),
  PicFetch also loads the other images in that file's folder — not
  subfolders — and parks on the file you opened, so Left/Right move to
  its neighbors. Opening two or more files keeps exactly that subset.
  Opening a folder still scans it recursively (see "Scanning folders").
  A folder that contains only that one image still has nothing to step
  to. While hide-duplicates is on (`D`, see "Grid overview"), arrows
  skip hidden extras and wrap among the remaining files; `Home`/`End`
  jump to the first and last remaining file.
```

Do not use Unicode arrows outside backticks (`manual_test.go` forbids it). “Left/Right” as words is fine.

Also update Getting Started if it implies dropping one file never becomes a set — only if a sentence is now false. Do not rewrite the whole manual.

- [ ] **Step 2: Manual DE**

In `internal/ui/help/manual_de.md`, replace the matching “zwei oder mehr” note with the same meaning (folder siblings, no subfolders, subset of two+ files unchanged, folder drop still recursive). Keep the German manual's tone. No Unicode arrows outside backticks.

- [ ] **Step 3: README**

`README.md` Features currently: “Drop multiple files at once and step through them with the arrow keys”. Change to a sentence that also covers the single-file folder case, e.g.:

```markdown
- Drop one image to step through the other images in the same folder
  with the arrow keys, or drop several files / a folder to walk that set
```

Keep it one bullet. Do not add marketing fluff.

- [ ] **Step 4: ARCHITECTURE.md**

`internal/ui` table, `drop.go` row — change the `filescan.Images` mention to `filescan.Images` / `filescan.Siblings`.

`internal/filescan` section:

```markdown
### `internal/filescan`

Recursive image gather for drop/open, plus a non-recursive sibling listing
when the user opened a single file.

| File | Responsibility |
|------|----------------|
| `filescan.go` | `Images(ctx, uris, max, progress)` (recursive); `Siblings(ctx, file, max, progress)` (parent dir only, opened file seeded first); symlink-cycle + per-call dedupe. |
```

Where to look: “How does drag-and-drop / folder scanning work?” → add `filescan.Siblings` for the single-file case.

- [ ] **Step 5: todos.md**

Under TODO, keep the item (do not move to Done). Add a pointer:

```markdown
- when only a single image is selected or dropped
  the left and right arrow keys should move to the next/previous image in the same folder.
  Plan: `docs/superpowers/plans/2026-08-26-folder-siblings-nav.md`
```

- [ ] **Step 6: Verify docs tests**

Run: `go test -count=1 -run 'TestManual' ./internal/ui/help/`

Expected: PASS (no Unicode arrows introduced).

- [ ] **Step 7: Do not commit**

---

## Execution notes for the controller

1. Do not dispatch Task 1 until Florian confirms the locked decisions or says to proceed with the defaults.
2. After each task: parent-review the diff, fix if needed, then dispatch the next implementer.
3. Task 2 is the only task allowed to use Opus, and only after `BLOCKED` on scan/sort races with `cursor-grok-4.6-xhigh`.
4. Suggested commit message (when Florian asks to commit, not before):

```
feat: walk same-folder siblings when a single image is opened

Opening one file now lists the parent directory (non-recursive) so
arrow keys move to the next/previous image, parked on the file that
was opened.
```

## Self-review (plan vs spec)

1. **Spec coverage:** Single-file drop/picker/CLI/session → siblings (Tasks 2, handleDrop is the shared path). Arrows (Task 3). Same folder not recursive (Task 1 + Task 2 no-recurse test). Keep opened file (Task 2). Subset / merge / non-image (Task 2). Cap + toast (Tasks 1–2). Docs (Task 4). Lonely file still no-ops (existing test, Task 3). Folder drop still recursive (existing test).
2. **Placeholders:** None. Parent/List failure is `TestSiblings_ListFailureReturnsOpenedFile` (missing directory, not `FakeURI`). Truncation toast copies `TestHandleDrop_TruncatedScanToastNamesTheCap` (`v.toast.card` / `v.toast.text.Text`).
3. **Type consistency:** `Siblings(ctx, file, max, progress) (images []fyne.URI, truncated bool)` throughout. `applyScannedFiles(merging, images, dropped []fyne.URI)`. `expandSiblings` is handleDrop-local; applyScannedFiles derives keep-file from `dropped`.
