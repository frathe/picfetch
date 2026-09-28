# 07: Explain update restrictions in Settings

Status: done
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [02: Capture immutable launch decisions](02-captured-launch-decisions.md).

**What to build:** the Updates tab accurately describes the launch. It always
shows the installed version; restricted sessions show the applicable explanations
instead of update controls, including both reasons for Store trials.

## Acceptance criteria

- [x] Root supplies captured permission/reasons to Settings. The mounted Updates
  tab shows installed version/build, ordinary portable controls, and exactly
  the applicable Store/trial explanation set for restricted cases. Verify: V1, V2.
- [x] Prohibited check/apply controls are absent from the mounted tree and
  restricted Settings does not send update actions to its host. Presentation
  does not reconstruct permissions from feature lifetime or replace root/updater
  effect admission. Verify: V1.
- [x] Ordinary controls retain their actions, and closing Settings rejects stale
  callbacks. Existing backend refusal stays intact while 05/06 migrate it;
  integrated stale/direct backend proof remains part of 08. Verify: V1, V2.
- [x] Preserve existing Store copy and provide the trial explanation through
  the established localization mechanism in every catalogue, with identity-map
  English, parity and font-safe strings. Verify: V3.

## Verification

Parent scope: AC15. This ticket depends only on the supplied policy contract;
it does not require completion of updater internals to render accurate content.
The lead owns user-visible wording and translation changes.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/settingswin -run '^TestUpdatesTabLaunchPolicy$'`.
  Require all six combinations, mounted version/reasons, absent denied controls
  and host-call observations; an unattached widget's Visible flag is insufficient.
- V2 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/settingswin -run '^(TestUpdatesTab_.*|TestUpdateCallbacksAfterSettingsCloseAreIgnored)$'`.
- V3 (existing): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestTranslations_'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestTranslationsHaveNoUnicodeArrows$'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Do not remove the Updates tab, prioritize one reason over another or add a
separate Settings permission rule.

Completed: seven mounted-tree cases (six combinations and absent policy) went
red/green, full Settings regressions and localization checks pass. Existing
catalogued trial wording was reused; no translation additions were needed.
All five changed code files have complete clean GoLand inspection evidence.
See the active plan's ticket 07 record.
