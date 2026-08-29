# Qodana CI Duplication Reporting — Implementation Plan

> **Superseded 2026-08-29** by `finished_refactorings/2026-08-29-qodana-duplication-close.md`.
> Chunks 1–2 landed as commit `210fee5`. Chunks 3–7 were replaced: their premise, that CI
> under-reports duplication, was disproved by run `33270269940`.

> Requested execution style: subagent-driven development with primary-agent
> review and correction after every chunk. The requested `superpowers` skill is
> not available in this session's skill allowlist, so this plan spells out the
> equivalent evidence, delegation, review, and verification gates directly.

## Selected TODO and scope assumption

The first heading under `todos.md`'s open TODO section says that source-level
false-positive suppressions are complete but still need CI confirmation. That
confirmation already exists:

- Run `33211124429` at `4cc8bb5` reported 107 problems.
- The suppression commit `4a4faf6` triggered run `33213561553`, which reported
  77 problems: exactly 30 fewer.
- The removed findings match the intended 30 non-issues:
  `GoUnusedParameter` -18, `GoBoolExpressions` -3, `GoMaybeNil` -2,
  `GoErrorStringFormat` -2, `GoVarAndConstTypeMayBeOmitted` -4, and
  `GoRedundantConversion` -1. The first five categories disappear;
  `GoRedundantConversion` falls from two findings to one because the unrelated
  `internal/imaging/raw_test.go` finding remains.
- The suspected declaration-level `DuplicatedCode` directive in
  `internal/imaging/dhash_test.go` is unrelated to those counts; that file uses
  `GoVarAndConstTypeMayBeOmitted`, and both of its findings disappeared.

The false-positive item therefore needs only backlog reconciliation. This plan
treats the next genuinely open implementation item as **“Qodana: CI
under-reports duplication against a full scan”**, while closing the confirmed
item in the same final documentation chunk.

If Open Decision D1 resolves to “only close the literal first heading,” stop
after Chunk 1 and the corresponding `todos.md` edit; do not change the workflow.

## Goal

Make Qodana's CI result inspectable and establish, on the same revision, why its
`DuplicatedCode` count differs from a full GoLand inspection. Preserve fast
pull-request feedback while ensuring pushes to `main` and manual diagnostic
runs are unambiguously full scans.

Completion requires more than changing one boolean. The final state must
provide:

1. a downloadable Qodana artifact containing SARIF and supporting logs;
2. explicit event-specific scan scope;
3. a same-SHA inventory of CI duplication results;
4. a comparison against a same-SHA GoLand export, including counting semantics
   such as one result per fragment versus one result per duplicate cluster;
5. either a proven explanation or a narrow reproduction of a real Qodana
   omission;
6. accurate `todos.md` history describing what was learned.

## Current evidence

### Repository configuration

`.github/workflows/qodana_code_quality.yml` currently:

- runs on `pull_request`, pushes to `main`, and `workflow_dispatch`;
- checks out full history;
- selects the mutable `JetBrains/qodana-action@v2026.2` release reference;
- sets `pr-mode: true` for every event;
- keeps Qodana caches enabled;
- sets `upload-result: false`, leaving no GitHub Actions artifact;
- sets `push-fixes: pull-request` even though no quick-fix argument is enabled.

`qodana.yaml` uses `qodana.starter`, selects `jetbrains/qodana-go:2026.2`, and
explicitly includes `DuplicatedCode`.

### `pr-mode` is not the push-scan explanation

The TODO's original `pr-mode: true` suspicion is not supported by the audited
action implementation. Workflow run logs resolved `@v2026.2` to
`b588768b6e7e6da579e518bc584f79de0d243692`; the workflow reference itself is
mutable:

