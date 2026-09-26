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
  File Matches for provisional findings. The follow-up below reconciles the
  initial count discrepancy without claiming that the remote gate passed.

## Follow-up: complete original export at 16:17 UTC

Scope: finish the existing Standard investigation with a documentation-only
update to this evidence, the disposition guide and `todos.md`. Preserve all
notices, dependencies, application behavior and CI enforcement. Lead owns the
classification and all changes; one bounded read-only scout gathers x/text
source/build-selection evidence while the lead checks go-digest and report
provenance. Delegation is independent, read-only, specified by exact versions
and targets, and verified by the same six `go list` commands. No review or fix
is delegated. No new unit test is appropriate for an account-side policy
decision; the actual remote status remains the pass/fail signal.

- The second ZIP SHA-256 is
  `9ce43b4ee151118cf2e91911924c941fcbcd42f402cbc9e1d5e8c1c25eedc18b`.
  Parsed both CSVs by issue ID: 8 prior rows, 13 current rows, no removed or
  changed prior rows. Added IDs 21232394-21232398 are five Denied CC findings.
  Both exports have `analyzedAt=2026-09-26T14:25:00.119Z` and root version
  `cf24b842e471a3f5dc32e130b38ed8798b76f2b0`, not notice-fix `f0ed64a`.
- `gh api repos/frathe/picfetch/commits/f0ed64a0ea319d16bd49013ab13c580d3b6355b3/status`
  reproduces the remaining failure: License Compliance is `error`, description
  `15 issues found`, updated 15:05:37 UTC. Security Analysis and Dependency
  Quality succeed. The supplied export cannot identify the two extra findings.
- go-digest v1.0.0 assigns its docs license only to README/contribution docs;
  all five selected Go files are Apache-licensed and `EmbedFiles` is empty.
  Packaging does not copy the upstream documentation; existing notices retain
  the Apache license. No additional notice or dependency change is justified.
- x/text v0.42.0's CC URL references are in test samples. All six production
  `go list -mod=readonly -tags=no_emoji,nodynamic -deps -json .` commands
  completed without package errors with cgo enabled. Package counts are
  Darwin 815/814, Linux 823/822 and Windows 819/818 (amd64/arm64). Each target
  selects 29 x/text packages, without `internal/testtext`, `cases` or the
  snippet-bearing test files. This is selection evidence, not cross-build proof.
- FOSSA access remains unavailable: no connector/CLI/API key and the Browser
  skill's supported discovery returned no available browsers. Requested the
  latest revision's export/File Matches from the user; no issue resolutions or
  global policy changes were made.
- Verification: `make fmt-check check-updater-notices check-avif-notices`
  and `git diff --check` passed. A CSV-to-guide check confirms all 13 exported
  issue IDs appear exactly once; all local Markdown links resolve. Lead reran
  the six-target dependency-selection assertions successfully and confirmed
  Apache headers on all five selected go-digest source files.
  The passing full `make verify` evidence for unchanged code/notices at
  `f0ed64a` carries forward; do not repeat the full race suite for prose edits.
  The earlier GoLand limitation remains unverified, not cleared by this update.

## Cost ledger

| Task | Spawns budget/actual | Review ownership | Full suite |
| --- | --- | --- | --- |
| Repository changes | 0/0 | Lead | Final gate only |
| FOSSA documentation | 1/1 | Lead | No |
| Follow-up x/text source/build lookup | 1/1 | Lead | No; unchanged code |
