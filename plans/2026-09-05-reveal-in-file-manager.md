# Reveal in file manager

**Route:** Standard. One feature, one new leaf package plus viewer/menu/overlay
glue. No cross-package refactor, no change to an existing contract other than
`infoview.New`'s constructor.

## Problem

PicFetch can put the current image's path on the clipboard (Actions -> Copy
image path, `Cmd/Ctrl+Shift+C`, `internal/ui/clipboard.go`) but has no way to
open the OS file manager with that file selected. Getting from "this picture"
to "this picture's folder" means pasting a path into another app.

`todos.md`'s entry asked for both halves. The "Copy full path" half already
ships — `viewer.copyPathToClipboard`, `menus.ActionItems.copyPath`,
`wireClipboardShortcuts` — so this delivers only the reveal half and corrects
the todo.

## Decisions

Resolved; do not relitigate.

| Question | Decision |
|---|---|
| Package | `internal/filemanager`, exported `var Reveal = func(path string) error`. Same dispatcher-var shape as `trash.Move` / `wallpaper.Set` / `clipboard.CopyFiles`, so `internal/uitest` can stub the whole platform dispatch. |
| Label | One string, `"Reveal in file manager"`, on every platform. Per-OS wording (Finder/Explorer/Files) would triple catalogue keys and put a `runtime.GOOS` branch inside the deliberately viewer-free `internal/ui/menus`. |
| macOS | `open -R <path>` shell-out, *not* cgo/AppKit. `trash`/`wallpaper`/`filepicker` use AppKit because their alternative was AppleScript, which triggers the one-time Automation prompt. `/usr/bin/open` is a LaunchServices binary, not an Apple Event to another app, so there is no prompt to dodge and no reason to pay for cgo. |
| Windows | `explorer.exe /select,"<path>"`, with the exact command line set through `SysProcAttr.CmdLine`. Go's own argument escaping would emit `"/select,C:\a b\x.jpg"` (quotes around the whole argument) for any path containing a space; the form Microsoft documents and every working recipe uses puts the quotes around the path only. Explorer's parser is not `CommandLineToArgvW`, so that difference is not safely guessable. The builder is a portable pure function, so the tricky part is unit-tested on Linux. |
| Windows exit code | `explorer.exe` returns a non-zero exit status even on success. `revealWindows` stats the path first (a real, reportable error) and then treats an `*exec.ExitError` from explorer as success. Anything else — a failure to start the process at all — is still returned. |
| Linux | `dbus-send` to `org.freedesktop.FileManager1.ShowItems`, the freedesktop interface Nautilus/Dolphin/Nemo/Thunar implement, which selects the file. Falls back to `xdg-open <parent dir>`, which only opens the folder. `--print-reply` is mandatory: without it `dbus-send` does not wait for a reply and a missing file manager exits 0, so the fallback would never run. |
| Accelerator | `Cmd/Ctrl+R`. Bare `R` stays Rotate (`keys.go`). `R` is not one of the glfw driver's special-cased bare combos (Z/Y/V/C/Insert/X/A), so a plain `desktop.CustomShortcut` is reachable. |
| Surfaces | Actions menu item next to Copy image path, a canvas shortcut, and a second hyperlink in the info overlay under "Show EXIF data". |
| Enablement | Identical to Copy image path: disabled when no files are loaded, and disabled by `applyComparisonIsolation`. |
| Subject | The current file only, never the grid selection. `batch.go`'s subject routing exists for delete/copy because those act on a set; a file manager reveal of twelve files in nine folders has no sensible meaning. |
| Async | Own goroutine behind a `completion.Signal` (`v.reveal`), like `copyImageToClipboard`. Every backing command blocks on external I/O. |
| Failure | Log via `fyne.LogError` plus a toast on every platform, like `reportClipboardError` — there is no cancel/failure ambiguity here on any OS, unlike `reportChooserError`'s Linux case. |

## Acceptance criteria

