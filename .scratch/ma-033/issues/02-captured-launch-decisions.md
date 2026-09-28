# 02: Capture immutable launch decisions

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [01: Make production startup testable](01-production-startup.md).

**What to build:** choose the launch's identity, storage decisions and self-update
permission once, then carry the validated decision through real startup into
viewer composition. Predecessor cleanup must obey that captured permission.

## Acceptance criteria

- [ ] The six distribution/purpose combinations produce the accepted permission
  and complete reason set; both trial flags are invalid. Capture actual compiled
  distribution, never infer it from paths or features. Verify: V1, V5.
- [ ] Explicit construction from zero options means ordinary launch. Missing or
  invalid policy rejects composition before preferences/session or features;
  later input mutation and another viewer cannot alter the captured facts.
  The passive value owns no callbacks, live features or resources. Verify: V1, V2.
- [ ] Preserve ordinary and both trial identities, the native Explorer identity
  agreement and current path-selection rules, including ordinary fallback
  semantics without introducing new canonicalization. Verify: V1.
- [ ] Production selects application identity and predecessor cleanup from the
  same validated decision. Restricted launches leave predecessor artifacts
  untouched; ordinary waiting remains before app creation. Verify: V3.
- [ ] Real viewer construction and shared test composition receive explicit
  policy; ordinary saved state and one-shot launch overrides remain compatible.
  Existing downstream guards remain until their owning tickets migrate them.
  Verify: V2, V4.

## Verification

Parent scope: AC1; capture/composition portions of AC2, AC3 and AC7; predecessor
portions of AC5/AC13; AC16 and AC17. Early evidence reservation belongs to 03,
construction-time consumer routing to 04, and direct updater admission to 05/06.

- V1 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPolicyContract$/^(matrix|validity|identity|storage)$'`.
- V2 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^(construction|feature_lifetime)$'`.
  Require children for absent policy before storage access, explicit ordinary
  construction, immutable capture and independent viewer instances. Later
  tickets extend these parents; only this ticket's inventoried children count here.
- V3 (**new**): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^(validation|ordering)$'`.
- V4 (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchOptions_'`.
- V5 (existing distribution guards plus **new** production capture):
  `go test -tags no_emoji,nodynamic -count=1 -v ./internal/distribution -run '^TestStoreManaged_'`;
  `go test -tags no_emoji,nodynamic,microsoftstore -count=1 -v ./internal/distribution -run '^TestStoreManaged_'`;
  `go test -tags no_emoji,nodynamic,microsoftstore -count=1 -v . -run '^TestLaunchStartupContract$'`.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
Keep necessary existing prerequisite probes separate from passive policy.
Never add an implicit ordinary fallback to ease call-site migration. Tagged
tests on this host do not establish native Windows Store qualification.
