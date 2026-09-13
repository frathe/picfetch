# FML-008 — Complete localization and native qualification

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-007.
Acceptance: AC-08 and final MVP acceptance in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; one full MVP gate.

## Files

- Complete all `translations/*.json`, README, and English/German in-app manuals.
- Add a real search-session test in `scripts/explorereval/search_real_test.go`
  under `explorertrial`, using the production worker and installed assets.
- Extend applicable native guard inventories, exact Qodana exclusions/shards,
  and architecture docs. Record results in this directory's qualification report
  when executed and update `todos.md` only when the MVP is complete.

## Contract and work

Explain the loaded-collection scope, ranking order, single-reference selection,
Back/Exit, Save to Favorites scope, skipped files, preparation cost, and the
20-visit/session-memory limits. Show status for zero results, missing assets,
unsupported platforms, failures, and capacity limits. Keep technical model
scores out of confidence-style UI. Verify long German text, keyboard focus,
reference identification, visible controls, and returning to a prior viewport.

Use current assets with no new downloads when already installed. Confirm that
Store builds keep bundled-runtime/model-only-download behavior and that
unsupported/no-cgo builds fail gracefully. Qualify a real cold search, warm
reference switch, Back, and cancellation on the existing supported desktop
targets. Preserve Windows' truthful isolation status and macOS/Linux checks.
Record exact OS/architecture/asset identity; a compile or skipped test is not
native behavior evidence. Reuse existing notices and verify dependency closure
has not changed.

## Acceptance and verification

Named native tests must execute with installed assets; unavailable targets are
explicitly unverified. Recheck FML-001's quality/performance evidence using the
final worker path. Review changed code with GoLand and close confirmed findings.
Run existing golden coverage; regenerate only affected screenshots with
`make golden` and visually inspect them.

```sh
go test . -run '^TestTranslations_' -count=1
make check-test-shards
make verify
go test -tags=explorertrial ./scripts/explorereval -run '^TestRealSearchSession$' -count=1 -v
```

The last command runs natively on each qualified target with its pinned assets.
Use the repository's native Linux/amd64 environment for the full Docker gate.

Done when AC-01 through AC-08 have recorded evidence, the user-facing workflow
and documentation agree, and remaining limitations are stated precisely.
No release, merge, or publication is authorized by this planning ticket.
