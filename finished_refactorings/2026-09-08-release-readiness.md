# Release readiness since v1.0.2

Route: Deep release review, with focused fixes. Baseline
`73a5f6381ab8e643de47a6c2038ed1d929f11670`; initial reviewed head
`d306654e49cec03432070d6219cc920cac9c2b72`. The user requests independent
Standards and Spec reviewers and bounded fix subagents; this explicitly
overrides the process guide's review/delegation limits. One integrator owns
final review, shared manifests and verification. No merge,
release, tag, Store mutation, secret access or reviewer comments.

## Acceptance and ownership

1. Review all release executable changes against standards and discovered
   archived specs; separate the axes and record coverage and remaining gaps.
   Owners: independent read-only Standards and Spec agents; integrator assesses
   findings. Verify: fixed diff inventory and cited finding/coverage records.
2. Substantiate CodeQL archive traversal alert, reject unsafe staging and
   preserve exact artifact admission. Owner: bounded archive agent, only
   `scripts/storepublish/github.go` and existing `main_test.go`.
   Verify: negative fixtures then `go test -race ./scripts/storepublish`.
3. Reconcile inherited case-alias regressions and any launch finding. Owner:
   integrator; image transaction and current-view identity paths. Verify:
   focused native tests on case-insensitive storage, plus portable regressions.
4. Triage exact-head Qodana SARIF; fix real defects and use inspection/file
   exclusions only for justified non-issues. Owner: integrator; coordinate
   Store test edits after archive handback. Verify:
   `make check-qodana-test-exclusions`, focused package tests, fresh CI Qodana.
5. Check the final working tree locally and inspect exact remote-head results
   separately. Commit/push requires an explicit user override of AGENTS.md
   line 10. Owner: integrator. Verify: `make verify` once at final integration
   (record any resource failure separately) and existing GitHub review results.
   Do not infer newer-tree validity from older CI or package smoke evidence.

Initial evidence and downloaded reports are in `/tmp/picfetch-release-review`.
Inherited uncommitted changes in `internal/imaging/mutations_test.go` and
`internal/ui/exportwork_test.go` belong to the withdrawn task; preserved for
completion, with original patch captured before integration.

## Task graph and budget

Standards review, Spec review and archive fix run independently while the
integrator gathers Qodana/CI evidence and reconciles inherited regressions.
All converge on final verification. Budget: three initial agents, one review
round per axis, one final full gate. Additional bounded fixes only when a new
independent finding warrants them. Actuals and evidence will be recorded at
handoff.

## Integration record

Both independent axes and all focused fixes are complete. Four unique P2
runtime defects were closed (case aliases, Save/Trash ordering, queued
picture-frame advances and cancelled startup mode), plus archive staging
hardening. The integrator verified the affected native race packages and UI
regressions, formatting and exclusions. The full `make verify` gate passed:
format/TUF/Qodana guards, vet, build and all four concurrent Linux/amd64 race
partitions. UI partition times: ui-1 301.341s, ui-2 276.064s, ui-3 269.931s.
Automatic approval review rejected the commit because AGENTS.md line 10
forbids `git commit`. The coordinator confirmed its commit/push instruction
was inferred, not an explicit user override. No commit/push occurred and no
retry/workaround is authorized. Results and the unchanged remote head will
are recorded in the task handoff. HEAD and the PR remain `d306654`.

Actual budget: three agents, reused for four bounded fixes; one independent
review per axis and one integrator pass over each fix. One final full gate.
The wider three-agent concurrency and delegated fixes are explicitly requested
by the user; source ownership remained disjoint. Coverage and Qodana rationale
live in `finished_refactorings/2026-09-08-release-readiness/assessment.md`.
