# Help menu release notes and asynchronous artwork

Help -> Release Notes and the post-update What's New window read this build's
bundled release notes. Both offer a browser link to older GitHub releases.
Optional Markdown images load online without blocking the window. Notes without
images remain a normal text view and start no image requests.

Route: Standard initially; Deep follow-up for worker lifecycle, shutdown and
root-harness wiring. No dependency changes, new bundled artwork, persistent
image cache or release publication. No commits authorized.

## Decisions and acceptance

- `.github/release-notes.md` remains authoritative because Store tooling reads
  current and historical tags. `make release` copies it into help in the same
  commit as the version bump; parity/version tests guard drift.
- The update cache controls one-time display only. Its old Body field remains
  backward-compatible but is not used for the displayed notes.
- Images are optional and fetched from each document's own HTTP/HTTPS URLs,
  including query strings and redirects. No asset-specific mapping or bundle.
  Text is immediately readable; fixed-height image slots show loading/failure.
- At most three image workers per window perform both HTTP I/O and decoding.
  Requests have context cancellation and a 20-second client timeout. Responses
  are capped at 8 MiB and 16 megapixels before decode. Standard PNG/JPEG/GIF
  (first frame) and existing WebP support add no dependency/license obligations.
- Per-instance UIQueue delivery rechecks the session before publishing pixels.
  Close cancels; reopening starts a fresh session. Stop ends admission; Wait
  and Settle observe current/retired workers off UI. Root shutdown and harness
  cleanup include Help; tests use local HTTP or in-memory transports.
- English/German UI labels and manuals remain synchronized. Release prose stays
  in its published language. Older notes are browsed on GitHub.

AC1: Help menu, version/body, singleton, Escape, reopen and explicit history link.
`go test -tags no_emoji,nodynamic ./internal/ui/help -run 'TestHelpMenu|TestReleaseNotes|TestShowWhatsNew' -count=1`

AC2: Generic optional artwork; nonblocking loading, cancellation, stale-delivery
rejection, error fallback, nested list/table images, no requests for text-only
notes, supported formats, URL queries/redirects and bounded decoding.
`go test -race -tags no_emoji,nodynamic ./internal/ui/help -count=1`

AC3: Post-update window uses bundled notes even when cached prose is obsolete;
marker matching/clearing stays intact, translations remain complete.
`go test -tags no_emoji,nodynamic ./internal/ui -run 'TestMaybeShowWhatsNew|TestWhatsNewCache' -count=1`
`go test -tags no_emoji,nodynamic . -run TestTranslations -count=1`

## Task ownership and delegation

1. Lead owns architecture, UI/lifecycle, menu, root update routing, tests, docs,
   all review and fixes, and final verification.
2. Initial read-only scout traced release/Store artifact ownership and ordering.
3. One independent helper implementer owned only `releaseimage.go` and its test.
   Fixed contract: `loadReleaseImage(context.Context, *http.Client, string)
   (image.Image, error)`. G1 bounded prompt; G2 focused httptest command;
   G3 two exclusive files; G4/G5 independent transport/decoder context while
   lead implemented UI; S/W no scripted transform or implementation prescription.
   Lead independently reran tests and reviewed the helper and its limits.

Graph: artifact recon -> menu/text -> asynchronous UI and independent HTTP helper
-> update routing -> focused tests -> lead review -> final gates.

## Verification evidence

- Initial red: missing Help action/bundle/history link. Initial feature/package,
  manual/release-generator and translation tests passed after implementation.
- Online-image red: the notes rendered but the local server never received a
  request. Green: image request/decoding completes off UI and renders afterward.
- Post-update red: the window displayed `obsolete cached release notes`.
  Green after routing through ShowReleaseNotes: bundled body, cache clearing,
  empty cache and version-mismatch tests pass (`internal/ui 0.403s`).
- Nested-image red: only the list image was prepared (1 of 3). After traversing
  table headers/cells, all three are prepared before Fyne layout.
- Complete Help package passes (`2.852s`); complete Help race suite passes
  (`33.460s`). Focused helper tests passed in the lead's suite as well as the
  implementer's run. Its initial red was compile-only, then disabling byte and
  pixel guards made their tests fail for the intended reasons before restoration.
- Root translation checks pass (`0.464s`). `make vet` and native
  `go build -tags no_emoji,nodynamic ./...` pass with portable MinGW on PATH.
  Changed-file goimports reports no files and `git diff --check` passes.
- Captured/inspected the actual Fyne rendering after loading the Trane source
  through a local HTTP server: `.scratch/release-notes/online-art-preview.png`.
  The artwork is visible with preserved proportions; text and history link
  remain in the scroll/window surface. Temporary capture test was removed.
  No artwork file or generator changes remain. No external image host was
  contacted by tests, and no golden screenshots were changed.
- Existing root menu-structure test assumes English but sees `Datei` under the
  German Windows locale (`menu_test.go:41`). Other focused menu tests passed;
  this unrelated locale assertion was not changed or described as green.
- Full `make verify SHELL=sh.exe` remains environment-dependent: prior attempt
  failed at check-test-platform because Git Bash cannot fork (0xC0000142,
  make Error 254). Docker reports native linux/x86_64 but only 16,592,285,696
  bytes, below the required 16 GiB. No limits or tests were relaxed.
- GoLand inspection remains unavailable: no callable inspection tools and no
  connected document sessions. No clean IDE result is claimed.
- The new helper test is listed exactly in Qodana exclusions. Existing root UI
  tests were extended; no new top-level root tests require shard assignments.
- Concurrent Windows Make compiler/README/todos edits were preserved.

- Final focused race run, including the nested-image change and post-update
  integration, passes: `internal/ui/help 2.892s`, `internal/ui 2.313s`.
  Command: `go test -race -tags no_emoji,nodynamic ./internal/ui/help ./internal/ui
  -run 'TestReleaseNotes|TestShowWhatsNew|TestLoadReleaseImage|TestMaybeShowWhatsNew|TestWhatsNewCache' -count=1`.
  Output is retained in `.scratch/release-notes/online-race.log`.
- Repeated full-gate attempt confirms the same external blocker:
  `make verify SHELL=sh.exe` exits at check-test-platform with fork failures
  and make Error 254. It does not reach the complete Docker/race suite.

Implementation, lead review and focused verification are complete. Keep this
evidence in plans until environment-dependent gates are completed.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Artifact reconnaissance | 1 / 1 read-only | lead source check | no |
| UI/lifecycle/update routing | 0 / 0 | 1 lead per phase | no |
| Independent HTTP helper | 1 / 1 | 1 lead | no |
| Final gates | 0 / 0 | lead | attempted; environment blocked |
