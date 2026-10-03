# Immutable ordinary release publication actions

## Contract and scope

Address the last medium finding in `.scratch/sec_todo.md`, after the completed
PR 75 review loop at `8505c434f4df247a5450b8c118b4e9b6b40886d9`.
The ordinary release publisher must execute a reviewed immutable action commit.
Every peer action in the same `contents: write` job shares that authority, so
pin the entire publication job. Preserve versions, inputs, artifact names,
notice checks, manual signing approval, permission boundary and tag trigger.

Route: Standard, lead inline; zero delegates. Files: release.yml, existing
msixstage_test.go, this plan and todos.md. No application code, dependencies,
packages, test files or UI strings are added. Other jobs' build-tool trust is
outside this publication-action finding.

## Source and compatibility review

The previous floating `softprops/action-gh-release@v3` resolves to the same
commit selected below (3.0.3). Official GitHub commit APIs report `verified=true`
and `reason=valid` for all four revisions, checked 2026-09-30. The publisher's
immutable action.yml and src/run.ts/src/util.ts retain tag detection, `dist/*`
file expansion, body_path reading and the default `github.token` input.
The peer actions' metadata retains all current inputs and Node 24 entry points.
No runtime version upgrade is part of this change.

| Action | Existing ref | Selected full commit |
|---|---|---|
| softprops/action-gh-release | v3 (3.0.3) | `efb35369e0ad2afab669f228072c1b0d510eae64` |
| actions/checkout | v7 (7.0.1) | `3d3c42e5aac5ba805825da76410c181273ba90b1` |
| actions/download-artifact | v8 | `3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c` |
| actions/setup-go | v7 | `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` |

Sources are the corresponding official GitHub repositories at these revisions.
Each LICENSE was read and is MIT: retain copyright/license when copying or
redistributing their software. These actions and their bundled Node dependency
closures execute only on CI runners; they are not copied into PicFetch or its
release archives. The same selected revisions already back the current floating
refs. No shipped dependency or third-party notice change is needed. Runner action
checkouts retain upstream license files. Existing weekly GitHub Actions Dependabot
coverage keeps future pin updates reviewable; version comments remain beside pins.

Publisher metadata: https://github.com/softprops/action-gh-release/blob/efb35369e0ad2afab669f228072c1b0d510eae64/action.yml
Publisher license: https://github.com/softprops/action-gh-release/blob/efb35369e0ad2afab669f228072c1b0d510eae64/LICENSE

## Tasks and verification

1. Add semantic YAML coverage for all publication action refs and unchanged
   publisher files/body inputs. Red: `go test -tags no_emoji,nodynamic
   ./scripts/msixstage -run TestReleasePublicationActionsPinned -count=1`.
2. Pin all seven action steps (four unique upstream actions) in the publication
   job. Green with the same command. Independently restore a mutable publisher
   tag and a mutable peer-action tag; require both variants to fail, then restore.
3. Run owning/adjacent workflow package tests, actionlint, GoLand inspections of
   both code/config files including weak warnings and `make verify-build`.
   Review the final diff and record evidence. Full suite remains hosted CI;
   a real release upload requires a separately authorized tag/signing run.

Graph: regression -> pins -> qualification. Budget: zero spawns, one inline
candidate review, one build/static gate, no broad local race rerun.

## Evidence and remaining qualification

- Red: `TestReleasePublicationActionsPinned` rejected all seven floating refs.
- Green: both publication-pin and release-permission guards pass. Five negative
  probes reject a mutable publisher, mutable peer, abbreviated SHA, missing
  publisher and changed artifact input. The final workflow was restored after
  each run and passed as the control.
- `go test -tags no_emoji,nodynamic ./scripts/msixstage ./scripts/linuxdesktop
  ./scripts/updaternotices ./scripts/wingettag` passed (owning package 3.512s).
- Existing actionlint 1.7.12 passes. `make verify-build` exited 0, including
  format/TUF/generated-input/notice/exclusion checks and full host vet/build.
  `git diff --check` passes.
- GoLand inspected both changed code/config files including weak warnings. The
  Go test is clear. At the new immutable publisher ref, its metadata inspection
  incorrectly reported `files` and `body_path` undefined; the pinned upstream
  action.yml explicitly declares both. Two input-local `UndefinedParamsPresent`
  suppressions were added and reinspection confirms those warnings are gone.
  The [inspection documentation](https://www.jetbrains.com/help/inspectopedia/UndefinedParamsPresent.html)
  identifies this exact suppression. The unchanged Certum HTTP timestamp weak
  warning retains its source-backed disposition from the permission plan.
  Policy tests and actionlint passed again after the comment-only suppression.
- Candidate review (lead): all seven publication action steps use full SHAs;
  no tag or abbreviated ref remains in that write-enabled job. Selected refs
  resolve to the current upstream versions; Node entry points, publisher inputs,
  notice checks, artifact names and protected signing dependency are unchanged.
  The earlier permission guard still passes. Existing exact test exclusion
  applies; no new test file or shard entry is required.

Implementation and local verification are complete. Hosted results for the
pushed head will be recorded on PR 75. Pinning fixes mutable resolution; it does
not prove all upstream software safe or remove other build-tool trust. Actual
publication remains qualification for the next separately authorized tag release;
no release was created for testing. Full native Linux/amd64 suites run in hosted
CI because the local Docker daemon is ARM64.

Ledger: zero spawns, one inline candidate review, five negative probes, one
owning/adjacent package run and one local build/static gate.
