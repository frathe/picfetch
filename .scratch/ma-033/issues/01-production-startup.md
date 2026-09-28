# 01: Make production startup testable

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** None (can start immediately).

**What to build:** preserve the launch users have today while making the actual
entry point's order and error exits observable. This prefactoring must let later
tickets prove early refusal without substituting a test-only startup algorithm.

## Acceptance criteria

- [ ] Help and malformed flags exit without desktop startup; private worker
  dispatch stays ahead of desktop preparation. Verify: V1, V2.
- [ ] Native Open With installation stays before driver/app initialization;
  ordinary predecessor cleanup/wait stays before app creation and preferences.
  Existing restricted-launch artifact protection remains intact. Verify: V1, V2.
- [ ] Production and tests use the same high-level orchestration, with per-call
  external operations that expose ordered observations and error injection;
  no mutable package-level test switches are introduced. Verify: V1.
- [ ] Ordinary startup, saved preferences/session restoration and launch overrides
  keep their behavior. Resource ownership and the timing of trial reservation
  are not silently changed in this prefactor. Verify: V1, V3.

## Verification

Parent scope: AC5's existing sequence and AC16 compatibility. Policy-dependent
ordering and early reservation are completed by 02 and 03, not claimed here.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^(early_exit|ordering)$'`.
  Require named children for private dispatch, native install, predecessor order,
  the ordinary run path and propagation of injected app/run failures.
- V2 (existing): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^(TestLaunchArgs_|TestTrialLaunchPreservesPredecessorArtifacts$)'`.
- V3 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchOptions_'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Keep the entry point thin. Demonstrate preservation against production-facing
observations, not source-text matching. This ticket grants no new permissions.