```
AC1  filemanager.Reveal dispatches by GOOS; macOS runs `open -R <path>`.
     go test ./internal/filemanager/

AC2  Linux prefers dbus-send's FileManager1.ShowItems with a percent-encoded
     file:// URI and --print-reply, falls back to xdg-open on the parent
     directory, and errors when neither tool is installed.
     go test ./internal/filemanager/

AC3  Windows emits exactly `explorer.exe /select,"<path>"` as its command
     line and reports explorer's non-zero exit as success, while a missing
     path is still an error.
     go test ./internal/filemanager/

AC4  viewer.revealCurrentFile runs on its own goroutine, hands the current
     file's path to filemanager.Reveal, no-ops with no files loaded or during
     comparison, and reports a failure as a toast.
     go test ./internal/ui/ -run TestReveal

AC5  The Actions menu carries "Reveal in file manager" with the
     Cmd/Ctrl+R accelerator, disabled with no files and by comparison
     isolation.
     go test ./internal/ui/menus/ && go test ./internal/ui/ -run TestActionsMenu

AC6  Cmd/Ctrl+R reaches revealCurrentFile through the production wiring.
     go test ./internal/ui/ -run TestRevealShortcut

AC7  The info overlay shows a "Reveal in file manager" link whenever the card
     itself is shown, independent of whether the file has EXIF.
     go test ./internal/ui/infoview/

AC8  Every new user-visible string is in both catalogues, English is still an
     identity map, and no Unicode arrow reaches the renderer.
     go test . -run TestTranslations &&
     go test ./internal/ui/help/ -run TestManualHasNoUnicodeArrows

AC9  The manual documents the command in both languages, and Qodana's
     duplication exclusions cover the new test files.
     go test ./internal/ui/help/ -run TestManualDocumentsReveal &&
     make check-qodana-test-exclusions
```

## Non-goals

- No per-OS label or per-OS menu wording.
- No reveal of a grid multi-selection.
- No change to Copy image path, which already ships.
- No `internal/wincom` COM path (`SHOpenFolderAndSelectItems`). The shell-out
  is what the todo asked for and what every other OS integration here does.

## The honest limit

The macOS and Windows execution paths cannot be run from this machine. Their
*command construction* is unit-tested on Linux; that a `explorer.exe
/select,"…"` command line actually selects the file in a real Explorer, and
that `open -R` actually raises Finder, remain unverified here. The Linux path
is verified by unit tests over the same stub seam the rest of the repo uses,
and by a real run.

## Tasks

### Task 1 — `internal/filemanager`
Owner:   T0 inline (Law 1: the platform reasoning above *is* the context)
Files:   create `internal/filemanager/filemanager.go`, `windows.go`,
         `notwindows.go`, `filemanager_test.go`
Depends: —
Contract:
```go
var Reveal = func(path string) error   // dispatcher var, stubbable
func explorerCmdLine(path string) string
```
Test:    per-platform command construction, the Linux fallback chain, the
         missing-tool error, and explorer's success-with-nonzero-exit.
Verify:  `go test ./internal/filemanager/`
Budget:  0 spawns · 1 review round · full suite: no

### Task 2 — viewer glue
Owner:   T0 inline
Files:   modify `internal/ui/reveal.go` (new), `viewer.go`, `harness_test.go`,
         `internal/uitest/stubs.go`; test `internal/ui/reveal_test.go`
Depends: 1
Contract: `func (v *viewer) revealCurrentFile()`; `v.reveal completion.Signal`;
         `uitest.StubReveal(t, func(path string) error)`
Test:    goroutine + signal, no-files no-op, comparison refusal, toast on error.
Verify:  `go test ./internal/ui/ -run TestReveal`
Budget:  0 spawns · 1 review round · full suite: no

### Task 3 — menu, shortcut, info overlay
Owner:   T0 inline (touches a user-visible string — never delegated, §8)
Files:   `internal/ui/menus/menus.go` + its test, `internal/ui/menu.go`,
         `actionmenu.go`, `shortcuts.go`, `internal/ui/infoview/card.go` + its
         test, `internal/ui/build.go`, `translations/*.json`
Depends: 2
Contract: `menus.Callbacks.Reveal func()`, `menus.ActionItems.Reveal()`
Test:    label + accelerator + disabled matrix; shortcut reaches the command;
         the card's link is shown with the card.
Verify:  `go test ./internal/ui/menus/ ./internal/ui/infoview/ ./internal/ui/ -run 'TestReveal|TestActionsMenu'`
Budget:  0 spawns · 1 review round · full suite: no

### Task 4 — docs and hygiene
Owner:   T0 inline
Files:   `ARCHITECTURE.md`, `internal/ui/help/manual.md`, `manual_de.md`,
         `internal/ui/help/manual_test.go`, `qodana.yaml`, `todos.md`
Depends: 3
Verify:  `make check-qodana-test-exclusions && go test . ./internal/ui/help/`
Budget:  0 spawns · 1 review round · full suite: no

