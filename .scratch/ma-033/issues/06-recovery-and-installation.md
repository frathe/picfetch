# 06: Enforce policy for recovery and installation

Status: done
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [05: Enforce policy for checks and staging](05-checks-and-staging.md).

**What to build:** launch-time recovery, update notifications and installation
use the same permission as checks. Restricted launches leave update artifacts
untouched; ordinary users retain safe recovery and the existing quit/relaunch flow.

## Acceptance criteria

- [x] Reading, saving and clearing What's New/apply-failure records requires
  captured permission through all application-facing routes. Restricted and
  missing-policy calls cause no covered I/O, including at startup. Raw record
  serialization is not an alternative public bypass. Verify: V1, V2.
- [x] Ordinary backup cleanup observes the failure record before any reporter
  consumes it; failed restore and unreadable records preserve the backup.
  Restricted launches neither sweep backups nor consume notification records.
  Preserve policy-based pre-app predecessor ordering from 02. Verify: V2, V3.
- [x] Direct and root apply-intent/apply/relaunch calls refuse before stage
  inspection, binary mutation or quit when policy denies them, including with
  preconfigured stages. Refusal leaves no stuck completion or false ready state.
  Verify: V1, V2.
- [x] Actual registered shutdown preserves ordinary apply without relaunch;
  explicit Perform update validates usable staging before requesting quit and
  relaunch intent. Closing/replacing trial features cannot enable installation.
  Evidence ownership from 03 remains independent. Verify: V2, V4.
- [x] Stage re-verification/retention, failure classification/recording,
  transaction serialization and stale callback behavior remain unchanged for
  allowed launches. Verify: V4.

## Verification

Parent scope: apply/relaunch portions of AC8/AC10/AC12/AC14, update-record portion
of AC9 and startup recovery portion of AC13. Tests use actual production hooks,
not only harness teardown or direct calls to a replacement test algorithm.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdaterLaunchPolicy$/^(admission|preconfigured|records|persistence)$'`.
  Complete the refusal inventory with apply intent, apply, relaunch and all
  application-facing record operations, preserving 05's existing cases.
- V2 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^(update_entrypoints|feature_lifetime|update_records|backup_order|shutdown)$'`.
- V3 (new startup guard and existing recovery guards):
  `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^ordering$'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestSweepUpdateBackup_'`.
- V4 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^(TestUpdater_|TestApplyStagedUpdate_)'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestPerformUpdate_|TestApplyStagedUpdate_|TestCurrentUpdateCallback_)'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Keep low-level installation/trust algorithms, source edits and unrelated
feature shutdown policy outside this refactoring.

Completed with updater and root behavioral red/green evidence, backup-order and
shutdown negative mutations, ordinary regressions, Store-tagged policy guards,
focused race/vet and complete GoLand inspection. See the active plan's ticket 06
record; integrated CI/native qualification remains ticket 09.
