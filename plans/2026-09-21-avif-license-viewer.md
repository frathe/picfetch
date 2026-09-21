# AVIF notices and offline Licenses viewer

Route: Deep (release/Store packaging, resource plumbing and Help UI).
Deliverable: ship the reviewed AVIF/WASM notices, display the release's exact
notice document from Help -> Licenses offline, and reject incomplete packages.

## Decisions and scope

- Preserve the user's existing `todos.md` edits; no commits or publication.
- Keep `THIRD-PARTY-NOTICES.md` authoritative. Embed it in package main and
  pass the immutable text through `ui.Run` to Help before showing the viewer.
- Reuse Help's singleton window and Markdown renderer. License source text
  remains verbatim; translate only the Licenses UI label (English/German).
- Extend the existing final-archive checker, which already checks loose
  notices in all six GitHub archives and both Store payloads.
- Pin AVIF's Go module, embedded payload and build recipe, plus reviewed
  source notices. A changed payload or dependency must require a fresh review.
- No decoder/runtime upgrades, legal-text paraphrasing, runtime downloads,
  updater changes, or release publication. Source provenance uncertainty will
  be recorded explicitly; artifact fixtures do not prove native platform runs.

## Acceptance criteria and agreed test boundaries

1. AVIF notices contain complete reviewed source licenses, including libaom's
   patent grant and relevant WASI/runtime texts. Versions/payload/source hashes
   are checked against the resolved Go dependency.
   Command: `make check-avif-notices` and focused notice-tool tests.
2. Help's public menu opens a Markdown Licenses window with supplied shipped
   text, works without fetching resources, raises an existing window, and
   closes/reopens normally. Main embeds the authoritative file. Wheel scrolling
   works over headings and license blocks in both directions; license blocks
   keep their complete text and monospace style while wrapping with the page.
   Command: `go test -tags no_emoji,nodynamic ./internal/ui/help . -run 'TestLicenses|TestEmbeddedNotices|TestTranslations'`.
3. Every release format rejects absent/stale loose notices or embedded notices
   in its executable, including each architecture of an MSIX bundle.
   Command: `go test ./scripts/updaternotices -run 'TestArtifact|TestEmbedded|TestWorkflow'`.
4. Notice checking runs in CI and local verification. Changed code passes
   focused regression tests and GoLand inspections including weak warnings.
   Commands: `make verify`, Windows cross-vet, and GoLand file inspections.

## Tasks and graph

`T1 provenance -> T2 notice inventory -> T4 final verification`
`T3 viewer ---------------------------> T4`
`T5 artifact guard -------------------> T4`

### T1 — Source provenance
Owner: T3 read-only scout; Lead assesses conclusions.
Files: module cache/upstream only, no repository edits.
Contract: exact dependency versions, payload hash, source/license provenance,
and explicit limits where build inputs are not pinned upstream.
Verify: independently inspect pinned build recipe, payload and source files.
Budget: 1 spawn, 1 Lead assessment, no full suite.

### T2 — Notice inventory and upgrade gate
Owner: T0 inline.
Files: notice document; new AVIF inventory/checker beside updater tooling;
Makefile/CI; packaging documentation; Qodana exclusion for any new tests.
Contract: `make check-avif-notices` checks the selected module/payload and
verbatim source text against a reviewed manifest; generation stays offline.
Test: current inputs pass; missing/changed inputs and stale notice text fail.
Budget: 0 spawns, 1 review round, focused tests only.

### T3 — Embedded offline viewer
Owner: T0 inline.
Files: main.go/main_test.go; internal/ui/run.go; Help implementation/tests;
translations/en.json and de.json; ARCHITECTURE.md.
Contract: main's embedded string reaches Help before window display; Help
provides `SetLicenses(string)` and `ShowLicenses()` with a singleton window.
Test: menu-driven content/lifecycle and embedded-file parity.
Budget: 0 spawns, 1 review round, focused tests only.

### T5 — Finished archive embedding guard
Owner: delegated implementer (artifact_embedding).
Files: scripts/updaternotices/artifacts.go and main_test.go only.
Contract: existing `-artifact` requires exact notice bytes in the executable
as well as existing loose-file checks. Bound reads; propagate full read errors.
Test: missing/stale embedding on every format/Store architecture; chunk edges.
Verify: `go test ./scripts/updaternotices -run 'TestArtifact|TestEmbedded|TestWorkflow'`.
Budget: 1 spawn, 1 Lead review, focused tests only.

### T4 — Final verification and record
Owner: T0 inline.
Files: docs, todos.md, this evidence record.
Verify: all acceptance commands, negative guards, diff review, GoLand, full
`make verify` where the native Linux/amd64 daemon requirement can be met.
Budget: 0 spawns; final suite once; unavailable native checks reported honestly.

