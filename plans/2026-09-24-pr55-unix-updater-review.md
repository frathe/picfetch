# PR #55 Unix updater review

Route: Standard. The PR description is the spec: reject planted predictable
`.new` symlinks while preserving same-directory copy, rename, and rollback.
The existing `internal/update` apply and startup cleanup boundaries are the
test seams. No new dependency, package, translated string, merge, or release.

## Findings and acceptance criteria

1. A crash after `CreateTemp` can leave `.picfetch.new-*` or
   `.Info.plist.new-*` beside the installed app. A later launch must remove
   stale regular staging siblings, preserve recent files, unrelated files,
   links and backups, and retain the legacy `.new` / `.apply.cmd` sweep.
   Verify: `go test -tags no_emoji,nodynamic ./internal/update -run 'TestSweepLeftovers_' -count=1`.
2. Preplanted fixed `.new` symlinks must not be followed, and a normal update
   must install both binary and plist with the intended modes.
   Verify: `go test -tags no_emoji,nodynamic ./internal/update -run 'TestApplyUnix_' -count=1`.
3. Updater package regressions, formatting, vet, and build must pass locally;
   GoLand must inspect changed Go files. The invoked GitHub review loop lets CI
   own the broad race suite.
   Verify: `go test -race -tags no_emoji,nodynamic ./internal/update -count=1`,
   `make fmt-check`, and `make verify-build`.
4. A fresh Codex code and security review on the latest pushed commit must
   finish without actionable findings; required CI, Qodana, and CodeQL must
   pass. Verify using the PR's commit-bound reviews and checks.

## Tasks

1. Lead: add one interrupted-update cleanup regression, observe it fail, then
   implement bounded-memory sibling scanning on Unix. Keep one generated-name
   prefix shared by staging and cleanup. Budget: no delegated implementation.
2. Lead: update the stale `ARCHITECTURE.md` updater map and the Done changelog
   in `todos.md`; review the final diff and local checks.
3. Lead: commit and push, reply to the finding, resolve its thread, and obtain
   a fresh Codex review and CI results.

One read-only CI scout inventories hosted checks while the lead reviews and
fixes. It has a bounded PR/SHA scope, zero edits, and no review authority;
its results are checked against GitHub. Budget: one scout, one full hosted suite.

## Evidence

- Initial PR head: `2e10bbd35d2a09f083102f4aa5774b9fcfc07bde`.
- Initial Codex code review found the interrupted-update cleanup gap; security
  review completed without findings. Qodana and CodeQL passed on this head.
- Standards: the updater rows in `ARCHITECTURE.md` still describe fixed `.new`
  staging, and need to match the corrected behavior. No other standards or
  spec deviation confirmed in the initial diff.
- Red: `TestSweepLeftovers_RemovesInterruptedUnixTemporarySiblings` failed
  because both randomized binary and plist files survived. It uses the
  existing startup sweep seam and also checks that fresh files, unrelated
  names, links, directories, the installed executable, and its backup stay.
- Green: focused Unix apply/sweep tests and the entire `internal/update` race
  package passed. `make verify-build` passed formatting, TUF, generated assets,
  notices, vet, and build. The Windows updater test binary cross-compiled.
- The GoLand inspection service only accepts its open project at
  `/home/finis/Projects/picfetch`; it rejected the isolated PR worktree at
  `/tmp/picfetch-pr55-20260924` as outside that project. Changed-file GoLand
  warnings remain unverified locally; hosted Qodana is the available inspection.
- The five-minute grace avoids removing another instance's active staging
  file. A file left by a crash within that interval is cleaned on a later
  launch after it has aged; no timed background cleanup runs while PicFetch is
  open.
