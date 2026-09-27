# MA-028 Windows 11 qualification

Status: Windows qualification complete, including operator-performed physical
Ctrl+A/C in both image and Grid naming fields. Linux physical-input/literal
Make-run qualification and full macOS desktop qualification remain open in
[ticket 10](ma-028/issues/10-native-qualification.md). This record does not close
all-platform AC10 or archive the plan. Final documentation-head CI and reviews
are recorded separately on PR 66.

## Final physical-keyboard qualification

Tested source: `10f16a0126dde8f852a058682dd0681c61111122`, with no source edits.
Windows 11 Pro 25H2, build 26200.9457, amd64; Go 1.27.1 and Fyne 2.8.0.
The new isolated ordinary profile and copied generated fixtures were launched
with literal `make run`. The resulting executable SHA-256 was
`f01b83e198f889066a7d0832ca05899a4566e269d99d1445dbf062c5ef50873e`.
Raw artifacts are retained locally under
`.scratch/ma-028/windows-mcp-2026-09-27`; see `environment.json`,
`tested-executable.json`, the physical-result JSON files and screenshots 07-16.

T0 prepared Add to Favorites over the loaded image, entered distinctive text,
then replaced the clipboard with an unrelated sentinel without injecting Copy
or Select All. The user physically pressed Ctrl+A then Ctrl+C and confirmed
completion. T0 read the OS clipboard, inspected the complete highlighted field,
and compared the underlying image pixels. The same procedure was repeated
over Grid with only the first of three fixtures selected.

| Context | Physical result observed (UTC) | Clipboard result | Visible/state evidence |
| --- | --- | --- | --- |
| Image | 17:20:37 | Exact `MA028 IMAGE physical 927`, replacing the image sentinel | Screenshot 10 shows full-field selection. All 77,694 exposed image pixels match screenshot 09; after Cancel, all 157,491 content pixels match the original loaded view. Title, image index and dimensions are unchanged. |
| Grid | 17:28:58 | Exact `MA028 GRID physical 927`, replacing the Grid sentinel | Screenshot 15 shows full-field selection. All 50,050 pixels covering the three thumbnails and selection borders match screenshot 14; the same rectangle is identical before opening and after cancelling the dialog. Only the first thumbnail remains selected. |

The comparisons are retained in `image-pixel-comparison.json` and
`physical-pixel-comparisons.json`, with explicit regions and zero differing
pixels. Both dialogs were cancelled; no Favorite was created. Native close
ended the owned application and Make launcher with exit 0 (`app-exit.txt`).

Windows-MCP supplied setup screenshots through an evidence-only local stdio
client. Its initial coordinate-based input did not reliably retain the intended
window, so those attempts are excluded. Final preparation used verified Win32
window/control handles and foreground ownership. An intervening unavailable
desktop paused collection until the user returned. Neither setup automation
nor screenshots alone are treated as physical input: the two successful
operator-performed shortcut sequences above provide that evidence.

Together with the native menus, modal controls, held-copy recovery/close and
review-fix observations below and on PR 66, this closes the Windows portion of
V10. This follow-up changes documentation only. Source test/inspection evidence
carries forward at its recorded revisions, including the final `10f16a0`
Windows adapter inspection; fresh remote gates remain required for its new head.

## Environment and isolation

- Windows 11 Pro 25H2, build 26200.9445, amd64; Go 1.27.1,
  MinGW-w64 GCC 16.2.0, pinned Fyne 2.8.0; German desktop/app locale.
- Initial clean revision: `bae3c55c78d3f794456447d4baad9c84b7a49eb1`.
  Final native repeats used the focus repair in this change. Exact source
  hashes are below; the commit and remote gates are pinned on PR 66.
- Literal `make run` executed `go run -tags no_emoji,nodynamic .` for the
  ordinary application identity. Only the child received isolated `USERPROFILE`,
  `APPDATA`, `LOCALAPPDATA`, `TEMP` and `TMP`. Fyne derives Windows configuration
  from `USERPROFILE`; `APPDATA` alone would not isolate preferences. Go caches
  were explicitly preserved. Update checks stayed off.
- Only three generated 480 x 320 opaque PNGs were opened. The normal profile,
  Favorites and library were not used. The first native chooser exposed a
  missing `Desktop` directory in the isolated profile; creating it repaired
  the harness setup. Subsequent native Open succeeded.
- Input used Win32 `keybd_event` with mapped scan codes and `mouse_event`,
  target-window focus checks, actual menu clicks and `WM_CLOSE`. This is
  OS-injected native input, not physical keyboard evidence or Fyne test-driver
  dispatch. Early special-key attempts without scan codes are excluded.

