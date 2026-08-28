# `viewer` field-cluster extraction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use `superpowers:subagent-driven-development`
> to execute this plan stage-by-stage. Steps use checkbox (`- [ ]`) syntax.
>
> **Controller extra (this session):** after every stage the parent agent reviews the
> subagent's diff line by line, fixes it up itself, runs that stage's verification
> commands, and then **stops** and hands Florian a suggested commit message. Do not
> dispatch Stage N+1 until Florian has confirmed the commit landed. Do not run
> `git commit` (`AGENTS.md`).

**Goal:** Break four field clusters out of the 87-field `viewer` god object into
four viewer-independent subpackages, and collapse the push-based menu
synchronisation into one recompute entry point.

Closes `todos.md` "The `viewer` god object — 87 fields and still growing"
(priority 14) plus `needs_refactoring.md` items **5** (menu Checked/Disabled
synchronised from every mutation site) and the remaining half of **6**
(cache eviction enforced from outside `appState`).

## Field arithmetic

`viewer` is **exactly 87 fields** today, lines 37–481 of
[viewer.go](../internal/ui/viewer.go). (`todos.md` says the menu items are 16
fields; the real count is 20 — 3 File + 5 Window + 12 Actions.)

| Stage | Cluster | Fields before | Fields after | Δ |
|-------|---------|--------------:|-------------:|--:|
| 0 | *(item 6 prep — no field change)* | — | — | 0 |
| 1–3 | menu items | 20 | 1 `menus *menus.Menus` | −19 |
| 4 | updater | 6 | 2 `updater *autoupdate.Updater` + `updateOp requestLifecycle` | −4 |
| 5 | info overlay | 7 | 1 `info *infoview.Card` | −6 |
| 6 | display state | 4 | 1 `display display.State` | −3 |
| | **total** | **87** | **55** | **−32** |

## Architecture after this plan

```
internal/ui/menus        (new) the 20 fyne.MenuItems, plus Apply(State) — the
                               whole Checked/Disabled matrix as one pure
                               function of a value struct. Fyne-typed but
                               viewer-free and unit-testable with no app.
                               No cgo: the Darwin native-bar fold stays in
                               internal/ui/windowmenu_darwin.go.

internal/ui/autoupdate   (new) release check/download policy, the staged-update
                               lifecycle, the What's-New cache, and the
                               last-check-day mutex. Takes a context.Context
                               and a persist seam; owns no fyne.App.

internal/ui/infoview     (new) the info card's three widgets and its text
                               formatting, driven by Update(State).
                               formatFileSize moves here.

internal/ui/display      (new) the decoded frames, the frame index, the
                               view-only rotation, and the crossfade.
                               Owns the RotateSteps composition.

internal/ui             composes all four, computes each package's State
                        snapshot in exactly one function, and keeps every
                        decision that reads more than one feature
                        (applyRotationLayout, syncMenus' choke points,
                        displayedDimensions).
```

## Why this approach

`viewer` grows by construction: every feature that needs a widget reference or a
flag adds a field, and the struct comment — genuinely excellent, which is why
this is Risk 3 — is now 440 lines nobody holds in one head. The four clusters
below were picked because each already has a visible seam in those comments and
a natural home file.

**Snapshot structs, not Host interfaces.** `menus` and `infoview` read state
spread across `appState`, `grid`, `dupes`, `slides`, `exif`, `help` and `zoom`.
A consumer-side `Host` in the style of `grid`/`deletion`/`slideshow` would need
~13 methods for `menus` alone — not the *narrow* Host `ARCHITECTURE.md` asks
for, and it would leave the coupling implicit and permanent. A value `State`
struct filled in one function makes the whole dependency surface one struct
literal you can read at a glance, and makes the enablement matrix unit-testable
without a Fyne app at all.

Alternatives rejected:

- **In-package value structs** (the `settings`/`vectorView` precedent): least
  churn, but the enablement matrix stays reachable only through a live viewer,
  which is the thing that makes it hard to change today.
- **Host interfaces**: consistent with the existing feature packages, but see
  above — this Host is not narrow, and the compile-time coupling is worse than
  the struct literal it replaces.

## Locked decisions

Confirmed by Florian on 2026-08-28. Do not revisit without asking.

1. **Scope:** all four clusters, plus `needs_refactoring.md` item 5 (bundled into
   the menu stages) and the remaining half of item 6 (Stage 0).
2. **Packaging:** all four clusters become subpackages under `internal/ui`.
   Florian was shown, and accepted, the objection that `display` is read from
   five files including the decode path and gains the least from a package wall.
