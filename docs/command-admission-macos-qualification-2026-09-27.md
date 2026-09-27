# MA-028 macOS qualification

Status: macOS desktop qualification complete, including physical editor input.
Linux and Windows are also complete. Fresh PR 66 gates remain before closure of
[ticket 10](ma-028/issues/10-native-qualification.md).
The user requested completion, commit/push and the PR 66 review loop before
ticket closure. No merge or release is authorized.

## Environment and isolated launch

Source revision: `41cdaa9ae63ac404fd5c30041145c694ddf766f8`; macOS 27.0
(26A428), arm64; Go 1.27.1, pinned Fyne 2.8.0. Product source is unchanged.
Raw evidence stays ignored under `.scratch/ma-028/macos-2026-09-27`.

Literal `make run` runs its ordinary `go run -tags no_emoji,nodynamic .` target.
Four evidence-only Go overlays change the app identity to
`io.github.frathe.picfetch.ma028.20260927` and redirect Favorite, preset and
update-stage roots into the evidence profile. Fyne preferences/session data
use the separate identity beneath the ordinary macOS Library roots. `HOME`
is unchanged. No trial mode, command policy, input handling, clipboard logic,
decoder or worker lifetime is substituted. Overlay source/output hashes and
the exact launcher are retained in `overlay-manifest.json`, `prepare.mjs`,
`overlay.json` and `launch-app`. This is a qualified ordinary app with storage
substitutions, not a claim of a byte-identical production binary.

The `go run -exec` wrapper copies the compiled executable into a temporary
app bundle and executes that copy so native desktop tooling can address it.
Executable SHA-256:
`4714c46b14c9e731798e59292d4e74f33e07cc8543b20dce666b9e82e8e0bc5e`.
It carries Go's ad-hoc linker signature. Initial setup attempts are excluded:
the desktop tool could not address an unbundled process; the wrapper initially
matched only Go's temporary executable path rather than its cached path,
leaving an empty bundle and a Finder missing-executable error. Correcting the
path match produced the running, addressable app. Both earlier owned processes
were terminated before the successful session; this was a harness failure.

Three generated 480 x 320 opaque PNGs form the test collection. The native Open
chooser loaded `fixture-1.png` and its two siblings. No user library is used.
Clipboard backup/probes and screenshots remain local and are not published.
Computer-use actions supply OS-injected input; separately recorded physical
Cmd+A/C must come from the operator.

## Focused native guards

The unchanged source passed:

```text
go run ./scripts/nativeguards -suite command-admission -capture <evidence>/native-guards.json
ok github.com/frathe/picfetch/internal/ui 0.572s
PASS required: TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset
PASS required: TestSetMenuItemModifierMask_ClearsDefaultCommand
```

Both required guards executed without skipping. Their isolated AppKit and
filesystem observations are distinct from the full desktop procedure below.
The initial sandbox cache denial and missing evidence-directory attempt are
setup failures, not test results; the recorded final command completed cleanly.

## Desktop outcomes

The user physically performed the requested Cmd+A/C sequence in the image
and Grid naming fields and confirmed each in chat. T0 seeded an unrelated
clipboard sentinel after preparing each field, without injecting either
shortcut. The clipboard then contained exactly `MA028 IMAGE physical 927` and
`MA028 GRID physical 927`, respectively. Image title/index stayed unchanged;
Grid retained only its first selected thumbnail. Decoded screenshot comparisons
found zero differences across 109,312 exposed image pixels and 180,480 pixels
covering all three Grid thumbnails and their selection borders. The same Grid
rectangle was identical before opening and after cancelling its dialog.
`physical-results.json` records regions and outcomes; captures 04-11 retain the
observations. The screenshots were captured after chat confirmation returned
focus; their inactive fields do not independently prove selection highlighting.
Raw computer-use snapshots contain JPEG bytes despite the initial `.png`
filenames; the comparison decodes the actual format.