Raw artifacts remain ignored under `.scratch/ma-028/windows-2026-09-27`:
`launch.ps1`, `desktop.ps1`, `clipboard-launcher.go`, fixtures, app/build/test
logs, clipboard PNGs, numbered screenshots and inspection JSON. They include
local paths and are intentionally not published.

The launch script preserves `GOPATH`, `GOMODCACHE` and `GOCACHE`, creates the
profile's `AppData/Roaming`, `AppData/Local` and `Desktop`, sets `CGO_ENABLED=1`,
prepends `.tools/windows/mingw64/bin`, and runs `make run` from the repository
root. The helper starts hidden; PicFetch itself is the visible test subject.

## Native outcomes

| Reproducible scenario | Observed result | Retained artifacts |
| --- | --- | --- |
| Open a fixture with Ctrl+O. In Add to Favorites (Alt+Shift+F), click the name field, paste distinctive text, Ctrl+A, replace clipboard with a separate sentinel, then Ctrl+C. Repeat after G opens Grid. | Exact text replaced the sentinel in both contexts. Grid thumbnails/selection were pixel-identical after cancelling. | `19-grid-editor-verified`, `20-grid-restored`, `23-image-editor-verified`, text payloads, `payload-verification.txt`; final-source `75-final-editor` |
| Outside the dialog, use Ctrl+C and actual Actions -> Copy image. Seed text before each copy and wait for an image. | Both copies matched all 153,600 fixture pixels. | `native-copy-verified.png`, `menu-copy-verified.png`, `25-full-actions-menu` |
| In the real Favorite modal, try Ctrl+S, Ctrl+E and Actions, then its Add/Cancel controls. | Modal ownership preserved; menu access blocked. Add saved exactly the three isolated fixture paths; Cancel restored the view. | `24-modal-refused-verified`, isolated Favorite `file-list.json`, `75-final-editor` |
| Rotate, enter Actions -> Copy selection, drag a rectangle and Return. Hold successful clipboard completion; inspect menus and try disabled Save, Ctrl+S and Ctrl+Shift+C. | Conflicting File/Actions items disabled, permitted zoom/copy retained. Save refusal toast visible; fixture pixels unchanged. Copy Path left the 107 x 113 crop byte-identical. | `33-region-drawn`, `34-busy-file-menu`, `36-busy-save-feedback`, `37-busy-actions-menu`, `region-held.png`, `region-after-refusal.png` |
| Release the launcher and invoke Copy Path. | File actions re-enabled and the fixture path replaced clipboard data. | `38-file-menu-recovered`, `recovered-copy-path.txt` |
| Open Export/Delete; use Tab, arrows, checkbox click, Return and Escape; try conflicting shortcuts. | Initial focus defect reproduced, then fixed and repeated successfully. Export's PNG control reached the real save chooser, which was cancelled. Delete confirmation used Return on Cancel; no file was deleted. | `39` through `64`; final-source `67` through `70` |
| Maximize after prompt dismissal, then Escape. | Welcome restored to 536 x 379 native pixels, `IsZoomed=false`, without clicking to recover input. | `65-fixed-post-prompt-reset`, `71-final-reset`, `76-final-cancel-reset` |
| Native close during a fresh held copy, then separately while a Favorite modal is visible. | App PID 23108 exited while held writer PID 20352 remained alive; release followed app exit. Final busy/modal sessions exited 0. All owned app/hold processes were released. | `73-final-busy-file-menu`, `74-final-busy-feedback`, `busy-close.txt`, `app-busy-close-exit.txt`, `77-final-modal-close`, `final-modal-close.txt`, `app-exit.txt` |

The hold uses an evidence-only `powershell.exe` in the app child's PATH. It
forwards unchanged arguments/environment to absolute system PowerShell, waits
for successful real PNG publication, writes its PID to `ready`, then waits for
an owned `release` file with a ten-minute timeout. It holds only when
`PICFETCH_CLIPBOARD_PNG` and the explicit `hold` marker are present. This
observes the production clipboard worker without product-code or OS clipboard
policy changes. It is an external process-completion hold, not the spec's
optional instance encoder seam.

## Prompt-focus repair and verification

Native Tab could focus Export's `widget.Check`, which ignores Escape, stranding
the prompt. Delete's panel could retain focus after hiding, swallowing the next
viewer Escape. The regression reproduced both through Fyne's focused dispatch.

