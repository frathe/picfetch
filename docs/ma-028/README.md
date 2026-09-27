# MA-028: shared command admission

This is the authoritative, portable tracker for MA-028. The user approved moving
its planning Markdown from `.scratch/ma-028` on 2026-09-27 for cross-desktop
qualification. Other features retain the usual local-tracker convention.

## Start here

- [Accepted specification](spec.md): decisions, acceptance criteria and native
  test procedure.
- [Ticket 10: completed qualification](issues/10-native-qualification.md):
  native/operator evidence and the clean acceptance round. All ten tickets are
  resolved; MA-028 is complete.
- [Execution plan and complete ticket index](../../finished_refactorings/2026-09-27-ma-028-command-admission.md):
  dependencies, implementation history and verification ownership.
- [Implementation verification](../command-admission-verification-2026-09-27.md)
  and [Linux qualification](../command-admission-linux-qualification-2026-09-27.md)
  and [Windows qualification](../command-admission-windows-qualification-2026-09-27.md)
  and [macOS qualification](../command-admission-macos-qualification-2026-09-27.md):
  completed checks, precise scope and remaining limitations.
- [PR 66 review-loop evidence](../command-admission-pr-66-review-2026-09-27.md):
  static-analysis dispositions, fresh reviews and external verification.
- [Interview](interview.md) and [design](../command-admission.md): accepted
  decisions and their rationale. `draft-issues/` contains historical ticket
  drafts, not current status; use `issues/` for ongoing updates.

## Evidence and privacy

The planning documents are tracked; raw screenshots, logs, binaries, fixtures,
mutation files and checksums remain ignored under `.scratch/ma-028` on the
original Linux, Windows and macOS workstations. Paths to those artifacts in the summaries are
optional local audit references, not requirements for another checkout.
The [privacy audit](../ma-028-portability-audit-2026-09-27.md) records the review,
redaction and exclusions. The interview uses `$GOMODCACHE` for its pinned Fyne
source reference; it contains no workstation home path.

For future relevant changes, use the spec's native procedure and the completed
ticket 10 evidence as the regression baseline. Physical-input checks remain
distinct from automated native-menu/filesystem guards. PR 66 retains the exact
latest-head review/CI disposition, including the documentation-only closure.