| Native scenario | Observed result | Local evidence |
| --- | --- | --- |
| Actions -> Copy image; separately native Cmd+C, each following an unrelated text sentinel | Each copied exactly all 153,600 fixture pixels. | `menu-copy.png`, `native-copy.png`, captures 13-15 |
| In Add to Favorites, attempt Cmd+S/Cmd+E and inspect Actions; attempt its disabled Copy image | Modal retained; all Actions items disabled; blocked native menu activation preserved the independent clipboard sentinel. | Captures 16-19 |
| Add to Favorites: Tab, Right, Return | Its own Add control saved exactly the three fixture paths beneath the isolated Favorite root. Earlier Escape controls cancelled the physical-check dialogs. | Captures 21-23, isolated `file-list.json` |
| Export: conflicting Cmd+S; Up/Return on metadata; Tab/Escape; reopen and Return on PNG | Prompt retained ownership, metadata toggled, Escape dismissed it and PNG entered the real native Save dialog, which was cancelled. | Captures 24-27 |
| Actions -> Move image to Trash; Cmd+E, Tab, Return on Cancel | Delete prompt retained ownership and its Cancel control dismissed it; all fixture hashes remain unchanged. | Capture 29 and fixture hashes |
| Rotate, Actions -> Copy selection, operator drag, Return; hold successful clipboard completion | Published a real region PNG while its launcher remained held. File Open/Save/Export/Close disabled. Actions disabled conflicting commands while retaining zoom and copy. | Captures 41-47, `region-held.png`, launcher PID 23086 |
| Attempt disabled Save; Cmd+S; Cmd+Shift+C while held | Save shortcut showed the expected finishing-copy feedback; the region PNG stayed byte-identical and the fixture hash was unchanged. | Captures 44-46, `region-after-refusal.png` |
| Release the launcher, then Actions -> Copy image path | The launcher exited, native menus re-enabled, and Copy Path published the exact fixture path. | Captures 49-51 |
| Start a second operator-drawn region copy; click native close while its Copy button is disabled and completion is held | The real region PNG was present; Make/app exited 0 before the explicit release marker was created. No app/held-launcher process remained afterwards. | Capture 55, `region-close-held.png`, `busy-close-result.json`; ready PID 25005 |
| New literal Make-run session: load fixture, open Add to Favorites, click native close | Modal was still visible at close; Make/app exited 0. All three fixture hashes are unchanged and the pre-test clipboard was restored. | Captures 56-58, terminal exit and process observations |

Automated drag attempts produced no selection rectangle and are excluded.
Operator-created rectangles supplied both real regions; the first was submitted
with automated native Return, the second with the requested operator Return.
No application fix was needed. Computer-use menu/keyboard/click automation and
operator-performed physical input remain separately identified above.

The clipboard hold forwards PicFetch's unchanged `osascript` arguments to the
system launcher, then holds only a successful PNG publication while an owned
marker is present. A release marker or ten-minute bound ends the hold. This is
an external process-completion hold, matching earlier Linux/Windows evidence;
it is not an instance encoder seam or a replacement clipboard implementation.

## Verification and ownership

T0 owns native execution, assessment and any fixes. One read-only scout located
profile and clipboard seams; T0 verified the source citations. No review or
implementation was delegated. Unchanged product-source test/GoLand evidence
carries forward at its recorded revisions. Fresh remote code/security reviews,
required CI/CodeQL and an inspected post-suppression Qodana SARIF remain required
on the latest pushed head. Full race-suite execution belongs to CI for this loop.

Additional focused macOS race verification passed on unchanged source:
root UI command-admission/window/Export-prompt tests in 62.373s; Choice widget
tests in 2.342s; menu availability tests (`TestApply_`) in 1.414s. The initial
multi-package filter selected no menu/deletion tests; those empty selections
are not claimed as coverage. Native guards passed separately above.
