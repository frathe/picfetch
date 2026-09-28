# 05: Enforce policy for checks and staging

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [02: Capture immutable launch decisions](02-captured-launch-decisions.md).

**What to build:** automatic and manual update checks, verifier preparation and
staging obey captured launch permission through both viewer actions and direct
updater calls, without changing ordinary portable update behavior.

## Acceptance criteria

- [ ] Missing/invalid or restricted policy refuses verifier preparation,
  stale-stage access/removal, worker admission, requests, downloads and staging
  before covered I/O. Preconfigured clients and pre-existing stages do not
  grant permission. Verify: V1.
- [ ] Root startup/manual actions and preference toggles consume the same fixed
  decision. Feature close/replacement and stale/direct calls cannot enable
  checks. Explicit refusal completes its established callback/error protocol
  without progress, false success or stranded busy/completion state. Verify: V2.
- [ ] Restoring the last-check day seeds memory without update-specific
  persistence. A successful permitted check persists its day; restricted calls
  cannot invoke that callback. General settings persistence may retain an
  unchanged restored day in the selected namespace. Verify: V1, V2.
- [ ] Ordinary opt-in/daily/version/platform gates, manual bypass, one serialized
  transaction, lazy verifier preparation, stage authentication and reuse retain
  their behavior. Disabling automatic checks cancels work but keeps a completed
  stage; queued obsolete callbacks remain rejected. Verify: V3, V4.

## Verification

Parent scope: check/stage portions of AC8/AC10/AC12, last-check portion of AC9,
and AC11. Apply/relaunch and standalone update-record admission remain in 06;
do not weaken their existing guards while adding the updater policy input.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdaterLaunchPolicy$/^(admission|preconfigured|persistence)$'`.
  Require refusal with missing policy, all restricted combinations, configured
  dependencies and seeded stage fixtures; name this ticket's check/stage cases.
- V2 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^(update_entrypoints|feature_lifetime|update_records)$'`.
  Here update-record coverage includes the last-check restore/persist path;
  06 extends it to marker/failure records.
- V3 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestUpdateCheck_|TestManualUpdateCheck_|TestCurrentUpdateCallback_)'`.
- V4 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdater_'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Use call observations for forbidden reads and sentinels for durable outcomes.
No caller-discipline-only authorization or implicit ordinary updater policy.
