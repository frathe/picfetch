# MA-028: shared command admission

This is the authoritative, portable tracker for MA-028. The user approved moving
its planning Markdown from `.scratch/ma-028` on 2026-09-27 for cross-desktop
qualification. Other features retain the usual local-tracker convention.

## Start here

- [Accepted specification](spec.md): decisions, acceptance criteria and native
  test procedure.
- [Ticket 10: remaining qualification](issues/10-native-qualification.md):
  current native/operator gates and completed CI evidence. Tickets 01-09 are resolved;
  overall acceptance remains open.
- [Execution plan and complete ticket index](../../plans/2026-09-27-ma-028-command-admission.md):
  dependencies, implementation history and verification ownership.
- [Implementation verification](../command-admission-verification-2026-09-27.md)
  and [Linux qualification](../command-admission-linux-qualification-2026-09-27.md):
  completed checks, precise scope and remaining limitations.
- [PR 66 review-loop evidence](../command-admission-pr-66-review-2026-09-27.md):
  static-analysis dispositions, fresh reviews and external verification.
- [Interview](interview.md) and [design](../command-admission.md): accepted
  decisions and their rationale. `draft-issues/` contains historical ticket
  drafts, not current status; use `issues/` for ongoing updates.

## Evidence and privacy

The planning documents are tracked; raw screenshots, logs, binaries, fixtures,
mutation files and checksums remain ignored under `.scratch/ma-028` on the
original Linux workstation. Paths to those artifacts in the summaries are
optional local audit references, not requirements for another checkout.
The [privacy audit](../ma-028-portability-audit-2026-09-27.md) records the review,
redaction and exclusions. The interview uses `$GOMODCACHE` for its pinned Fyne
source reference; it contains no workstation home path.

On the next desktop, use the spec's native procedure and ticket 10's unchecked
items; record revision, OS/architecture, input route and actual outcomes there.
Physical-input checks remain distinct from automated native-menu/filesystem
guards. This move and PR publication do not establish those acceptance results.