3. **Coupling:** `menus` and `infoview` take a value `State` snapshot, not a
   `Host` interface. Exactly one `internal/ui` function builds each snapshot.
4. **Menu sync:** item 5 lands with the extraction. `Apply(State)` returns
   `changed`, and the native refresh happens only when it does — which deletes
   `HighlightChanged`'s four-boolean hand-diff outright.
5. **Cancellation:** `requestLifecycle` is **not** promoted to a shared package.
   The viewer keeps `updateOp requestLifecycle` and passes `ctx` plus a
   staleness func into `autoupdate`. That cluster shrinks 6→2, not 6→1.
6. **`display` depth:** it owns the frames, the index, the rotation arithmetic,
   the `imaging.RotateSteps` composition, and the fade. `applyRotationLayout`
   stays in `internal/ui` — it reads `zoom`, `vector`, `slides` and `settings`.
7. **Tests:** enablement/formatting matrices move down into the new packages as
   unit tests with no Fyne app. `internal/ui` keeps only integration assertions,
   reaching items through exported accessors (`v.menus.Actions().Hide()`).
   No `export_test.go` bridge in the new packages.
8. **No user-visible behaviour change** anywhere in this plan. Menu labels,
   accelerators, enablement, titles, toasts, key bindings, and the crossfade
   cadence stay exactly as they are. If a stage uncovers a latent bug, the
   subagent **reports** it and does not fix it.

## Global constraints

- **Do not run `git commit`.** End every stage with a suggested commit message.
- **Do not add `TODO`/`FIXME` comments.** Open work goes in `todos.md`.
- **No new translation keys and none removed.** Every `lang.L("…")` key that
  moves keeps its exact English text. `main_test.go` enforces locale parity.
- Move the *existing comments* with the code they document. These comments are
  the reason this refactor is Risk 3 rather than Risk 5; a stage that drops them
  has failed even if it compiles. Rewrite a comment only where the move made it
  factually wrong (e.g. a field path it names).
- Preserve the documented ordering hazards verbatim, in particular
  `rotateBy`/`resetRotation`'s "menu state before `applyRotationLayout`"
  race note ([rotate.go:28–41](../internal/ui/rotate.go)) and
  `applyRotationLayout`'s trailing "after the layout work" note.
- `internal/ui/grid` marshals through `g.ui.Do`, not `fyne.Do`. Nothing in this
  plan touches that; do not "simplify" it.
- Every stage must leave the tree building, formatted, vetted and green on its own.

## Verification (every stage)

Run from the repository root, in this order. A stage is not done until all four pass:

