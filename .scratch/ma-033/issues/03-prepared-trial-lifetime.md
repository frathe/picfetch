# 03: Prepare and finalize isolated trials

Status: done
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [02: Capture immutable launch decisions](02-captured-launch-decisions.md).

**What to build:** admit a trial before app storage opens, then retain one owner
of its evidence resources through startup, the UI run and finalization. Failed
attempts retain evidence, release acquired resources and require a fresh path.

## Acceptance criteria

- [x] Reject invalid, used and competing evidence paths before app storage or
  feature construction. Exactly one competing reservation succeeds; prior
  evidence is unchanged and no ordinary fallback is selected. Verify: V1, V2.
- [x] Explorer's verified OS network-denial prerequisite still runs, including
  Windows refusal; Location Map does not gain that prerequisite. Preserve
  help/private-worker/native-install ordering. Verify: V1, V2.
- [x] One preparation owner reserves resources before app creation and lends
  recorders to the UI. Partial acquisition and app/run errors release every
  acquired resource exactly once, retain partial evidence and preserve original
  plus finalization errors. Cleanup finishes before process exit. Verify: V1, V2.
- [x] Actual production shutdown stops/joins trial-evidence producers before the
  owner closes evidence or stops/joins its recorder. Verify flush/error and
  repeated-cleanup behavior using observable completion, not sleeps or work
  counters; the harness's stronger drain is not the proof. Verify: V1, V3.
- [x] Existing evidence formats, incomplete-session meaning, launcher behavior
  and signal-driven orderly stopping remain compatible. Reservation is not a
  ready/success marker. Unrelated shutdown joins are not broadened. Verify: V3, V4.

## Verification

Parent scope: AC4, AC6, early-admission portion of AC5, evidence portion of AC14,
and portable/fixture portions of AC18. Actual native execution remains in 09.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPreparationContract$/^(reservation|resources|evidence)$'`.
- V2 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^(early_exit|ordering|validation|prepared_cleanup)$'`.
- V3 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^shutdown$'`.
  Require named evidence children for both trial types, held producer work and
  production-hook/post-run completion; updater-apply coverage is added in 06.
- V4 (existing fixtures): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/explorertrial ./internal/locationtrial`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./scripts/locationmapqualify -run '^TestManual'`.
  Explorer's recorder package currently has no tests; a build-only result does
  not prove resource behavior. V1/V3 must supply the missing observations.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Do not recursively remove failed evidence, change its schema, turn policy into
a resource handle or add protection against external directory replacement.

Completed on 2026-09-28. V1-V4 ran uncached; production-hook held-producer
guards also passed under race, and root/launch/UI contracts passed with
`microsoft_store`. Lead review and GoLand inspections are recorded in the
[archived record](../../../finished_refactorings/2026-09-28-ma-033-launch-policy.md). This host's
unconfined Explorer prerequisite correctly refused a timeout rather than
claiming OS denial; actual isolated/native qualification remains ticket 09.
