# 05: Run all required CI work concurrently

**What to build:** Reduce avoidable CI waiting by starting all independent
required work together. Contributors see separately named validation, non-UI,
UI-shard, and Windows results, while release operators retain one mandatory
reusable-CI gate that fails when any entry fails.

**Blocked by:** 04: Run the sharded race contract locally.

**Status:** resolved

- [x] One Ubuntu 24.04 validation job retains formatting, trust-root
      validation, Linux GUI dependency setup, shard-manifest validation,
      vetting, normal build, both Windows cross-builds, and Windows vetting.
- [x] One Ubuntu 24.04 Linux race matrix has exactly four entries: non-UI and
      the three UI shards.
- [x] Every race entry independently checks out the code, sets up Go, installs
      Linux GUI dependencies, and calls the shared local/CI test contract.
- [x] The validation job, all four race entries, and the existing Windows test
      job have no dependency edges between them.
- [x] Matrix fail-fast is disabled, superseded-workflow cancellation remains
      enabled, and no job or step is allowed to hide a failure as optional.
- [x] Every Linux race entry retains the explicit locale, race detector, fresh
      execution count, 30-minute package timeout, and intended package/test
      scope.
- [x] Visible job, step, summary, and artifact names identify `non-ui` or the
      exact UI shard and distinguish repeated workflow attempts.
- [x] Every Linux entry uploads its raw event stream on success or failure,
      treats a missing artifact as an error, and uses 14-day retention.
- [x] The native Windows test job remains independent and its test command is
      identical to the base revision.
- [x] The reusable-workflow trigger, read-only permissions, concurrency group,
      and release gate remain intact, with no release-workflow source change.
- [x] Static workflow review and the previously passing full local verification
      show no lost validation step, package, test, or safety flag before the
      user creates the sharded checkpoint commit.

## Comments

- 2026-09-03: Claimed. The existing Deep plan fixes the topology and shared
  Make target contracts. This ticket changes only the reusable CI workflow and
  ticket/plan bookkeeping; release packaging, the Windows test command, shard
  assignment, Make targets, application code, and test scope remain fixed.
- 2026-09-03: Resolved. `ci.yml` now has one independent Ubuntu 24.04
  validation job, a fail-fast-disabled `non-ui`/`ui-1`/`ui-2`/`ui-3` Ubuntu
  race matrix, and the unchanged independent native Windows test job. Each
  race entry uses the prepared-runner Make contract and publishes an
  attempt-qualified job summary and required 14-day raw artifact even after a
  failed test step.
- 2026-09-03: The static YAML contract passed, including exact job topology,
  retained triggers/permissions/concurrency, absence of `needs` and
  `continue-on-error`, exact matrix values, failure policy, names, summary,
  artifact behavior, and Windows command. A parsed before/after comparison
  retained every prior validation/build step and the full Windows job body;
  `.github/workflows/release.yml` has no diff from `main`.
- 2026-09-03: `make fmt-check`, `go vet ./...`,
  `go test ./scripts/testshards -count=1`, all four direct-target dry runs, and
  `git diff --check` pass. Ticket 04's final `make verify` remains the full
  local race evidence because this ticket changes only Actions YAML. Actual
  six-slot execution, reusable-caller failure propagation, and artifact
  collection remain the user-owned sharded CI checkpoint in ticket 06.