## Delegation gate and cost ledger

Both delegated tasks satisfy G1 (bounded prompt), G2 (source commands or focused
tests), G3 (read-only or two exclusive files), G4 (independent context), G5
(Lead has not derived their implementation). Rule S does not replace the source
investigation or behavioral archive guard; Rule W: contracts only, no code in
the plan. Lead retains architecture, UI strings, all review and resulting fixes.
The harness has no repository-named Scout/Implementer models; inherited available
agents perform those roles. At most two run concurrently.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| T1 | 1/1 | 1 | no |
| T2 | 0/0 | 1 | no |
| T3 | 0/0 | 2 (including reported scrolling defect) | no |
| T5 | 1/1 | 1 | no |
| T4 | 0/0 | 1 | attempted; native-daemon gate blocked |

## Evidence and remaining limits

### Implemented

- Retained 26 full license/notice sources covering the AVIF wrapper, libavif,
  dav1d, libaom and bundled helpers, libyuv, WASI libc/compiler support, and
  wazero. The manifest pins source hashes and the exact cached Go-module build
  recipe/payload. Its checker rejects missing/changed files, module replacements,
  dependency drift and stale generated text. No dependencies were upgraded.
- Embedded the canonical document in main and plumbed it to Help before startup.
  Added localized Licenses action, Markdown display, singleton/Escape lifecycle,
  and full-document/embedded-byte parity tests. No notice fetches occur at runtime.
- Extended final archive checking to require the complete canonical document
  inside each executable as well as exact loose files. Tests cover all formats,
  both bundled Store architectures, missing/stale content, scan chunk boundaries,
  and ZIP CRC/gzip trailer failures. CI/Make and dependency-update docs are wired.

### Red/green and regression evidence (2026-09-21)

- Missing embed/menu tests failed before implementation. Notice and artifact
  negative fixtures reject changed or missing inputs; focused suites pass.
- Ronin reported that wheel scrolling stopped over code blocks. A direct canvas
  wheel test reproduced it twice: the heading scrolled down, the code block did
  not. Fyne 2.8's `CodeBlockSegment` contains its own horizontal scroller, which
  consumes vertical events. Replacing those top-level notice blocks with native
  monospace `TextSegment`s and a single vertical document scroller fixes both
  directions without changing any license text. Full-document tests reject
  remaining nested code-block scrollers and verify block style/content parity.
- `go test -tags no_emoji,nodynamic -race ./internal/ui/help ./scripts/avifnotices
  ./scripts/updaternotices . -count=1`: pass after the scrolling fix (Help 42.181s,
  AVIF checker 1.461s, artifact checker 4.394s, main 1.550s).
- Focused root UI menu/Help/startup/Run tests: pass. Windows/amd64 cross-vet:
  pass. `make verify-build`: pass (format, generated/TUF/notice checks, vet/build).
- GoLand inspections with `errorsOnly=false`: all 12 changed Go files clean;
  both scrolling-fix files inspected again after the fix, also clean.
- Offline license tests passed under macOS `sandbox-exec` with `(deny network*)`.
  Rendered top, AOM license/patent section, and document end inspected; regenerated
  screenshots after the scrolling fix preserve readable complete text.

### Package smoke evidence and limits

An isolated checkout in `/tmp/picfetch-license-package.PtcxEZ` preserved the
user's working-tree build metadata/artifacts. Both macOS architectures, Windows
amd64/arm64, Linux amd64/arm64 and Store Windows amd64/arm64 built successfully.
The six unsigned GitHub-layout archives and both Store executable smoke ZIPs
passed the existing final-artifact checker with the new embedded-byte guard.
Mach-O, PE and ELF architectures were checked; Store build tags/module versions
were checked with `go version -m`. These package builds preceded the scrolling
follow-up; the resource plumbing/document are unchanged, and the subsequent
viewer change is covered by the build/test gates above. Store smoke ZIPs are
not signed MSIX files; synthetic tests cover MSIX/bundle structure separately.

`make verify` cannot run its complete race suite here: the Docker daemon reports
Linux/aarch64, not the required native Linux/amd64. The isolation policy was not
weakened or bypassed. Native Windows/Linux offline UI runs and signed MSIX/WACK
qualification also remain unverified.

Upstream provenance still lacks the historical floating libyuv checkout and
attestation of the modified WASI SDK `33.0+m`/libc inputs. Exact reviewed source
references and retained notices are documented in
[`scripts/avifnotices/README.md`](../scripts/avifnotices/README.md); the payload
is pinned, but that cannot prove its historical source closure. These are open
release qualifications in `todos.md`, not a claim of legal certification or
fully attested source correspondence. No commit, push, signing or publication
was authorized or performed. Keep this plan active until acceptance/qualification.
