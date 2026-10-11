# Restrict ordinary release write authority

## Contract

Address the next medium item in `.scratch/sec_todo.md`. Ordinary tag-release
build tools must not receive repository-write authority. Only the final
publication job needs `contents: write`; checkout must not persist credentials.
Preserve the tag trigger, reusable CI gate, build outputs, manual Windows signing
approval, signed-artifact dependency and publisher authentication.

Route: Standard (one CI hardening change). Lead owns investigation, tests, patch
and review. No delegation: the existing repository agreement reserves reviews
and hot-context fixes for the lead. No new dependency, package, UI string or
runtime behavior. The separate mutable publisher-tag finding stays open.

## Source-backed investigation

`release.yml` grants workflow-wide contents write. `build-macos` and `build-cross`
inherit it while executing external tools. `sign-windows` overrides to read and
uses the protected `release-signing` environment. The reusable `ci.yml` already
sets read permissions. The final `release` job needs write for action-gh-release;
its checkout and both build checkouts retain credentials by default.

Use the existing job-level permission mechanism and checkout input. Artifact
upload/download in the same workflow uses the Actions runtime, not contents
write. Publisher action authentication remains its job token. No changes to
signing secrets or approval policy are needed.

## Tasks and proof

1. Add semantic YAML policy coverage beside the existing standalone release
   checks in `scripts/msixstage/msixstage_test.go`. Reject workflow-wide write,
   any non-publication job write, and persisted checkout credentials. Require
   publication contents write and the existing signing approval/dependency.
   Red/green command: `go test -tags no_emoji,nodynamic ./scripts/msixstage
   -run TestReleaseWorkflowPermissionBoundary -count=1`.
2. Change `.github/workflows/release.yml` and explain the authority boundary in
   `docs/release-signing.md`. Run the focused policy test, owning package and
   adjacent workflow tests, actionlint and GoLand inspections including weak
   warnings; challenge guards against unsafe variants.
3. Run local build/consistency verification, update `todos.md` and the scratch
   checklist with observed results. A real tag-triggered release publishes
   artifacts and requires signing approval; do not create a release for testing.

Graph: 1 -> 2 -> 3. Budget: zero spawns, one inline candidate review, no repeated
broad race suite for this workflow-only change. No new test file, Qodana
exclusion, package map entry or UI shard is needed.

## Evidence

Base: `44b04e0555ade9c64757f585e2e7f4a18e3ea812`, working-tree patch reviewed
on 2026-09-30. The prior Windows fix/review loop is complete at that commit;
[final evidence](https://github.com/frathe/picfetch/pull/75#issuecomment-5913430773)
is on PR #75. Its hosted results do not cover this new workflow patch.

- Red: `TestReleaseWorkflowPermissionBoundary` failed on workflow-wide write,
  inherited build authority, persisted checkouts and missing explicit publisher
  authority. The reusable `test` caller's permission ceiling was also write;
  its called CI jobs already narrowed their own effective permissions to read.
- Green: the same focused command passes after making workflow contents read,
  granting explicit contents write only to `release`, and disabling persistence
  on the three checkouts.
- Negative verification: independently injected a build-job write override,
  re-enabled checkout credential persistence, and removed the signing dependency.
  Every variant failed for its expected policy diagnostic. The intended file was
  restored in `finally` and the final control passed.
- `go test -tags no_emoji,nodynamic ./scripts/msixstage ./scripts/linuxdesktop
  ./scripts/updaternotices ./scripts/wingettag`: all four packages passed.
- `/tmp/picfetch-actionlint-1.7.12/actionlint .github/workflows/release.yml`:
  passed. Existing validated local tool; no shipped dependency added.
- `make verify-build`: passed (exit 0), including formatting, exact Qodana
  exclusions, TUF/assets/notices, full host vet and build.
- GoLand MCP `get_file_problems(errorsOnly=false)` reviewed the workflow and
  modified Go test, including weak warnings. The test has no findings. The
  workflow has only the existing HTTP timestamp warning at its unchanged
  Certum `/tr` endpoint. The [vendor's signing guide](https://files.certum.eu/documents/manual_en/CS-Code_Signing_in_the_Cloud_Signtool_jarsigner_signing.pdf)
  documents this endpoint; timestamp authenticity relies on the signed RFC-3161
  response and the existing SignTool verification, not HTTP transport secrecy.
  This is the same disposition recorded in the earlier PR 30 review; no broad
  suppression or unrelated signing change was added. IDE analysis is a fallback,
  not a new hosted Qodana SARIF result.
- Candidate review (lead, one round): no extra permission grants or authenticated
  Git operations in the affected build/publication steps. Checkout still uses
  its read-capable job token to clone, but does not retain credentials afterward.
  The [publisher action's token input](https://github.com/softprops/action-gh-release/blob/v3/action.yml)
  defaults to `github.token`; its explicit contents write permission remains.
  Tag triggers, CI/build dependencies, the protected signing environment and
  signed-artifact consumption are unchanged. Unspecified job permission scopes
  remain denied by GitHub's existing permission semantics.
- `git diff --check`: passed. Existing test file remains exactly excluded from
  Qodana duplicate-code checks; no new test file, UI test or package map change.

## Review and release qualification

The change is implemented and locally verified. Ronin subsequently authorized
commit/push and a fresh GitHub review loop on 2026-09-30. The final code/security
reviews, CI, CodeQL and fresh Qodana SARIF dispositions for the pushed head are
recorded on [PR #75](https://github.com/frathe/picfetch/pull/75). That mutable
record will hold the final outcome without another status-only commit; this
local evidence does not predeclare a passed hosted gate.
No release tag, publisher run or signing approval was initiated for testing.
The current Docker daemon is ARM64; the native Linux/amd64 full suite remains
unavailable locally. No broad race rerun or relaxed isolation policy was used
for this workflow-only patch. The floating publisher tag is the separate next
medium finding and remains unchanged.

Ledger: zero spawns; one inline candidate review; focused owning/adjacent tests,
three negative guard probes and one local build/consistency gate.