```
make fmt-check          # goimports -local github.com/frathe/picfetch
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

While iterating, the focused subsets are `go test -race ./internal/ui/...` and,
once a package exists, `go test -race ./internal/ui/menus/...` etc.

---

# Stages

Each stage is one subagent dispatch. `agent` is the `subagent_type`; `model` is
the override to pass.

| Stage | Model | Rationale |
|-------|-------|-----------|
| 0 | `sonnet` | Small and mechanical |
| 1 | `opus` | Everything downstream builds on this transcription |
| 2 | `opus` | 8 files, ~77 test refs, exact enablement matrix |
| 3 | `fable` | Pure judgment; failure is a user-visible stale menu |
| 4 | `sonnet` | Contained; the subtleties are spelled out below |
| 5 | `sonnet` | Contained; `infoUI` is already the seed |
| 6 | `fable` | Decode path + load-bearing `-race` ordering notes |
| 7 | `sonnet` | Documentation |

---

## Stage 0 — `appState` evicts the image cache on removal

**agent:** `go-expert` · **model:** `sonnet`
**Closes:** `needs_refactoring.md` item 6 (remaining half)
**Files:** `internal/ui/state.go`, `internal/ui/viewer.go`, `internal/ui/build.go`
(or wherever `appState` is constructed), `internal/ui/imgcache_test.go`

Today `RemoveFile` ([viewer.go:820](../internal/ui/viewer.go)) calls
`v.imgCache.Remove(target.String())` itself. That is the last file-set invariant
nothing but caller convention enforces: a future mutator that forgets the call
leaks decodes of files that are gone.

- [ ] Add an `onRemove func(fyne.URI)` field to `appState`, documented as the
      eviction hook and called from `removeFile` *after* `publish()`, so a
      subscriber always sees the new generation.
- [ ] Wire it in `build.go` **after** the `view = &viewer{…}` literal — the
      closure captures `view`, and `state:` is initialised inside that literal
      by `newAppState(...)`, so it cannot be set there:
      `view.state.onRemove = func(u fyne.URI) { view.imgCache.Remove(u.String()) }`.
      Guard against nil in `removeFile` so a zero `appState` in a test stays usable.
- [ ] Delete the `imgCache.Remove` call from `RemoveFile`, leaving only the
      `invalidateSort()` + `removeFile(i)` pair. Update `RemoveFile`'s doc
      comment to say eviction is now `appState`'s.
- [ ] **Out of scope, deliberately:** `clearFiles` keeps its paired
      `imgCache.Purge()` in `clearToDropzone`. That call has its own documented
      "purged, not left to age out" reasoning tied to the drop-zone reset, not
      to file-set mutation. Do not fold it in.
- [ ] Add a test asserting a removal evicts without the caller asking — call
      `v.state.removeFile` directly and assert the cache no longer holds the key.

**Review focus for the parent:** that `onRemove` fires after `publish()`, and
that no path now double-evicts.

---

## Stage 1 — `internal/ui/menus`, additive only

**agent:** `go-expert` · **model:** `opus`
*(Opus, not Sonnet: `Apply` is the artifact Stages 2 and 3 both build on. A
faithful-looking but subtly wrong transcription here propagates, and the
parent review is the only gate before it does.)*
**Files:** new `internal/ui/menus/menus.go`, `internal/ui/menus/menus_test.go`

Purely additive. `internal/ui` does not import the package yet, so this stage
cannot regress anything — which is why it is split from the switchover.

- [ ] Create `internal/ui/menus` with:
  - `type Menus struct` holding all 20 items: `save`, `export`, `closeFiles`;
    a Window group (`viewer`, `exif`, `grid`, `pictureFrame`, `help`); an
    Actions group (`sort []*fyne.MenuItem`, `hide`, `showVariant`, `rotate`,
    `zoomIn`, `zoomOut`, `merge`, `info`, `copy`, `copyPath`, `wallpaper`,
    `trash`).
  - `type State struct` — the value snapshot. Fields, derived from the three
    existing apply functions: `SortMode filesort.Mode`, `NoFiles`, `GridUp`,
    `NoImage`, `SlidesActive`, `ExifOpen`, `ManualOpen`, `Displayed`,
    `MergeMode`, `HideDuplicates`, `BrowsingDuplicates`, `VariantsSession`,
    `InfoVisible`, `CanSave`, `CanExport`, `CanWallpaper` bool, and
    `VariantGroupSize int`.
  - `func (m *Menus) Apply(s State) (changed bool)` — the **verbatim** logic of
    `applyActionsMenuState` ([actionmenu.go:11](../internal/ui/actionmenu.go)),
    `applyWindowMenuState` ([windowmenu.go:9](../internal/ui/windowmenu.go)) and
    the three File-menu assignments in `updateFileMenuState`
    ([save.go:93](../internal/ui/save.go)), rewritten only to read `s` instead
    of a viewer. `changed` is true when **any** item's `Checked` or `Disabled`
    actually moved — snapshot every item's pair before, compare after.
  - Exported accessors returning the `*fyne.MenuItem`s: `Save()`, `Export()`,
    `CloseFiles()`, `Window() WindowItems`, `Actions() ActionItems`, with
    `WindowItems`/`ActionItems` exposing one method per item and
    `ActionItems.Sort() []*fyne.MenuItem`.
  - `func New(c Callbacks, sortMode filesort.Mode) *Menus` building every item
    with its label, its display-only `desktop.CustomShortcut`, and its initial
    `Disabled`, where `Callbacks` is a struct of the callback funcs
    (`OpenFiles`, `SaveRotation`, `PromptExport`, `CloseFiles`, `ShowSettings`,
    `ShowViewer`, …). Move the `lang.L` labels across unchanged, keys
    byte-identical.

    *(Resolved during Stage 1: an earlier draft of this plan called the
    callback struct `Actions`, which collides with the `Actions()` accessor
    locked decision 7 pins. The callback struct is `Callbacks`; the accessor
    keeps the name. `New` takes a bare `filesort.Mode`, not a `State` — it
    sets the hardcoded initial `Disabled` values, which are not a function of
    any `State`, and taking one would imply it applies the matrix.)*
  - `func (m *Menus) FileMenu() *fyne.Menu`, `ActionsMenu()`, `WindowMenu()` —
    the three `fyne.NewMenu` compositions from `buildMainMenu`, including the
    separators and the `sortParent` child menu, in the same order.
- [ ] Keep the comments explaining *why* each item is display-only versus
      really bound (`copyItem`'s "a second CustomShortcut here would double-fire
      copy", `mergeItem`'s AppKit ⌘M note, `trashItem`'s ShortcutCut note).
      They move with their items.
- [ ] **Do not** move `refreshMainMenu`, `syncNativeMenuBar`,
      `mergeNativeWindowMenu` or `applyUnmodifiedNativeAccelerators`. Those are
      `*fyne.MainMenu`/cgo concerns and stay in `internal/ui`.
- [ ] `menus_test.go`: table-driven unit tests over `Apply`, with **no Fyne
      app** — construct `New(Actions{})` and assert the matrix directly. Cover
      at minimum: sort Checked follows `SortMode`; `hide` disabled on `NoFiles`
      or `VariantsSession`; `showVariant` enabled only when
      `HideDuplicates && VariantGroupSize >= 2` or `BrowsingDuplicates`, and
      never while `SlidesActive`; rotate/zoom disabled on `NoImage || GridUp`;
      `info` disabled on `GridUp`; `windowViewer` disabled unless grid or
      slides is up; `windowExif` disabled on `ExifOpen || !Displayed`;
      `changed` false for an identical re-Apply and true for a single flip.

**Review focus for the parent:** that `Apply` is a faithful transcription. Read
it against all three source functions line by line — this is the stage where a
silent enablement change would hide.

---

## Stage 2 — switch `internal/ui` over to `menus`

**agent:** `go-expert` · **model:** `opus`
**Files:** `internal/ui/viewer.go`, `menu.go`, `actionmenu.go`, `windowmenu.go`,
`save.go`, `build.go`, plus `actionmenu_test.go`, `menu_test.go`,
`windowmenu_test.go`, `keys_test.go`, `export_test.go`, `save_test.go`,
`wallpaper_test.go`

The switchover. Call sites stay 1:1 with today's — reducing them is Stage 3, so
that a regression here is separable from a regression there.

- [ ] Delete the 20 menu-item fields from `viewer` and add
      `menus *menus.Menus`. Move the field-group comments (why these are fields
      at all) onto the new single field, updated to point at the package.
- [ ] `buildMainMenu` becomes: build `menus.Actions{…}` from the viewer's
      existing method values, call `menus.New`, keep the five observer
      registrations (`help.SetOnManualClosed`, … `grid.SetOnDupeStateChanged`)
      and the initial sync, and return
      `fyne.NewMainMenu(view.menus.FileMenu(), view.favorites.Menu(), view.menus.ActionsMenu(), view.menus.WindowMenu(), view.help.Menu())`.
      Menu order must not change.
- [ ] Add `func (v *viewer) menuState() menus.State` — the **only** place the
      snapshot is built. Each field reads exactly what the old apply functions
      read (`v.FileCount() == 0`, `v.grid.Visible()`, `len(v.displayFrames) == 0`,
      `v.slides.Active()`, `v.exif.Open()`, `v.help.ManualOpen()`,
      `v.dupes.HideDuplicates()`, `v.grid.BrowsingDuplicates()`,
      `v.variantsSession()`, `v.grid.SourceDuplicateGroupSize()`,
      `v.infoVisible`, `v.canSaveRotation()`, `v.canExport()`,
      `v.canSetWallpaper()`, and `_, displayed := v.DisplayedFile()`).
- [ ] Add `func (v *viewer) syncMenus()`:
      ```go
      if v.menus == nil { return }
      v.favorites.SetHasFiles(v.FileCount() > 0)
      if v.menus.Apply(v.menuState()) { v.refreshMainMenu() }
      ```
      Note the inverted nil guard: today three functions each guard on a
      different field (`v.actionsHideItem == nil`, `v.windowViewerItem == nil`,
      and `updateFileMenuState` guards on nothing at all). One guard replaces
      all three — check that no test constructs a viewer that reaches
      `updateFileMenuState` before `buildMainMenu`.
- [ ] Replace `updateFileMenuState`, `updateActionsMenuState`,
      `updateWindowMenuState`, `applyActionsMenuState` and
      `applyWindowMenuState` with `syncMenus`, keeping **every** existing call
      site — same file, same line position, same ordering relative to its
      neighbours. Delete the five now-empty functions.
- [ ] **Behaviour note to preserve deliberately:** the old
      `applyActionsMenuState`/`applyWindowMenuState` calls did *not* refresh the
      native bar; only the three `update*` wrappers did. `syncMenus` refreshes
      whenever something changed. That is the intended fix (it is what
      `HighlightChanged` was hand-rolling), and it is a strict improvement, not
      a behaviour change the user can see — but call it out in the commit
      message.
- [ ] Delete `HighlightChanged`'s four-boolean snapshot-and-diff block
      ([viewer.go:558–571](../internal/ui/viewer.go)) and its trailing
      `if v.actionsHideItem == nil { return }` guard, replacing the whole tail
      with one `v.syncMenus()`. Rewrite the doc comment's last two sentences,
      which describe the diffing that no longer exists.
- [ ] Rewrite all ~77 test references to the accessors. Where a test asserts an
      enablement rule that is now covered by a `menus` unit test, **keep** the
      integration assertion — this stage adds coverage, it does not trade it.
- [ ] Delete `actionmenu.go`'s and `windowmenu.go`'s now-unused imports; the
      action *handlers* (`setActionsSort`, `toggleActionsHideDuplicates`,
      `showActionsVariant`, `rotateActionsImage`, …) all stay where they are.

**Review focus for the parent:** the nil-guard consolidation, and that no call
site moved relative to a documented ordering hazard (rotate.go especially).

---

## Stage 3 — collapse the menu push sites to choke points

**agent:** `go-expert` · **model:** `fable`
*(Fable: the only stage whose failure mode is a user-visible bug the test suite
may not catch — a menu that goes stale after some rare path. It is judgment, not
transcription.)*
**Closes:** `needs_refactoring.md` item 5
**Files:** `internal/ui/rotate.go`, `load.go`, `save.go`, `sort.go`, `info.go`,
`keys.go`, `viewer.go`, `windowmenu.go`, `drop.go`, plus affected tests

Item 5's actual payoff. `syncMenus` now recomputes everything from the model, so
the per-site call discipline is redundant: the question is only *where* the
choke points are.

- [ ] Enumerate today's ~20 call sites first and write the list into the stage
      report, grouped by the user action that reaches them. Do not start editing
      before that list exists.
- [ ] **Asymmetry found in Stage 1, account for it explicitly:** the File three
      (`save`/`export`/`closeFiles`) are refreshed today from *six* sites only —
      `load.go:42`, `load.go:321`, `rotate.go:41`, `rotate.go:58`, `save.go:75`
      and `clearToDropzone`. Neither `updateActionsMenuState` nor
      `updateWindowMenuState` nor any of the observer registrations reaches
      them. `syncMenus` recomputes all three from **every** choke point, which
      is a widening of when they refresh, not a narrowing. Benign as far as
      Stage 1's analysis went, but confirm it against your enumeration rather
      than assuming.
- [ ] **Related, also from Stage 1:** `copyActionsPath` and
      `wallpaperActionsImage` have no `FileCount() == 0` guard of their own,
      unlike `copyActionsImage` and `trashActionsImage`. For those two the
      `Disabled` flag is the only thing preventing a no-file invocation, so any
      change to *when* they are disabled is correctness-affecting, not cosmetic.
- [ ] Reduce to a small set of choke points at the end of each user action.
      Candidates, to be confirmed against the enumeration: `handleKeyEvent`'s
      tail, `finishLoad`, `finishSort`, `handleDrop`'s completion,
      `clearToDropzone`, the five feature observer callbacks registered in
      `buildMainMenu`, and each menu action handler that does not already route
      through one of those.
- [ ] **Keep** the explicit `syncMenus` in `rotateBy`/`resetRotation` where it
      is today — *before* `applyRotationLayout`. Its position is a documented
      `-race` fix under the fake test driver, not call-site discipline. Leave
      that comment intact and pointing at the right function name.
- [ ] Every removed call site must be justified in the stage report by naming
      the choke point that now covers it. A site with no covering choke point is
      a site that stays.
- [ ] Run the full suite **twice** — once normally, once with `-count=2` — to
      shake out ordering assumptions in the menu tests.

**Review focus for the parent:** this is the highest-judgment stage. Walk the
enumeration against the choke-point list personally; a menu that goes stale
after some rare path is exactly the regression this stage risks.

---

## Stage 4 — `internal/ui/autoupdate`

**agent:** `go-expert` · **model:** `sonnet`
**Files:** new `internal/ui/autoupdate/{updater.go,whatsnew.go,updater_test.go}`;
`internal/ui/viewer.go`, `autoupdate.go`, `run.go`, `features.go`,
`memlimits.go`, `harness_test.go`, `autoupdate_test.go`,
`preferences_wiring_test.go`

- [ ] Create `autoupdate.Updater` owning: the `*update.Client`, the stage dir,
      the `completion.Signal` for the background check, the current-version
      override, **and** `lastCheckDay` plus the mutex that guards it.
- [ ] **Note the field that moves out of `settings`:** `updateDayMu` today
      guards `v.settings.lastUpdateCheckDay`, which lives in the `settings`
      struct, not in the updater cluster. Move `lastUpdateCheckDay` into
      `Updater` and delete it from `settings` ([memlimits.go:61](../internal/ui/memlimits.go)),
      updating that struct's comment. `settings.checkForUpdates` **stays** —
      it is a settings-window checkbox and belongs to the settings surface.
- [ ] Construct with a config carrying a `Persist func(day string)` seam,
      wired in `internal/ui` to `preferences.SaveLastUpdateCheckDay(v.app, day)`.
      The package must not take a `fyne.App` for the day round-trip.
- [ ] Per locked decision 5, cancellation stays with the viewer:
      `func (u *Updater) Start(ctx context.Context, stale func() bool, currentVersion string)`
      begins the completion signal and runs the goroutine. The viewer keeps
      `updateOp requestLifecycle` and passes `token.context()` and
      `token.current`.
- [ ] Move `defaultUpdateDir`, `removeStaleUpdateStage`, `updateNow`,
      `applyStagedUpdate`, and the check/download goroutine body into the
      package. Move `whatsNewCache`, `saveWhatsNew`, `loadWhatsNew`,
      `clearWhatsNew` into `whatsnew.go`, taking `fyne.App` as they do now.
- [ ] `maybeShowWhatsNew` **stays** in `internal/ui`: it ends in
      `v.help.ShowWhatsNew(...)`. It calls the package's `LoadWhatsNew`/
      `ClearWhatsNew`.
- [ ] Viewer keeps exactly two fields: `updater *autoupdate.Updater` and
      `updateOp requestLifecycle`. `CheckForUpdates`/`SetCheckForUpdates`/
      `LastUpdateCheckDay`/`SetLastUpdateCheckDay` stay as viewer methods (they
      are the settings window's Host surface) and delegate.
- [ ] Four touch points outside `autoupdate.go`, not three: `build.go`'s
      `updateDir: defaultUpdateDir()` in the viewer literal, `run.go`'s
      re-default in `startViewerRuntime` (`if view.updateDir == ""`, which is
      what a test viewer with a `t.TempDir()` relies on), the
      `updateOp.lifecycle.invalidate()` in `registerShutdown`, and
      `currentPreferences`' `LastUpdateCheckDay`. These and
      `features.go:82`'s restore-before-`maybeStartUpdateCheck` ordering must
      all still work. That ordering comment is load-bearing — `Due` needs the
      saved day in place first.
- [ ] `harness_test.go`: `v.updateDir = t.TempDir()` becomes a constructor
      argument or setter on the test updater; the `drain` entry for
      `&v.updateDone` and `v.updateOp.lifecycle.invalidate()` must keep working.
      All ~52 `v.update*` references in `autoupdate_test.go` get rewritten.
- [ ] New `updater_test.go`: due/not-due, stale-stage removal, and the
      version-normalisation gates as unit tests with no viewer.

**Review focus for the parent:** the `features.go` restore ordering and the
`harness_test.go` drain wiring — a background goroutine outliving a test is the
failure mode here, and it shows up as a flake, not a failure.

---

## Stage 5 — `internal/ui/infoview`

**agent:** `go-expert` · **model:** `sonnet`
**Files:** new `internal/ui/infoview/{card.go,card_test.go}`;
`internal/ui/viewer.go`, `info.go`, `components.go`, `build.go`, `load.go`,
`actionmenu_test.go`, `exif_test.go`, `export_test.go`, `info_test.go`,
`save_test.go`, `vector_test.go`

`components.go`'s existing `infoUI` struct (text / exifLink / card) is already
the seed of this package — start from it.

- [ ] Create `infoview.Card` owning `visible bool`, the `*widget.Label`, the
      `*widget.Hyperlink`, the `*fyne.Container`, and the three current-file
      facts (`fileSize int64`, `hasEXIF bool`, `preview bool`).
- [ ] `func New(onShowExif func()) *Card` absorbs `newInfoOverlayUI`
      ([components.go:187](../internal/ui/components.go)) and retires the
      `infoUI` struct ([components.go:176](../internal/ui/components.go)).
      `build.go:50` calls `infoview.New` instead, and the three
      `infoText:`/`infoCard:`/`exifLink:` entries in the viewer literal collapse
      to one `info:`. `func (c *Card) Object() fyne.CanvasObject`
      hands the container to `build.go`'s overlay stack — **overlay order in
      build.go is load-bearing**; the card must land in the same position.
- [ ] `func (c *Card) SetFile(fileSize int64, hasEXIF, preview bool)` replaces
      the three assignments in `finishLoad` ([load.go:260](../internal/ui/load.go))
      and the `hasEXIF = false` reset in `viewer.go`.
- [ ] `type State struct { Name string; Index, Count, Width, Height int; ZoomPercent int }`
      and `func (c *Card) Update(s State)` carry `updateInfoOverlay`'s text
      build. `formatFileSize` moves into the package (it is pure).
      **`displayedDimensions` stays in `internal/ui`** — it reads `v.vector.svg`,
      `v.vector.logical` and the rotation, and its "why not img.Image.Bounds()"
      comment belongs with the vector code it is about.
- [ ] `func (c *Card) Toggle() bool`, `Visible() bool`, and
      `func (c *Card) Sync(hasImage bool, s State)` carry `toggleInfoOverlay`
      and `syncInfoOverlayVisibility`. Keep the comment explaining why the EXIF
      link is settled in the sync path and not in `Update` — a zoom cannot add
      or remove a file's metadata.
- [ ] The viewer keeps thin `toggleInfoOverlay` / `syncInfoOverlayVisibility` /
      `updateInfoOverlay` methods: they build the `State` (reading `state.files`,
      `state.index`, `zoom.Percent()`, `displayedDimensions()`) and call the
      card. `zoom`'s `onChanged` callback keeps pointing at
      `v.updateInfoOverlay`, unchanged.
- [ ] `menuState()`'s `InfoVisible` now reads `v.info.Visible()`.
- [ ] `card_test.go`: `formatFileSize` boundaries (B/KiB/MiB/GiB), the
      `(preview)` suffix, the `(i/n)` position suffix present only when `n > 1`,
      and the exifLink show/hide rule — all with no Fyne app beyond what a
      widget needs.

**Review focus for the parent:** build.go's overlay ordering, and that the
`lang.L` keys `"Show EXIF data"`, `"(preview)"` and `"Zoom: %d%%"` survive
byte-identical.

---

## Stage 6 — `internal/ui/display`

**agent:** `go-expert` · **model:** `fable`
*(Fable: `load.go` is the decode path and `rotate.go`/`save.go` carry `-race`
ordering notes that are load-bearing. A moved write here surfaces as a flake,
not a failure.)*
**Files:** new `internal/ui/display/{display.go,display_test.go}`;
`internal/ui/viewer.go`, `load.go`, `rotate.go`, `save.go`, `vector.go`,
`actionmenu.go`, plus `animate_test.go`, `actionmenu_test.go`, `export_test.go`,
`imgcache_test.go`, `save_test.go`, `slideshow_test.go`, `rotate_test.go`,
`wallpaper_test.go`, `vector_test.go`

The riskiest stage: `load.go` is the decode path, and `rotate.go`/`save.go`
carry documented `-race` ordering notes. `opus` for that reason.

- [ ] Create `display.State` (a value type, embedded as a viewer field, never
      copied) owning `frames []image.Image`, `idx int`, `rotation int`,
      `fade *fyne.Animation`.
- [ ] Methods, per locked decision 6: `SetFrames`, `Count`, `Index`, `SetIndex`,
      `Rotation`, `RotateBy(steps int)`, `ResetRotation() (changed bool)`,
      `Current() image.Image`, `Rotated() image.Image` (the
      `imaging.RotateSteps(frames[idx], rotation)` composition),
      `ReplaceCurrent(image.Image)`, `Clear()`, `StartFade(*canvas.Image, time.Duration, func())`,
      `ResetFade()`.
- [ ] `RotateBy` owns the `((r+steps)%4 + 4) % 4` normalisation.
      `redrawRotatedFrame` in `internal/ui` becomes
      `v.img.Image = v.display.Rotated(); v.img.Refresh(); v.animFrame.Add(1)`.
      `animFrame` **stays on the viewer** — `harness_test.go`'s drain
      synchronises on it and it is not display state proper.
- [ ] `applyRotationLayout` stays in `internal/ui` verbatim, including both its
      leading vector/logical-size block and its trailing "deliberately after the
      layout work" comment.
- [ ] `save.go`'s `canSaveRotation` reads `v.display.Rotation() != 0 && … && v.display.Count() == 1`;
      `saveRotation`'s fold becomes `v.display.ReplaceCurrent(v.img.Image)`
      followed by `v.display.ResetRotation()`. Keep the long comment explaining
      why the fold makes "unrotated" mean the file's new orientation.
- [ ] `vector.go:230`'s `v.displayFrames[0] = frame` becomes
      `v.display.ReplaceFirst(frame)` (or `ReplaceCurrent` if the index is
      provably 0 there — check, do not assume) with its
      `len(v.displayFrames) == 0` guard preserved as a `Count() == 0` check.
- [ ] `actionmenu.go`'s three `len(v.displayFrames) == 0` guards and
      `menuState()`'s `NoImage` become `v.display.Count() == 0`.
- [ ] `load.go`: `finishLoad`'s frame install, the animated-GIF single-frame
      case, the `displayFrameIdx = 0` / `rotation = 0` reset, `animate`'s
      `displayFrameIdx = idx` write, and `startFade`/`resetFade` all move to
      method calls. **Change nothing about when they run.**
- [ ] `clearToDropzone`'s `displayFrames = nil; displayFrameIdx = 0` becomes
      `v.display.Clear()`, keeping the comment about rotate/zoom enablement
      agreeing with the empty drop zone.
- [ ] `display_test.go`: rotation normalisation across negative and >4 steps,
      `ResetRotation` returning false when already 0, `Rotated` composing with
      the current index, `Clear` zeroing everything, and `ResetFade` being safe
      when no fade is running.
- [ ] Run `go test -race -count=2 ./internal/ui/...` — the animation and vector
      tests are where a moved write would surface.

**Review focus for the parent:** every `-race` comment in rotate.go and load.go
still describes what the code does, and `startFade`/`resetFade` are still called
from exactly the paths listed in `fadeAnim`'s old field comment.

---

## Stage 7 — documentation and backlog

**agent:** `general-purpose` · **model:** `sonnet`
**Files:** `ARCHITECTURE.md`, `AGENTS.md`, `todos.md`, `needs_refactoring.md`,
`plans/` → `finished_refactorings/`

- [ ] `ARCHITECTURE.md`: add the four new packages to the package map and the
      "where to look for X" index. `AGENTS.md` requires this in the same change
      as the package addition — it is being deferred to one stage only because
      the packages land across six commits.
- [ ] `AGENTS.md`: one line under "Architecture and Data Flow" noting that
      `menus` and `infoview` take a `State` snapshot rather than a `Host`, and
      why, so the next feature does not copy the wrong pattern.
- [ ] `todos.md`: move the god-object item from TODO to Done → Internal, with
      the real numbers (87 → 55 fields) and one line per package.
- [ ] `todos.md`, two gaps found and verified during Stages 2–3, neither a bug:
      - `internal/ui/favorites` calls `f.menu.Refresh()` from two sites
        (`SetHasFiles`, `refreshMenu`) with no `syncNativeMenuBar` follow-up, so
        adding/renaming/deleting a favorite already leaves the macOS bar
        un-merged until the next `refreshMainMenu`. The invariant worth writing
        down: nothing outside `refreshMainMenu` may call `Refresh()` on a menu
        that lives in the main bar.
      - Stage 3 removed the hand-syncs from F1 (`keys.go`) and Window → Help,
        which now rely on `help.SetOnManualOpened(view.syncMenus)` staying
        registered. That registration is guarded by no test in `internal/ui`:
        `ShowManual` cannot be called under the Fyne test theme (see the note
        at the tail of `e2e_test.go`), so removing the registration passes the
        whole suite. Verified by mutation. `help`'s own
        `TestHelp_SetOnManualOpenedFiresOnShow` pins that the hook *fires*, not
        that this package subscribes to it.
- [ ] `needs_refactoring.md`: mark items 5 and 6 resolved with dates; re-score
      item 3 against the new field count or retire it; update the "Suggested
      sequencing" section, whose steps 2 and 4 this plan consumes.
- [ ] `git mv plans/2026-08-28-viewer-field-clusters.md finished_refactorings/`.

---

## Risks and how each is contained

| Risk | Containment |
|------|-------------|
| A menu silently changes enablement | Stage 1 is additive with unit tests before anything is wired; Stage 2 is a 1:1 call-site swap; Stage 3 is separate so a regression is attributable |
| A menu goes stale after a rare path | Stage 3 requires a written enumeration of all ~20 sites, each removal justified by a named choke point |
| A `-race` flake from moved writes on the decode path | Stage 6 is `opus`, preserves the documented ordering comments verbatim, and runs `-count=2` |
| A background goroutine outlives a test | Stage 4 explicitly re-verifies `harness_test.go`'s `drain` wiring |
| Translation-key drift | No key is added or removed; `main_test.go`'s locale parity check runs in every stage's suite |
| Comment loss | Global constraint: comments move with their code; the parent reviews for it every stage |

## Not in this plan

- `needs_refactoring.md` item 9 (mode-interaction guards in `handleKeyEvent`).
  Stage 3 touches `keys.go` only to remove menu-sync calls.
- Item 8 (favorites prewarm budget) — an open design question, not debt.
- The remaining 55 fields. `state`, `vector`, `settings` and the widget
  references are either already grouped or genuinely belong to the façade.
