# 08: Complete policy adoption and native guard coverage

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [04: Select storage before constructing consumers](04-storage-at-construction.md),
[06: Enforce policy for recovery and installation](06-recovery-and-installation.md),
[07: Explain update restrictions in Settings](07-settings-explanations.md).

**What to build:** complete the application-wide launch-policy path and provide
native qualification suites that cannot pass with missing coverage. Every
covered effect has one captured decision, independently of live feature lifetime.

## Acceptance criteria

- [ ] The lead's production-caller inventory covers pre-app cleanup,
  construction, runtime actions, preference callbacks, records and shutdown.
  No application-facing effect route bypasses policy or infers update permission
  from live trial features. Remove any migration-only alternate route while
  retaining feature-local trial behavior and low-level mechanisms. Verify: V1, V2.
- [ ] Combined real composition proves both trial types and Store/trial cases,
  missing-policy refusal, lifetime independence, correct roots and Settings
  actions together. All earlier required children remain present and pass.
  Verify: V2.
- [ ] Focused ordinary and Store native runner suites select production startup,
  launch preparation, root/updater and Settings guards with exact required child
  inventory and correct build tags. Windows/Store cannot omit root startup or
  root UI; non-Linux hosts avoid unrelated Linux-only goldens. Verify: V3.
- [ ] Runner fixtures prove missing parents/children, skipped required cases,
  failed processes and mismatched required build selection cannot yield a clean
  capture. Captures identify the revision/platform/build and required outcomes.
  Native prerequisite refusal is preserved rather than skipped. Verify: V3.

## Verification

Parent scope: final convergence portions of AC2/AC10, AC19 and AC21. Real native
execution belongs to 09. Runner fixtures are not native-platform qualification.

- V1: execute the parent's exact **AC21 lead-assessment command**, then record
  the complete caller inventory and dispositions against AC1-AC20. The fixed
  baseline diff is an inspection input, not an automatic correctness verdict.
  The lead owns this assessment; a grep count or helper pass is insufficient.
- V2 (all **new** guard families developed by earlier tickets):
  `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunch(Policy|Preparation)Contract$'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdaterLaunchPolicy$'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/settingswin -run '^TestUpdatesTabLaunchPolicy$'`.
- V3 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./scripts/nativeguards -run '^TestLaunchPolicyNativeSuite$'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
The proposed focused suite names are launch-policy and launch-policy-store.
Equivalent existing suites are acceptable only with the same complete inventory
and an updated command mapping. Do not implement a general permission framework.