ChoicePanel now forwards card-owned keys to ChoiceCard and captures Tab. Root
assigns that single owner while a prompt is visible and releases it on dismissal.
Checkbox clicks return focus to the card. Existing Up/Down, Left/Right, Return
and Escape behavior is preserved; standalone dialog panels retain their Tab
behavior. No dependency, package, user-visible string or top-level test was added.

The new `TestCommandAdmissionModalOwnership` subtests failed before the fix.
A Go overlay restoring the original `commandadmission.go` made both fail again:
hidden Delete focus and Export swallowing Escape. The existing checkbox test
now dispatches Right/Return through the actual focus owner.

```powershell
go test -tags no_emoji,nodynamic -race -count=1 `
  ./internal/ui ./internal/ui/menus ./internal/ui/widgets ./internal/ui/deletion `
  -run '^(TestCommandAdmission.*|TestWindowCommandAdmissionMatrix|TestApply.*|TestReset.*|TestChoice.*|TestExportPrompt.*|TestExportOptions.*|TestHandleKey.*)$'
go vet -tags no_emoji,nodynamic ./internal/ui ./internal/ui/widgets ./internal/ui/deletion
go run ./scripts/nativeguards -suite command-admission -capture <evidence>/native-guards-final.json
make build
```

All four focused race packages passed (105.825 s, 1.271 s, 3.238 s, 1.858 s);
the ordinary selection also passed. Focused vet and native Make build passed.
The native runner passed the case-insensitive export/reset guard in 0.240 s
with no skip. Changed-file `goimports -local` and whitespace checks passed.
Existing shard assignments and exact Qodana exclusions cover both test files.

Limits, not passed gates:

- A broader exploratory `TestExport.*` selection reached existing host
  assumptions: Win32 stripped a trailing filename space and symlink cases
  lacked the required privilege. These are outside the final focused selection;
  Linux CI must execute their complete coverage.
- `make fmt-check` hit a Git-for-Windows shell fork failure. Direct whole-tree
  goimports included ignored scratch sources and existing CRLF checkout files.
  The eight changed Go files checked clean locally; clean Linux CI must pass
  the whole-repository gate. No unrelated files were reformatted. The combined
  Make invocation stopped before its exclusion target; existing exact paths
  were inspected directly.
- No broad local race suite was duplicated. Fresh CI, CodeQL, code/security
  reviews and inspected post-suppression Qodana SARIF for the pushed head are
  required and recorded on PR 66.

## Local inspections and source identity

GoLand `get_file_problems(errorsOnly=false)` inspected all eight changed Go
files twice, including weak warnings, without timeouts. No callable IDE-local
Qodana interface was available; this is the documented fallback and the active
IDE profile name is not exposed. Seven files had no findings.
`exportoptions_test.go` had two 12-line duplicate fragments at lines 170 and 629:
unchanged independent setup with different assertions, already covered by its
exact `DuplicatedCode` exclusion in `qodana.yaml`. T0 retained those tests and
the existing exclusion, without new suppressions. No actionable local finding
remains; CI SARIF is separate.

Final inspected/native-tested working-source SHA-256 (LF form before Windows
Git checkout conversion), under `internal/ui/`:

| File | SHA-256 |
| --- | --- |
| `commandadmission.go` | `ebbbfe095ae697c67d61d109c8389c1716a4f03db3a97ac185a586ca24c5b8eb` |
| `commandadmission_test.go` | `a725eafbd2bf54835fb1c89c8a18877c10c642d7c4fa4be2333dbb26be0f02b8` |
| `features.go` | `9b97f15290aef722e1215c860cfe553f262646c1a04dac22e760f339e660faa9` |
| `exportoptions.go` | `d2e97a62edfd04790500378981eda73ba17da3d44305c050e3559cab2d0dc915` |
| `exportoptions_test.go` | `53907afcb40e648a7f8ba943835c44511dd5ea2e154bc5b5d0e7655c6e70110a` |
| `deletion/deletion.go` | `3c050f5816ff71f0c341afdfbfa62a07a8d94decbdd3762c4746694c383f72b4` |
| `widgets/choicecard.go` | `eff57f938973a14069cf58cf4d162dc955d00de74f189b188c06310ddc657af6` |
| `widgets/choicepanel.go` | `93f3426b85581e961fab4a047880dae2d746fd4fdb34a86faf30e4c6716bc6aa` |

T0 owns execution, findings, fixes and assessment. One bounded read-only scout
supplied source locations; no implementation/review was delegated. Commit/push
and the PR review loop are authorized; merge/release are not. These initial
automated observations are supplemented by the final physical-keyboard section
above. Linux and macOS qualification still prevent cross-platform acceptance.
