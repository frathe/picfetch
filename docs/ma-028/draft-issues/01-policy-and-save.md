# 01: Establish shared admission through Save Changes

**What to build:** Save Changes checks whether it can save before disturbing an
idle image-region selection. Menus, its registered shortcut/accelerator and its
bare user entry share that decision; an admitted save still uses the existing
capture and write path. This is the first complete slice of shared admission.

**Blocked by:** None (can start after breakdown approval).

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. This slice owns
the cross-cutting interface and cannot be delegated as a T1 implementation task.

**Context:** [Accepted MA-028 spec](../spec.md), decisions D1-D7; see the
[execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md) for shared
signatures, file locators, budgets and evidence requirements. No new dependencies,
public API or runtime command catalogue. Other command families migrate later.

- [ ] Establish the private, pure request/context/decision contract. Distinguish
  surface, retained visit, modal/editor/menu ownership, operation facts,
  substantive capability, intent and prospective target. Return admission,
  refusal reason and required yield; repeated decisions are deterministic and
  leave inputs unchanged. Verify with V01.
- [ ] Observe existing owners on UI and query availability without feedback,
  cancellation, payload capture, I/O or worker admission. Repeated menu queries
  preserve idle selection. Verify with V01.
- [ ] With nothing savable, invoke the actual Save menu callback, production
  shortcut and bare save entry: retain the selection and start no save. With a
  savable rotation, yield once before capture/write. Change availability between
  query and invocation and prove the latter uses current facts. Verify with V01.
- [ ] Save is refused beneath delete/export/real-dialog owners and during busy
  region copying, with existing attempt feedback where applicable. A menu itself
  is not a modal owner. Native close and owner controls remain available.
  Save menu availability follows the same decision. Verify with V01.
- [ ] Record the implementation base, concrete value fields, Save route evidence
  and the migration inventory format. Keep unmigrated routes working alongside
  the new contract. Observe intended failure before the behavioral fix; record
  focused checks, shard/exclusion updates and changed-code inspection results.

**Verification V01:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionPolicy|TestCommandAdmissionQueries|TestCommandAdmissionYieldOrdering|TestSaveChanges.*)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^save$'`

These guards are to be implemented. Missing cases/skips do not pass. Native
dispatch qualification remains ticket 10; ticket 02 first checks its feasibility.