### Gate
`make fmt-check && go vet ./... && go build ./... && go test -timeout 30m -race ./...`
plus `GOOS=windows GOARCH=amd64 go vet ./internal/...` and
`GOOS=darwin GOARCH=arm64 go vet ./internal/filemanager/`, since two of the
three platform paths are cross-compiled only.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| 1    | 0 / 0                  | 1             | no         | hot context (G5) |
| 2    | 0 / 0                  | 1             | no         | hot context (G5) |
| 3    | 0 / 0                  | 1             | no         | user-visible strings, never delegated (§8) |
| 4    | 0 / 0                  | 1             | no         | scripted edits (rule S) |
| gate | —                      | —             | yes        | `make verify` |

Zero spawns, zero Scouts. Recon was six greps plus targeted reads; the whole
change was inside the Lead's context from the first file read.

## What review caught

- `TestSync_RevealLinkFollowsTheCardNotEXIF` did **not** fail when the reveal
  link was deleted from the card's container: `widget.Hyperlink.Visible()` is
  the widget's own flag, not a statement about the tree it is in, so a link
  that never reaches the screen still reports `true`. Replaced the visibility
  assertion with an `inCard` walk of `c.Object()`, which does fail on that
  mutation. The same blind spot applies to the pre-existing `ExifLink`
  assertions; they are left alone here because their show/hide toggle is
  driven by `Sync` and is genuinely covered.
- The first `TestManualDocumentsReveal` checked both manuals for `Ctrl+R`.
  The German manual says `Strg+R`, so the guard failed on its own first run —
  the modifier name is itself translated.

## Verification

Negatively verified (mutation, observed failure, restore):

| Guard | Mutation | Result |
|---|---|---|
| `TestCompareCommandIsolation_ShortcutsAreIgnored` | drop the comparison gate from `revealCurrentFile` and register the shortcut on the raw adder | fails: `reveal=true` |
| `TestApply_ClipboardWallpaperAndTrash` | `reveal.Disabled = false` | fails on both no-files rows |
| `TestCompareMenuState_DisablesEveryOrdinaryItemButHelp` | drop `reveal` from `applyComparisonIsolation` | fails |
| `TestSync_RevealLinkFollowsTheCardNotEXIF` | leave `revealLink` out of the card container | fails |
| `TestRevealCurrentFile_NoFilesIsNoop` | drop the empty-file-set guard | fails |
| `TestRevealLinux_SendsShowItemsOverDBus` | drop `--print-reply` | fails |
| `TestRevealWindows_TreatsExplorersNonZeroExitAsSuccess` | report explorer's exit status | fails |
| `TestExplorerCmdLine_QuotesOnlyThePath` | drop the quotes | fails |

Verified against the real desktop (Linux/GNOME): `org.freedesktop.FileManager1`
is activatable on this session bus, the literal `dbus-send ... ShowItems` call
this code builds returns `method return` for a path containing both a space
and a `#`, and `filemanager.Reveal` itself returned nil through the Go
dispatcher for the same file.

Final gate, `make verify`: **exit 0**. Zero `action=fail` events across all four
partitions; `internal/filemanager` green in `non-ui`; all seven new
`internal/ui` tests green under `-race` in the Linux/amd64 container; the e2e
goldens matched (no `testdata/failed/` written), since no golden shows the info
overlay.

Still unverified, as the honest limit above predicted: the macOS `open -R` and
Windows `explorer.exe /select,` executions. Their command construction is
unit-tested; nobody has watched them run.

## What the gate caught

The first `make verify` failed — after building the Docker image — on
`scripts/testshards check`: every top-level `internal/ui` test must carry an
exact shard assignment in `.github/testshards/internal-ui.tsv`, and the seven
new ones had none. Fixed by distributing them 3/2/2 across `ui-1`/`ui-2`/`ui-3`
in sorted position and refreshing the header entry counts. Written into
`AGENTS.md` next to the Qodana test-exclusion rule, which is the same shape of
chore, so the next test added to that package does not repeat the 25-minute
round trip. `make check-test-shards` answers it in seconds.

A second lesson, this one about the tooling rather than the repo: the first run
was launched as `make verify 2>&1 | tail -60`, so the pipeline's exit status was
`tail`'s and the completion notification read "exit code 0" for a run that had
in fact failed. Gates must be run without a trailing pipe, or under
`set -o pipefail`.