- [`action.yaml`](https://github.com/JetBrains/qodana-action/blob/b588768b6e7e6da579e518bc584f79de0d243692/action.yaml)
  defines `pr-mode` as “Analyze ONLY changed files in a pull request.”
- [`scan/src/utils.ts`](https://github.com/JetBrains/qodana-action/blob/b588768b6e7e6da579e518bc584f79de0d243692/scan/src/utils.ts)
  obtains a comparison SHA only from a pull-request payload or an explicit
  `QODANA_PR_SHA`. Without one, it does not append Qodana's `--commit` option.
- Push-run logs for `4a4faf6`, `720d68e`, and `89b25ae` show
  `qodana scan ... --skip-pull` with no `--commit` option.

Thus push scans were already full scans despite the misleading unconditional
input. Making the input event-specific still improves clarity and protects the
intended contract, but it should not be presented as the proven count fix.

### Artifact observability is definitely missing

The audited action revision's `upload-result` input compresses its `results-dir` and
uploads it as the `qodana-report` artifact. The latest inspected run,
`33266113935`, has zero artifacts because the workflow explicitly disables
that behavior.

Enabling the artifact is the smallest change that lets us inspect rule IDs,
locations, fingerprints, related locations, and report metadata without
requiring Qodana Cloud access.

### Historical counts are not a sufficient comparison

The TODO compares Qodana at `4cc8bb5` with a local IDE scan at `e9cfe7b`.
Those commits are close and the intervening release commit does not materially
change Go code, but a conclusive comparison must use the exact same SHA and
inspection versions.

Counts alone may also compare different units. Qodana may report a duplicate
cluster once with related locations while GoLand's Problems view counts each
fragment. The SARIF and IDE export must be normalized before calling the CI
result an under-report.

## Recommended design

### Workflow change

Change only `.github/workflows/qodana_code_quality.yml` initially:

```yaml
pr-mode: ${{ github.event_name == 'pull_request' }}
upload-result: true
```

This contract means:

- pull requests analyze changes relative to their merge base;
- pushes to `main` run without `--commit` and analyze the whole project;
- manual runs run without `--commit` and analyze the whole project;
- every run retains its report as a downloadable artifact.

Keep `use-caches: true` for the first comparison. Changing scan scope,
artifact behavior, and cache policy at once would make the result harder to
attribute. Add or run a cold-cache diagnostic only if the warm same-SHA result
still omits exact duplicate fragments found by GoLand.

Keep the default artifact name. Artifacts are scoped to a workflow run, so
adding the run ID to `artifact-name` gives no useful disambiguation.

### Report normalization

For each report, extract a stable inventory rather than comparing headline
counts only. Each normalized entry should include, when present:

- SARIF `ruleId`;
- primary repository-relative path and start line;
- related duplicate paths and lines;
- result fingerprint or partial fingerprint;
- message text;
- whether the location is production, test, script, generated, or excluded
  code;
- one raw-result count, one primary-location count, and one expanded-fragment
  count.

Do not add a repository script unless this analysis proves recurring. A
temporary `jq` pipeline in a validated `mktemp -d` directory is enough for a
one-time comparison.

### Diagnosis order

Investigate cheapest explanations first:

1. Different count units: cluster results versus individual fragments.
2. Different project scope: tests, build-tagged files, scripts, ignored files,
   or generated files.
3. Different inspection profile/settings: GoLand's Project Default profile
   versus Qodana's `qodana.starter` plus explicit `DuplicatedCode` inclusion.
4. Stale or mismatched revision/version.
5. Warm-cache effect, tested by repeating the same SHA without caches.
6. A Qodana linter bug or environment-specific duplication-index omission.

Do not change thresholds, suppress findings, or refactor source until the exact
missing fragments and cause are known.

## Alternatives rejected

- **Set `pr-mode: false` for every event.** This would make pull-request scans
  full and slower without addressing the known push discrepancy; push scans
  already have no `--commit` argument.
- **Assume `pr-mode: true` caused the old push count.** The audited action source
  and actual run commands contradict that explanation.
- **Disable caches in the first workflow edit.** This changes a second variable
  before obtaining a report from the existing behavior.
- **Run Qodana locally without its token.** The release linter has already been
  shown to reject that setup, while CI owns the configured secret.
- **Use only Qodana Cloud.** The TODO exists partly because that access is not
  reliably available to the implementation agent. A GitHub artifact is
  portable and tied to the workflow run.
- **Add a baseline or quality gate now.** A baseline would hide the inventory
  before its completeness is trusted; a threshold would gate on an unexplained
  number.
- **Fix all remaining duplicated code.** That is a separate source-refactoring
  program, not a reporting diagnosis.
- **Add a permanent report-parser package or script immediately.** One
  diagnostic comparison does not yet justify repository-owned tooling.

## Planned file map

- `.github/workflows/qodana_code_quality.yml`
  - Make PR mode event-specific.
  - Enable report artifacts.
  - Optionally clean up inert quick-fix configuration only if D4 authorizes it.
- `todos.md`
  - Move the now-confirmed false-positive item to `Done -> Internal`.
  - Close or rewrite the duplication-reporting item only after the same-SHA
    comparison identifies the cause.
- `plans/2026-08-29-qodana-ci-duplication-reporting.md`
  - Move to `finished_refactorings/` only when all acceptance criteria pass.

Expected untouched files:

- `qodana.yaml`, unless exact report comparison proves profile configuration is
  the cause and a minimal explicit setting is required;
- all Go source and test files;
- `ARCHITECTURE.md`, because no package, file ownership, or runtime flow changes;
- translations and golden screenshots;
- dependency manifests.

## Delegation strategy

Chunks run sequentially. Later chunks depend on reviewed output from earlier
ones, and the primary agent must inspect and correct each result before another
agent edits overlapping files.

Each `spawn_agent` call should use `fork_turns: "none"` and include:

- the relevant `AGENTS.md` rules;
- the selected chunk only;
- explicit file ownership and forbidden files;
- the current reviewed diff or run ID when needed;
- a reminder not to commit.

| Chunk | Model | Effort | Reason |
|-------|-------|--------|--------|
| 1 — independent evidence audit | `gpt-5.6-terra` | `high` | Bounded read-only CI/source cross-check; strong analysis without using the most expensive coding model. |
| 2 — workflow observability edit | `gpt-5.6-sol` | `high` | GitHub expression behavior and event-dependent Qodana semantics need strongest coding judgment. |
| 3 — full-run artifact inventory | `gpt-5.6-sol` | `high` | SARIF normalization and cluster/location counting can easily produce a false diagnosis. |
| 4 — same-SHA report comparison | `gpt-5.6-sol` | `xhigh` | Root-cause analysis spans two report formats, scopes, and inspection semantics. |
| 5 — conditional correction | `gpt-5.6-sol` | `high` or `xhigh` | Model effort depends on whether the cause is simple workflow scope or a linter reproduction. |
| 6 — backlog reconciliation | `gpt-5.6-terra` | `medium` | Small documentation update requiring exact evidence and no speculative claims. |
| 7 — independent final audit | `gpt-5.6-terra` | `high` | Fresh read-only review plus full repository verification. |

No chunk needs Opus: the task splits cleanly into workflow, artifact, comparison,
conditional correction, and documentation stages. Opus is also not available in
this session's subagent model allowlist; `gpt-5.6-sol` is the strongest available
model for the difficult stages.

---

## Chunk 1 — Independently verify baseline evidence

**Subagent:** `gpt-5.6-terra`, reasoning `high`

**Mode:** Read-only. No file edits.

Tasks:

- [ ] Read `ARCHITECTURE.md`, `AGENTS.md`, both open Qodana TODOs,
      `qodana.yaml`, and `.github/workflows/qodana_code_quality.yml`.
- [ ] Verify run `33211124429` at `4cc8bb5` reported 107 total findings and 42
      `DuplicatedCode` findings.
- [ ] Verify run `33213561553` at `4a4faf6` reported 77 total findings and that
      the exact 30 intended false positives disappeared.
- [ ] Verify a push-run command line contains no `--commit` argument.
- [ ] Read the exact action revision resolved by the historical runs,
      `b588768b6e7e6da579e518bc584f79de0d243692`, not `main`, and confirm
      how `pr-mode` and `upload-result` work.
- [ ] Confirm run `33266113935` has no GitHub Actions artifacts.
- [ ] Return a concise evidence ledger with run URLs, SHAs, totals, rule counts,
      action-source permalinks, and any contradiction found in this plan.
- [ ] Do not use or request the value of `QODANA_TOKEN`.

### Primary-agent review gate after Chunk 1

- [ ] Re-run the decisive `gh run view` / `gh api` reads independently.
- [ ] Compare the ledger against actual logs rather than trusting copied TODO
      prose.
- [ ] Correct this plan's factual notes if the audit finds drift.
- [ ] If D1 says to stop after literal first-heading closure, edit only
      `todos.md`, review the diff, run documentation checks, and hand off.
- [ ] Otherwise begin Chunk 2 only when the evidence proves artifact absence
      and full push-scan behavior.

---

## Chunk 2 — Make workflow scope explicit and upload reports

**Subagent:** `gpt-5.6-sol`, reasoning `high`

**Files:**

- Modify `.github/workflows/qodana_code_quality.yml` only.
- Do not modify `qodana.yaml`, `todos.md`, Go files, dependencies, or this plan.

Tasks:

- [ ] Replace unconditional `pr-mode: true` with
      `${{ github.event_name == 'pull_request' }}`.
- [ ] Rewrite the nearby comment to state the three event contracts precisely;
      do not claim that the old value narrowed push scans.
- [ ] Change `upload-result: false` to `upload-result: true`.
- [ ] Keep `results-dir`, cache keys, artifact name, Qodana image, linter
      profile, annotations, and PR comments unchanged.
- [ ] Apply D4 exactly: leave `push-fixes` and permissions untouched by default;
      remove/narrow them only if the user explicitly expands scope.
- [ ] Preserve `fetch-depth: 0`, required for PR merge-base analysis.
- [ ] Do not add `TODO`/`FIXME`, a baseline, failure threshold, local secret, or
      another workflow job.
- [ ] Run `git diff --check` and inspect the resulting YAML diff.

### Primary-agent review gate after Chunk 2

- [ ] Inspect the full worktree diff, not only the subagent summary.
- [ ] Confirm exactly one file changed and no secret value entered the diff.
- [ ] Check expression spelling, quoting, indentation, event names, and comments.
- [ ] Compare the inputs with the audited revision's `action.yaml` definitions.
- [ ] Fix any problem personally before continuing.
- [ ] Run:

  ```sh
  git diff --check
  make fmt-check
  ```

- [ ] Record that local validation cannot prove artifact upload; GitHub Actions
      execution is the integration gate.

---

## External gate — Put the workflow revision on GitHub

The assistant must not run `git commit` under repository rules. A Qodana run
also cannot execute the uncommitted local workflow. Before Chunk 3, the user
must commit and push the reviewed workflow change to a branch, or otherwise
make that exact revision available on GitHub.

After the revision exists remotely, triggering `workflow_dispatch` is an
external state change. Do it only with the user's authorization from D3.

Required invariants:

- [ ] The remote workflow SHA equals the locally reviewed SHA.
- [ ] The dispatch targets that branch/ref, not stale `main`.
- [ ] The branch contains the workflow change and the same Go code that the
      local IDE report will inspect.
- [ ] Do not merge, tag, release, or push quick fixes as part of this gate.

---

## Chunk 3 — Run a full scan and inventory the artifact

**Subagent:** `gpt-5.6-sol`, reasoning `high`

**Mode:** External CI execution plus read-only analysis. No repository edits.

Tasks:

- [ ] Trigger `qodana_code_quality.yml` with `workflow_dispatch` on the exact
      reviewed remote ref, if D3 authorizes agent-triggered CI.
- [ ] Wait for completion without polling aggressively.
- [ ] Verify the log's `qodana scan` command has no `--commit` argument.
- [ ] Verify the action input reports `pr-mode: false` and
      `upload-result: true`.
- [ ] Verify exactly one non-expired `qodana-report` artifact exists.
- [ ] Download it into a validated `mktemp -d` directory. Do not extract into
      the repository or a broad path.
- [ ] Inspect archive members before extraction, then extract safely.
- [ ] Locate SARIF, logs, and report metadata. Record Qodana linter/build,
      analyzed SHA, profile, total findings, and per-rule counts.
- [ ] Normalize all `DuplicatedCode` results using the fields listed in Report
      normalization.
- [ ] Count raw results, primary locations, related locations, unique fragments,
      and unique clusters separately.
- [ ] Classify repository paths by source/test/script/build-tag category.
- [ ] Return the run URL, artifact metadata, normalized counts, and retained
      temporary path. Do not copy the report into git.

### Primary-agent review gate after Chunk 3

- [ ] Independently check run SHA, event, action inputs, command line, conclusion,
      and artifact metadata.
- [ ] Re-run the normalizer against several hand-checked SARIF entries.
- [ ] Check that URI decoding and line-number bases are correct.
- [ ] Confirm related locations are not accidentally counted twice.
- [ ] Preserve the artifact until Chunk 4 completes; then remove only the exact
      temporary directory or allow normal temp cleanup.
- [ ] If artifact upload failed, fix only the workflow issue and repeat Chunk 2's
      review plus this run. Do not proceed using headline logs alone.

---

## Chunk 4 — Compare CI and GoLand on the same SHA

**Subagent:** `gpt-5.6-sol`, reasoning `xhigh`

**Mode:** Read-only report analysis. No repository edits.

**Prerequisite:** A GoLand inspection export for the exact CI SHA, supplied as
described by D2. A screenshot or headline count is insufficient; the export
must contain rule IDs and locations.

Tasks:

- [ ] Verify both reports identify the same repository revision, Go language
      level, and 2026.2 inspection family. Record any build-number difference.
- [ ] Normalize the GoLand export to the same path/line/cluster representation
      as the SARIF inventory.
- [ ] Compare:
  - raw results;
  - unique fragment locations;
  - duplicate clusters;
  - production versus test and script paths;
  - files present only in one report;
  - shared fragment locations grouped differently.
- [ ] Test the count-unit hypothesis first. If Qodana's 42 results expand to
      roughly the IDE's 90 fragment hits, document that and stop looking for
      omitted code.
- [ ] If exact fragments are absent, determine whether all omissions share a
      scope property, profile setting, file type, build tag, or directory.
- [ ] Compare `qodana.starter` with the local Project Default inspection profile;
      do not assume that enabling the same inspection ID makes all thresholds
      and scopes identical.
- [ ] Produce a root-cause report with a set difference, not a guess.
- [ ] Recommend exactly one Branch A–D below.

### Primary-agent review gate after Chunk 4

- [ ] Hand-check at least three shared clusters and three report-only fragments
      in source.
- [ ] Verify the comparison did not mix zero-based and one-based lines or count
      both cluster anchors and members as independent clusters.
- [ ] Challenge the proposed cause with one falsification test.
- [ ] Fix analysis scripts in temp and rerun if totals do not reconcile.
- [ ] Select one conditional branch only after exact evidence supports it.

---

## Chunk 5 — Apply the smallest evidence-driven correction

**Subagent:** `gpt-5.6-sol`; `high` for A–C, `xhigh` for D

Only one branch should run.

### Branch A — Counting semantics explain the gap

If Qodana clusters and GoLand fragments represent the same underlying duplicate
locations:

- [ ] Make no further workflow, `qodana.yaml`, or source changes.
- [ ] Record both count units and the reconciliation formula for `todos.md`.
- [ ] Treat CI as complete for its configured scope, but name the count unit
      whenever using it as a future gate.

### Branch B — Profile or scope explains the gap

If both tools intentionally analyze different scopes/settings:

- [ ] Decide whether CI or GoLand is authoritative according to D5.
- [ ] If CI should match GoLand, add the smallest explicit Qodana profile/scope
      configuration and no broad exclusions.
- [ ] If the difference is intended, make no configuration change; document the
      expected scope and how to reproduce each count.
- [ ] Re-run the exact same-SHA full scan after any configuration change and
      repeat Chunk 3 normalization.

### Branch C — Warm cache causes the mismatch

If exact report comparison gives credible cache evidence:

- [ ] Run the same remote SHA once with a cold Qodana cache.
- [ ] Prefer a one-time diagnostic mechanism; do not disable useful caching
      permanently unless repeated same-SHA evidence proves incorrect warm runs.
- [ ] Compare warm and cold normalized SARIF sets.
- [ ] If they differ, retain logs/cache-key evidence and open a narrow upstream
      reproduction before changing permanent CI policy.

### Branch D — Qodana omits equivalent in-scope fragments

If same-SHA, same-scope, same-count-unit reports still differ:

- [ ] Reduce one missing duplicate cluster to the smallest safe reproduction.
- [ ] Confirm Qodana 2026.2 misses it while GoLand 2026.2 reports it.
- [ ] Check whether a current 2026.2-compatible Qodana patch release fixes it;
      do not upgrade dependencies or action versions without separate review.
- [ ] Keep the repository workaround narrow. Prefer documenting the known
      reporting limitation over distorting source or suppressing valid findings.
- [ ] Rewrite the TODO as an upstream-tool limitation with reproduction and
      tracking link if it cannot be fixed locally.

### Primary-agent review gate after Chunk 5

- [ ] Inspect every changed file and reject unrelated cleanup.
- [ ] Reproduce the selected branch's evidence independently.
- [ ] Fix implementation or wording personally where evidence and diff diverge.
- [ ] Re-run Chunk 3 after any workflow or Qodana configuration edit.
- [ ] Do not let a successful workflow conclusion substitute for report-set
      equality; no quality threshold currently makes these findings fail CI.

---

## Chunk 6 — Reconcile backlog and archive the plan

**Subagent:** `gpt-5.6-terra`, reasoning `medium`

**Files:**

- Modify `todos.md`.
- Move this plan from `plans/` to `finished_refactorings/` only after all prior
  gates pass.
- Do not modify source, workflow, configuration, translations, or architecture.

Tasks:

- [ ] Move `Qodana: false positives are flagged in code` from TODO to
      `Done -> Internal` and replace “needs CI confirmation” with exact run and
      count evidence: 107 to 77, all intended 30 absent.
- [ ] For duplication reporting:
  - move it to `Done -> Internal` if the gap is fully explained or fixed;
  - otherwise replace it with a narrower open item containing exact same-SHA
    missing locations, reproduction steps, and the remaining external blocker.
- [ ] Correct the old claim that unconditional `pr-mode: true` narrowed push
      scans. State that the audited action revision adds `--commit` only for PR
      context.
- [ ] Mention artifact availability and exact event-scope policy.
- [ ] Keep release-note prose focused on behavior and evidence; do not paste raw
      logs or this full plan into `todos.md`.
- [ ] Move this plan only after its acceptance checklist is true.

### Primary-agent review gate after Chunk 6

- [ ] Compare every count, SHA, run ID, and conclusion against saved evidence.
- [ ] Ensure no open uncertainty is presented as a completed fix.
- [ ] Confirm unrelated TODOs and Done entries are unchanged.
- [ ] Confirm `ARCHITECTURE.md` remains accurate without edits.
- [ ] Fix wording personally, then run `git diff --check`.

---

## Chunk 7 — Independent final audit and verification

**Subagent:** `gpt-5.6-terra`, reasoning `high`

**Mode:** Read-only. No edits unless the primary agent assigns a narrowly scoped
follow-up after reviewing findings.

Tasks:

- [ ] Review the complete diff from the starting SHA.
- [ ] Check event behavior for PR, push-to-main, and manual dispatch separately.
- [ ] Verify artifacts cannot accidentally contain secrets; inspect member names
      and sample logs for redaction without publishing report contents.
- [ ] Verify the chosen diagnosis from normalized report evidence.
- [ ] Check `todos.md` and archived plan against final state.
- [ ] Run the repository's required handoff checks from the root:

  ```sh
  make fmt-check
  go vet ./...
  go build ./...
  go test -timeout 20m -race ./...
  git diff --check
  ```

- [ ] Report exact pass/fail status. Do not hide unrelated pre-existing failures.

### Primary-agent final review

- [ ] Read the auditor's findings and independently inspect any claimed problem.
- [ ] Fix in-scope defects personally and rerun affected gates.
- [ ] Re-run the complete required verification after the final fix.
- [ ] Check `git status --short` and preserve unrelated user changes.
- [ ] Do not commit.
- [ ] Suggest a commit message, for example:

  ```text
  ci: make Qodana reports inspectable
  ```

## Acceptance checklist

- [ ] The 30 false-positive suppressions are recorded as CI-confirmed using run
      `33213561553`.
- [ ] Pull requests use differential analysis; pushes and manual runs use full
      analysis.
- [ ] Qodana report artifacts are uploaded and downloadable.
- [ ] A full scan log has no `--commit` option.
- [ ] CI and GoLand reports compared on the exact same SHA and compatible 2026.2
      builds.
- [ ] Raw result, cluster, primary-location, and expanded-fragment counts are
      distinguished.
- [ ] Duplication mismatch has a proven cause, or the remaining TODO contains a
      minimal exact reproduction rather than a headline-count guess.
- [ ] No source duplication is refactored merely to make reporting totals match.
- [ ] No secret, baseline, quality threshold, dependency, translation, golden,
      package, or runtime behavior changes unintentionally.
- [ ] `ARCHITECTURE.md` remains accurate.
- [ ] Full required verification passes.
- [ ] Plan is archived only after completion.
- [ ] No commit was created by an agent.

## Open decisions for the user

### D1 — Literal next heading or next real implementation item?

**Recommended:** Close the already-confirmed false-positive heading as
bookkeeping, then execute this plan for duplication reporting. Alternative:
stop after moving only the confirmed heading to Done.

### D2 — Can you provide a same-SHA GoLand report export?

**Recommended:** Export the full 2026.2 inspection result with paths, lines, and
inspection IDs after the workflow revision is fixed, then place it in a stated
workspace or temporary path. Without it, agents can make CI observable and
inventory SARIF, but cannot prove the exact IDE/CI set difference.

### D3 — May agents trigger and monitor the manual Qodana run after you push?

**Recommended:** Yes. You commit and push the reviewed workflow because agents
must not commit; agents then dispatch the exact remote ref, wait, download the
artifact, and analyze it. Alternative: you trigger it and provide the run ID.

### D4 — Remove inert quick-fix permissions/configuration?

**Recommended:** Leave `push-fixes: pull-request` and write permissions out of
scope for this change. They are inert without `--apply-fixes`, but removing them
is adjacent security cleanup and deserves a separate decision. If authorized,
the workflow subagent may remove `push-fixes` and narrow permissions in Chunk 2.

### D5 — Which scan should become authoritative if profiles intentionally differ?

**Recommended:** Make Qodana's explicit repository configuration authoritative
for CI, and document GoLand profile differences. Alternative: configure Qodana
to match the Project Default IDE profile exactly, accepting any broader CI cost
or noise.
