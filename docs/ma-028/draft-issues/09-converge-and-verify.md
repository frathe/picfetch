# 09: Complete the migration and verify shared ownership

**What to build:** All six application-wide command families now use shared
admission through every applicable route; remove the superseded policy and
prove the integrated behavior and ownership before native acceptance.

**Blocked by:** 02: Preserve text editing and all copy targets; 03: Admit Open,
Close Files and Favorites consistently; 04: Protect file actions without losing
committed effects; 05: Unify ordinary window and Help entry; 06: Admit map and
mosaic visits across preparation; 07: Preserve restricted browsing across sort,
duplicates and search; 08: Share navigation and presentation admission.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. All structural
review, findings, fixes and final deterministic gates remain with T0.

**Context:** [MA-028 spec](../spec.md), all ACs with AC11/AC12 owned here;
the [execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md)
defines the migration ledger and evidence rules. Native AC10 remains ticket 10.

- [ ] Reconcile every actual command against all applicable menu, accelerator,
  shortcut, plain key, bare entry, Host/link, drop/OS and async routes. Record
  N/A reasons, observation/effect owners, intent/target and executed proving
  cases. No omitted family or wrapper-only bare-entry claim. Verify V09.
- [ ] Remove obsolete yielding wrappers, menu admission formulas and duplicated
  cross-feature predicates only once all callers have migrated. Retain feature
  validity, operation tokens, explicit composition and existing lifetime owners.
  Lead structural review demonstrates that a new restricted visit needs shared
  context/policy changes rather than fresh predicates in every adapter. Verify
  against the recorded implementation base; grep/test output alone is insufficient.
- [ ] The complete policy, route, modal, focused-editor, busy-copy, query,
  yield-order, visit and async matrices pass with real effects/absence assertions.
  Cross-family prompt entry/exit and busy entry/exit refresh menus, including
  Favorites/Help/native refresh. Negatively verify new guards and restore the
  implementation before the green run. Verify V09 and all spec AC regressions.
- [ ] Keep exact Qodana exclusions, root-UI shard rows/counts and documentation
  current. Inspect every changed code file including weak warnings; record
  analyzed revision, tool/profile, scope, findings/dispositions and reruns.
  Current Qodana CI is enabled; incomplete scans or licensing errors are not a
  pass, and a fresh post-suppression SARIF is required for a Qodana claim.
- [ ] Run the full Makefile verification once on native Linux/amd64 Docker,
  retain outputs and reconcile failures. Update the backlog/architecture/evidence
  record, leaving final native acceptance open. No commits or PR workflow are
  authorized by this ticket alone.

**Verification V09:**

`go test -tags no_emoji,nodynamic ./internal/ui -list '^TestCommandAdmission'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmission'`

Run every finalized spec AC regression command and compare executed cases with
the ledger; missing/skipped tests fail acceptance. Review
`git diff <recorded-implementation-base> -- internal/ui`, substituting the actual base.

`make check-test-shards`

`make check-qodana-test-exclusions`

`make verify`

Unavailable inspections/CI/native infrastructure remain explicitly unverified.
Do not waive worker isolation or reintroduce the retired Qodana-CI pause.
