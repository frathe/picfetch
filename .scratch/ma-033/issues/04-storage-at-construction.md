# 04: Select storage before constructing consumers

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [03: Prepare and finalize isolated trials](03-prepared-trial-lifetime.md).

**What to build:** every admitted trial starts its consumers with the correct
isolated storage. Correctness no longer depends on constructing against defaults
and retargeting consumers afterward; ordinary launches keep their existing data.

## Acceptance criteria

- [ ] Application identity, Favorites, Explorer presets, general analysis and
  updater storage agree with the captured launch input from first construction.
  Both trial types and Store/trial combinations select the existing layouts.
  Verify: V1, V2.
- [ ] Analysis-cache and Explorer consumers receive the complete selected roots
  before construction can observe them. Fyne cache resolution uses the correctly
  identified app; an identity-derived isolated cache is not misreported as an
  ordinary-install leak. Verify: V2.
- [ ] All effectful trial consumer startup follows successful reservation;
  failed preparation cannot fall back, create ordinary roots or start a consumer.
  Runtime feature close/replacement cannot retarget launch storage. Verify: V2, V3.
- [ ] Preserve ordinary fallback locations, preferences/session restoration,
  launch overrides and native-launcher identity agreement. Existing Favorite
  ownership and shared model/runtime storage retain their own contracts.
  Verify: V1, V4.

## Verification

Parent scope: real-consumer portions of AC3/AC7, AC16, storage/identity portion of
AC18. Use observable construction inputs and external calls, not only final
fields inspected after all late retargeting has already happened.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPolicyContract$/^(identity|storage)$'`.
- V2 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^(construction|feature_lifetime)$'`.
  Require every listed consumer, both trial types, ordinary fallback and a
  denied-construction case in the selected child inventory.
- V3 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^(validation|prepared_cleanup)$'`.
- V4 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchOptions_'`;
  `go test -tags no_emoji,nodynamic -count=1 -v ./scripts/locationmapqualify -run '^TestManual'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Reuse existing construction/harness interfaces. Storage routing is not a
filesystem sandbox or a promise that later disk writes cannot fail.
