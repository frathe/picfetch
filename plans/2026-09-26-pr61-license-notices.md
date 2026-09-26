# PR 61 license notices and FOSSA dispositions

Route: Standard. Repository-only follow-up requested on 2026-09-26.
Base: PR 61, `cf24b842e471a3f5dc32e130b38ed8798b76f2b0`.
Worktree: isolated from the user's active checkout. The user subsequently
authorized a commit and push to PR 61; no merge, release or FOSSA write is in scope.

## Scope and decisions

- Complete the Fyne font and Windows GLFW header notices using the exact
  dependency versions already shipped. No dependency or runtime changes.
- Retain the existing MIT project license and all LGPL/GPL/LLVM notice texts.
- The eight exported findings are policy flags; the GitHub status reports 13.
  Repository changes cannot approve issues in the existing FOSSA policy.
- Keep the remote gate enabled. Document narrow project/version decisions and
  require File Matches for provisional findings. No speculative scan exclusions.
- Test the existing boundary: the canonical shipped notice document, against
  pinned upstream source texts. Existing archive/embedded-resource tests cover
  delivery. No new production interface, package, or source test seam.

## Tasks and acceptance evidence

1. Lead: add an exact-source regression in the existing updater-notices test
   package, see it fail for the omitted notices, then supply the full texts.
   Files: `scripts/updaternotices/main_test.go`, retained LGPL text,
   `THIRD-PARTY-NOTICES.md`, `Makefile`, updater-notices README.
   Verify: `make check-updater-notices check-avif-notices` and
   `go test -tags no_emoji,nodynamic -race ./scripts/updaternotices ./scripts/avifnotices`.
2. Research scout: official FOSSA configuration and issue-scope facts only.
   File: `docs/fossa-license-ci-2026-09-26.md`. Lead owns the finding assessment.
   Delegation: bounded unfamiliar product-documentation search; no implementation
   overlap, no source review, one output checked against primary sources.
3. Lead: record source/build evidence and reviewed/provisional dispositions in
   the guide; update `todos.md` with the remaining FOSSA work.
   Verify: `git diff --check`; source links and issue IDs match the supplied CSV.
4. Lead: run focused notice/viewer tests, formatting, and GoLand inspections of
   changed Go code. Full verification follows the repository's native-amd64
   Docker gate where available; incomplete checks stay explicit.

## Dependency obligations

- Fyne v2.8.0: DejaVu/Bitstream/Arev, Noto and Inter font notices. Preserve full
  supplied license texts, font copyright attribution and exact module URL.
- GLFW `v0.1.0-pre.1.0.20260707082822-2a407d02d01a`: retain embedded GLFW's zlib
  license and Wine/Andrew Fenn header attribution. The Windows headers contain
  interface declarations/layouts/constants and short call macros; review under
  LGPL 2.1 section 5, separately from any runtime linking claim. Supply the full
  LGPL 2.1 text, pinned source URL, and unmodified-source availability.
- Retain original source hashes. Only CRLF-to-LF normalization is permitted in
  the Markdown reproduction. No licenses are shortened, translated or removed.

## Evidence

- RED: the new exact-source test failed for the omitted font/header license
  texts, source URLs and retained LGPL file before the notice changes.
- GREEN: focused desktop-notice and workflow checks passed; then
  `make check-updater-notices check-avif-notices` passed. No module versions or
  dependency checksums changed.
- `make verify` passed on 2026-09-26 against this PR worktree on a verified native
  Linux/amd64 Docker daemon: formatting, metadata/notice gates, vet, build and
  the complete race suite (non-UI plus all three UI shards). The notice packages,
  archive checks and Help's complete offline-notice rendering test passed.
  Artifacts: `.scratch/race-runs/20260926T145335Z-pJPnax`; host/container exit
  codes are zero, with no failed test or data-race events in the captured JSON.
  Analyzed content is the eight-file change over the base above; the changed
  Go file SHA-256 is
  `121855ddf24b888058003615dceacffd8a72819325052dfacbcc475ef5a2ac1f`.
  Subsequent edits update documentation evidence only.
- GoLand fallback inspection/build were attempted for the only changed Go file,
  `scripts/updaternotices/main_test.go`, with all severities requested. The IDE
  rejected the path because the isolated PR worktree is outside its open project;
  selecting that worktree as `projectPath` also failed because it is not open.
  No inspection results were returned: IDE analysis remains unverified, not
  clean. The user's active checkout is deliberately left untouched. Qodana CI
  remains explicitly waived under the standing repository instruction.
- `git diff --check` identifies two trailing spaces copied verbatim from Fyne's
  DejaVu license. They are retained intentionally to preserve exact upstream
  text; the staged whitespace check excluding that document passed for every
  other changed file. Lead review confirmed no other whitespace findings,
  dependency changes, omitted pre-existing notices or CI gate exclusions.
- Remote FOSSA remains pending authenticated project-owner review, including
  File Matches for provisional findings and the 13-versus-eight discrepancy.

## Cost ledger

| Task | Spawns budget/actual | Review ownership | Full suite |
| --- | --- | --- | --- |
| Repository changes | 0/0 | Lead | Final gate only |
| FOSSA documentation | 1/1 | Lead | No |
