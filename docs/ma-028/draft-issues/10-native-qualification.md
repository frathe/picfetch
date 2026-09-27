# 10: Qualify native input and accept the integrated migration

**What to build:** Establish actual Linux, Windows and macOS desktop evidence
that editor shortcuts reach text, menus reflect current admission, and native
dispatch cannot bypass the integrated policy. Close MA-028 only with complete
deterministic and native evidence.

**Blocked by:** 09: Complete the migration and verify shared ownership.

**Status:** draft (publish as ready-for-human for native operation, with T0
agent support; change to ready-for-agent only if every required native desktop
is actually accessible to the implementing agent).

**Agent/model:** A native desktop operator on each OS, supported by the T0 Codex
Lead on GPT-6 Astra, high reasoning. A model cannot substitute for an unavailable
OS or physical/native accelerator evidence. T0 owns all diagnosis/fixes/review.

**Context:** [MA-028 spec](../spec.md), AC10 and its four-step native procedure;
the [execution plan](../../../plans/2026-09-27-ma-028-command-admission.md)
requires an early feasibility observation during 02 and final evidence here.

- [ ] On each native Linux/Windows/macOS desktop, run the integrated revision
  with `make run`; record OS/architecture, revision, Fyne version, input route,
  reproducible steps and actual visible/payload outcomes. Launch alone does
  not establish any scenario below. Verify V10.
- [ ] With loaded images/Grid and a focused naming/editing field, select
  distinctive text and use physical platform Copy and Select All. Confirm text
  payload/selection and unchanged underlying image/Grid, including the route
  that would otherwise match an application menu accelerator. Verify V10.
- [ ] Separately invoke explicit image menu actions in an admitted context and
  under a real dialog. Confirm image intent when admitted, modal refusal when
  blocked, and positive prompt control/confirmation/cancel behavior. Verify V10.
- [ ] Exercise actual menu selection and native availability through modal and
  held region-copy entry/exit. Disabled items cannot bypass live admission;
  legitimate menu selections still work, refusal feedback is preserved and
  native close remains usable. Hold copy reproducibly through an instance-owned
  seam if needed; do not rely on clipboard speed. Verify V10.
- [ ] Reconcile results with ticket 09 and all spec ACs. T0 fixes confirmed
  defects inline and repeats affected automated checks, inspections and native
  scenarios on the changed revision. A newly discovered behavior choice outside
  D4-D7 requires its concrete D1 scenario decision. An unavailable platform stays
  unverified and this ticket stays open.
- [ ] Record the final evidence and remaining limitations. Only after all
  required gates pass may the implementation be proposed as accepted and the
  plan archived. No merge, release, commit or push is implied.

**Verification V10:** `make run` on each native platform followed by the four
scenario groups above and retained observations. Test-driver dispatch, source
traces, unrelated native suites and a blank runbook cannot pass this ticket.
