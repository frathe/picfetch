# Qodana Duplication Reporting — Close-Out Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development`
> to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> The primary agent reviews and corrects after every task before dispatching the next.

**Goal:** Close both open Qodana headings in `todos.md` by proving what CI actually
reports, silencing the 33 test-only duplication clusters at the configuration level,
and fixing the one genuine finding that remains.

**Architecture:** No production Go code changes. One test fix, one `qodana.yaml`
configuration change, and a documentation reconciliation, each backed by evidence
taken from the uploaded CI artifact of run `33270269940` and from a same-SHA GoLand
inspection driven through the `goland` MCP server.

**Tech Stack:** Go 1.x + Fyne (unchanged), GitHub Actions, `JetBrains/qodana-action@v2026.2`,
linter `jetbrains/qodana-go:2026.2`, `gh` CLI, `jq`, GoLand 2026.2 via MCP.

**Spec:** This document is self-contained. Its predecessor is
`finished_refactorings/2026-08-29-qodana-ci-duplication-reporting.md`, whose Chunks 1
and 2 landed as commit `210fee5`. That document's premise — "CI under-reports
duplication" — is disproved below; this plan supersedes its Chunks 3–7.

---

## Global Constraints

- Repository root: `/Users/ronin/Projects/picfetch`. Branch: `main`, currently in sync
  with `origin/main` at `210fee5`.
- Verification before any handoff, from the repository root, exactly as CI runs it:
  `make fmt-check`, `go vet ./...`, `go build ./...`, `go test -timeout 20m -race ./...`.
  `make verify` runs the same set.
- `goimports -local github.com/frathe/picfetch` governs import grouping. Any new import
  must survive `make fmt-check`.
- **Subagents do not create git commits.** Each task ends with a staged, reviewed working
  tree and a proposed commit message; the user commits and pushes.
- No production Go source file is modified by this plan. If a task appears to require
  one, stop and escalate to the primary agent.
- Do not refactor any source or test file merely to make a duplication count fall.
- Reference artifact, already downloaded, do not re-download unless it is missing:
  `$SCRATCH/qodana/x/`, where
  `SCRATCH=/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad`.
  Re-fetch command if absent:
  `gh run download 33270269940 -n qodana-report -D "$SCRATCH/qodana" && unzip -o -q "$SCRATCH/qodana/qodana-report.zip" -d "$SCRATCH/qodana/x"`

---

## Established evidence (do not re-derive; verify in Task 1)

Run `33270269940`, event `push`, head SHA `210fee5`, conclusion `success`, artifact
`qodana-report` (1770477 bytes). This is the first run under the revised workflow, so it
is a full scan with `upload-result: true`.

**Per-rule counts at `210fee5`:**

| Rule | `qodana_inspections_summary.csv` | `qodana.sarif.json` results | SARIF locations |
|---|---|---|---|
| `DuplicatedCode` | 75 | 33 | 71 |
| `GoVarAndConstTypeMayBeOmitted` | 4 | 0 | 0 |
| `GoBoolExpressions` | 3 | 0 | 0 |
| `GoMaybeNil` | 2 | 0 | 0 |
| `GoErrorStringFormat` | 2 | 0 | 0 |
| `GoRedundantConversion` | 1 | 0 | 0 |
| `GoTypeAssertionOnErrors` | 1 | 1 | 1 |
| **Total** | **88** | **34** | **72** |

**Three conclusions follow, and the plan exists to harden and record them:**

1. **The CSV is a pre-suppression, per-fragment tally; the SARIF is the post-suppression
   result set.** The five zero-SARIF rules sum to 12, which is exactly the count of
   findings covered by the 8 `//goland:noinspection` comment sites listed in `todos.md`
   (`GoMaybeNil` x2, `GoBoolExpressions` x3, `GoErrorStringFormat` x2,
   `GoRedundantConversion` x1, `GoVarAndConstTypeMayBeOmitted` x4). All 12 findings are
   therefore confirmed suppressed in CI, end to end. The other 18 suppressions were `_`
   parameter renames and never produced a finding to count.
2. **`DuplicatedCode` needs four numbers, not two.** 33 SARIF results (clusters), 71
   location slots across them, **63 distinct fragments** (the slots are not disjoint —
   8 fragments each belong to 2 clusters), and 75 CSV rows. `relatedLocations` is empty
   throughout, which rules out hidden cluster members. So Qodana emits **one SARIF result
   per duplicate cluster**, and the CSV-to-SARIF gap for this rule is 75 − 63 = 12
   fragments, whose cause is **not yet established** — Task 2's fragment-set diff is what
   settles it. An earlier draft of this plan asserted 75 − 4 = 71 from the four suppressed
   orientation copies; that subtraction compared a fragment count against a slot count and
   is withdrawn. The historical "CI 42 vs IDE 90" gap in `todos.md` likewise compared
   cluster results against fragment problems, on top of a profile difference.
3. **All 33 remaining clusters are in `_test.go` files. Production duplication is zero.**
   Fragment counts by file: `internal/ui/exifwin/exifwin_test.go` 8,
   `internal/imaging/raw_test.go` 7, `internal/ui/slideshow_test.go` 5,
   `internal/ui/grid_test.go` 4, `internal/imaging/orientation_test.go` 4,
   `internal/imaging/gif_test.go` 4, then 16 files with 2 and 6 files with 1.

**A fourth fact the predecessor plan did not have:** `qodana.yaml` sets
`profile: name: qodana.starter` and then re-enables `DuplicatedCode` through
`include:`. The local GoLand scan that produced 155 problems used the IDE's Project
Default profile. CI and IDE were never running the same inspection set, so any
head-to-head total was meaningless before this plan's Task 2 normalizes it.

**The one genuine open finding:** `scripts/wingettag/tag_test.go:52`,
`GoTypeAssertionOnErrors`, severity `note` — "Type assertion on errors fails on wrapped
errors."

---

## File map

| Path | Change | Task |
|---|---|---|
| `plans/2026-08-29-qodana-evidence.md` | Create — frozen inventory and reproduction commands | 1 |
| `plans/2026-08-29-qodana-evidence.md` | Modify — append the same-SHA GoLand comparison | 2 |
| `scripts/wingettag/tag_test.go` | Modify — `errors.As` instead of a type assertion | 3 |
| `qodana.yaml` | Modify — exclude `DuplicatedCode` from test files | 4 |
| `todos.md` | Modify — move both Qodana headings to Done with corrected prose | 6 |
| `AGENTS.md` | Modify — one line recording the Qodana counting semantics | 6 |
| `plans/2026-08-29-qodana-duplication-close.md` | Move to `finished_refactorings/` | 6 |
| `plans/2026-08-29-qodana-serialisation-bug-report.md` | Create — the upstream report text | 8 |

`finished_refactorings/2026-08-29-qodana-ci-duplication-reporting.md` stays where it is;
Task 6 adds a superseded-by pointer to its head rather than editing its body.

---

## Delegation strategy

| Task | Agent type | Model | Why |
|---|---|---|---|
| 1 | `general-purpose` | `sonnet` | Mechanical `jq`/`awk` over a downloaded artifact plus tabulation. Deterministic, high volume, no design judgment. |
| 2 | `general-purpose` | `opus` | Must drive the `goland` MCP server, reconcile two different counting units across two different inspection profiles, and decide what "same scan" even means. Genuine judgment; cannot be split further without losing the comparison. |
| 3 | `go-expert` | `haiku` | Two-line idiomatic Go fix with the exact replacement code given below, plus one focused test run. |
| 4 | `general-purpose` | `sonnet` | One YAML edit whose syntax is uncertain and must be checked against vendor documentation, with a stated fallback. Needs care, not depth. |
| 5 | — | — | External gate: the user commits and pushes; agents then dispatch and read the run. |
| 6 | `general-purpose` | `sonnet` | Careful prose reconciliation across three documents against evidence already gathered. |
| 7 | `general-purpose` | `opus` | Independent adversarial audit of everything above, including re-running the full verification suite. Must be willing to reject earlier tasks. |
| 8 | `general-purpose` | `sonnet` | Assemble an upstream bug report from evidence already gathered and verified. Writing, not investigation. |

The primary agent reviews the diff and the claimed evidence after every task, fixes what
is wrong itself, and only then dispatches the next.

---

## Task 1: Freeze the CI evidence

**Files:**
- Create: `plans/2026-08-29-qodana-evidence.md`
- Read only: `$SCRATCH/qodana/x/qodana.sarif.json`,
  `$SCRATCH/qodana/x/log/qodana_inspections_summary.csv`,
  `$SCRATCH/qodana/x/log/qodana-config.json`

**Interfaces:**
- Consumes: nothing.
- Produces: `plans/2026-08-29-qodana-evidence.md`, containing a section
  `## CI inventory at 210fee5` with the per-rule table, a section
  `## Duplication clusters` with all 33 clusters, and a section
  `## Reproduction` with every command used. Tasks 2, 6 and 7 read this file.

- [ ] **Step 1: Confirm the artifact is present, or fetch it**

```bash
SCRATCH=/private/tmp/claude-502/-Users-ronin-Projects-picfetch/5155ce10-0033-47c4-a2cd-52cd5a3bcf6f/scratchpad
test -f "$SCRATCH/qodana/x/qodana.sarif.json" || {
  gh run download 33270269940 -n qodana-report -D "$SCRATCH/qodana"
  unzip -o -q "$SCRATCH/qodana/qodana-report.zip" -d "$SCRATCH/qodana/x"
}
ls "$SCRATCH/qodana/x"
```

Expected: `log  open-in-ide.json  projectStructure  qodana-short.sarif.json  qodana.sarif.json  report`

- [ ] **Step 2: Confirm the run's identity**

```bash
gh run view 33270269940 --json headSha,conclusion,event
```

Expected: `headSha` is `210fee54929de03fc0316025834874f965df2cd0`, `conclusion` is
`success`, `event` is `push`. If `headSha` differs, stop and escalate — every number in
this plan is SHA-bound.

- [ ] **Step 3: Reproduce the per-rule SARIF counts**

```bash
SARIF="$SCRATCH/qodana/x/qodana.sarif.json"
jq '[.runs[].results[]] | length' "$SARIF"
jq -r '[.runs[].results[].ruleId] | group_by(.) | map({r:.[0],n:length}) | sort_by(-.n) | .[] | "\(.n)\t\(.r)"' "$SARIF"
```

Expected: total `34`; then `33 DuplicatedCode` and `1 GoTypeAssertionOnErrors`.
Any other output means the plan's premise is wrong — stop and escalate.

- [ ] **Step 4: Reproduce the per-rule CSV counts**

```bash
awk -F';' 'NR>1 && $8+0>0 {print $8"\t"$1}' "$SCRATCH/qodana/x/log/qodana_inspections_summary.csv"
```

Expected exactly seven rows: `2 GoMaybeNil`, `3 GoBoolExpressions`, `75 DuplicatedCode`,
`1 GoRedundantConversion`, `1 GoTypeAssertionOnErrors`, `2 GoErrorStringFormat`,
`4 GoVarAndConstTypeMayBeOmitted`.

- [ ] **Step 5: Reproduce the cluster/fragment arithmetic**

```bash
jq -r '[.runs[].results[] | select(.ruleId=="DuplicatedCode")] | {results:length, locations:(map(.locations|length)|add), related:(map(.relatedLocations//[]|length)|add)}' "$SARIF"
```

Expected: `{"results":33,"locations":71,"related":0}`. Record in the evidence file that
`related` being 0 is what rules out "cluster members hidden in `relatedLocations`" as an
alternative explanation.

- [ ] **Step 6: Emit the full 33-cluster inventory**

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | "| " + ([.locations[] | .physicalLocation.artifactLocation.uri + ":" + (.physicalLocation.region.startLine|tostring) + "-" + (.physicalLocation.region.endLine|tostring)] | join(" | ")) + " |"' "$SARIF"
```

Paste the output into the evidence file under `## Duplication clusters`, one row per
cluster, and add a leading column with the cluster's index 1..33. Above the table, state
the count of clusters whose fragments all live in one file versus those spanning files.

- [ ] **Step 7: Confirm every fragment is in a test file**

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" | grep -cv '_test\.go$'
```

Expected: `0`. If it is not 0, list the offending files in the evidence file under a
heading `## Production duplication` and escalate — Task 4's premise fails.

- [ ] **Step 8: Record the effective profile**

```bash
jq -r '.. | objects | select(has("content")) | .content' "$SCRATCH/qodana/x/log/qodana-config.json" | grep -A2 '^profile:'
```

Expected: `profile:` then `  name: qodana.starter`. Record this verbatim in the evidence
file, and note that `qodana.yaml` re-enables `DuplicatedCode` via `include:`.

- [ ] **Step 9: Write the evidence file**

Assemble `plans/2026-08-29-qodana-evidence.md` with these sections, in order:
`## Run identity`, `## CI inventory at 210fee5` (the per-rule table),
`## Suppression accounting` (the 12-of-12 and the 75−4=71 arithmetic),
`## Duplication clusters` (the 33-row table), `## Effective profile`,
`## Reproduction` (every command above, copy-pasteable). Prose is normal English, not
compressed. State observed numbers only; draw no conclusion the commands did not show.

- [ ] **Step 10: Report, do not commit**

Report to the primary agent: the file path, whether every expectation above matched, and
any mismatch verbatim. Do not run `git commit`.

### Primary-agent review gate after Task 1

Re-run Steps 3, 4, 5 and 7 yourself. Confirm the evidence file's numbers match your own
output character for character, that the 33-row table has 33 rows, and that no sentence
in the file asserts something the commands did not produce. Fix any drift before Task 2.

---

## Task 2: Same-SHA GoLand comparison through the `goland` MCP server

**Files:**
- Modify: `plans/2026-08-29-qodana-evidence.md` — append `## GoLand comparison at 210fee5`

**Interfaces:**
- Consumes: `plans/2026-08-29-qodana-evidence.md` from Task 1, specifically the CI
  numbers 33 clusters / 71 location slots / 63 distinct fragments / 75 CSV rows, the fact
  that 8 fragments belong to 2 clusters each, the open 75 − 63 = 12 gap, and the
  `qodana.starter` profile.
- Produces: an appended section stating the IDE's `DuplicatedCode` count at the same SHA,
  the unit it counts in, and a one-paragraph verdict naming the cause of any residual
  delta. Tasks 6 and 7 quote this verdict.

**Context the executor needs:** `todos.md` records a local full IDE scan at `e9cfe7b`
reporting 155 problems, 90 of them `DuplicatedCode`. That scan predates commit `795aa80`
("deduplicate shared image test fixtures") and several later cleanups, and it used the
IDE Project Default profile rather than `qodana.starter`. Both differences must be
controlled before the 90 is compared to anything.

- [ ] **Step 1: Confirm the working tree is exactly the scanned SHA**

```bash
git rev-parse HEAD && git status --porcelain
```

Expected: `210fee54929de03fc0316025834874f965df2cd0` and empty status. If the tree is
dirty, stop and escalate — a dirty tree invalidates the comparison.

- [ ] **Step 2: Confirm the MCP server sees this project**

Call `mcp__goland__get_project_modules` and `mcp__goland__get_repositories`.
Expected: the picfetch module and this repository. If the server is not connected, stop
and report that Task 2 is blocked on GoLand being open on `/Users/ronin/Projects/picfetch`.

- [ ] **Step 3: Enumerate the Go files to inspect**

```bash
cd /Users/ronin/Projects/picfetch && git ls-files '*.go' | wc -l
git ls-files '*.go' | grep -c '_test\.go$'
```

Record both numbers. The CI summary CSV reports `DuplicatedCode` was performed 413 times
and the Go inspections 306 times; note in the evidence file how those relate to the file
counts, and do not claim they must be equal.

- [ ] **Step 4: Run the IDE inspection over the tree**

Use `mcp__goland__lint_files` across the Go files from Step 3, in batches if the tool
limits input size. If `lint_files` does not surface `DuplicatedCode`, fall back to
`mcp__goland__run_inspection_kts` with an inspection scoped to `DuplicatedCode`; use
`mcp__goland__generate_inspection_kts_api` and
`mcp__goland__generate_inspection_kts_examples` first to get the current API shape.
Capture the raw results to `$SCRATCH/goland-dup-210fee5.txt`.

- [ ] **Step 5: Count in both units**

From the captured results, produce two numbers: the count of reported
`DuplicatedCode` problems (the IDE's fragment-style unit) and, if the IDE groups them,
the count of distinct duplicate groups. State which unit each number is in. Do not
average, estimate, or reconcile by adjusting a number.

- [ ] **Step 6: Diff the fragment sets, not the totals**

Build the set of `file:startLine` fragments from the IDE run and from the CI SARIF
(`jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | .locations[] | .physicalLocation.artifactLocation.uri + ":" + (.physicalLocation.region.startLine|tostring)' | sort -u`),
sort both, and run `comm -3`. The CI side has exactly 63 entries — `uri:startLine` and
`uri:startLine:charOffset` yield the same 63, so no fragment is lost to the coarser key.
List up to 20 entries from each side in the evidence file. A set difference is the
finding; a total difference is not.

- [ ] **Step 7: Write the verdict**

Append `## GoLand comparison at 210fee5` stating, in normal English: both counts with
their units, the fragment-set difference, and which of these accounts for it —
counting unit, profile (`qodana.starter` vs Project Default), the commits between
`e9cfe7b` and `210fee5`, or a genuine Qodana omission. If and only if a genuine omission
survives, add a subsection `## Minimal reproduction` naming one specific fragment pair
the IDE reports and CI does not, with file and line numbers.

- [ ] **Step 8: Report, do not commit**

Report the two counts, the set difference size, and the verdict sentence verbatim.

### Primary-agent review gate after Task 2

Check that the verdict names a cause supported by the set difference rather than by
totals; that the units are labelled on every number; and that any claim of a Qodana
omission carries a concrete fragment pair. If the executor reconciled two numbers by
assertion rather than by a set operation, reject and re-dispatch with that correction.

---

## Task 3: Fix the wrapped-error type assertion

**Files:**
- Modify: `scripts/wingettag/tag_test.go` — imports, and the assertion at line 52
- Test: `scripts/wingettag/tag_test.go` itself (`TestPatternMatchesBashERE`)

**Interfaces:**
- Consumes: nothing.
- Produces: a tree where `scripts/wingettag/tag_test.go` no longer type-asserts on an
  error. Task 7 re-checks that the finding is gone.

**Context the executor needs:** the finding is `GoTypeAssertionOnErrors` at
`scripts/wingettag/tag_test.go:52`. `TestPatternMatchesBashERE` shells out to `bash` and
wants to distinguish "the regex did not match" (a non-zero exit, reported by
`exec.Cmd.Run` as `*exec.ExitError`) from "bash could not be run at all" (any other
error). A bare type assertion misses a wrapped `*exec.ExitError`; `errors.As` unwraps.

- [ ] **Step 1: Read the current code**

```bash
sed -n '50,58p' /Users/ronin/Projects/picfetch/scripts/wingettag/tag_test.go
```

Expected to contain `if _, ok := err.(*exec.ExitError); !ok {`.

- [ ] **Step 2: Verify the test passes before the change**

Run: `cd /Users/ronin/Projects/picfetch && go test ./scripts/wingettag/...`
Expected: `ok`. This is the behaviour the change must preserve.

- [ ] **Step 3: Apply the fix**

Replace this block:

```go
		got := err == nil
		if !got {
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("bash %q: %v", tc.tag, err)
			}
		}
```

with:

```go
		got := err == nil
		if !got {
			if _, ok := errors.AsType[*exec.ExitError](err); !ok {
				t.Fatalf("bash %q: %v", tc.tag, err)
			}
		}
```

Use `errors.AsType`, not `errors.As`. The module targets `go 1.26.7`, where
`errors.AsType[E error](err error) (E, bool)` exists, and this repository already uses it
— `internal/ui/openfiles.go:102` calls `errors.AsType[*exec.ExitError](err)` for exactly
this type, and `internal/imaging/loader_test.go` uses the `if _, ok := errors.AsType[...](err); !ok`
shape three times. It also preserves the original line's shape, keeping the diff to one
line.

and add `"errors"` to the import block, which becomes:

```go
import (
	"errors"
	"os/exec"
	"testing"
)
```

- [ ] **Step 4: Verify the test still passes**

Run: `cd /Users/ronin/Projects/picfetch && go test ./scripts/wingettag/... -run TestPatternMatchesBashERE -v`
Expected: `PASS`.

- [ ] **Step 5: Prove the new branch is reachable**

Temporarily change `exec.Command("bash", ...)` to `exec.Command("definitely-not-a-shell", ...)`,
run the same test, and confirm it fails with `bash "...": exec: "definitely-not-a-shell": executable file not found`
rather than passing. Then revert that temporary edit and re-run Step 4. Report both
outputs. This is the mutation check that the `errors.As` branch still guards the right case.

- [ ] **Step 6: Verify formatting and the wider build**

```bash
cd /Users/ronin/Projects/picfetch && make fmt-check && go vet ./... && go build ./...
```

Expected: all three succeed with no output about `tag_test.go`.

- [ ] **Step 7: Report, do not commit**

Report the diff, both Step 5 outputs, and the Step 6 result. Proposed commit message:

```text
fix: match wrapped exec.ExitError in wingettag test
```

### Primary-agent review gate after Task 3

Confirm the temporary mutation in Step 5 was actually reverted (`git diff` must show only
the import line and the assertion block). Confirm `errors` sits in the correct
`goimports -local` group. Re-run `go test ./scripts/wingettag/...` yourself.

---

## Task 4: Exclude test files from `DuplicatedCode`

**Files:**
- Modify: `qodana.yaml` — the trailing `include:` block gains an `exclude:` companion

**Interfaces:**
- Consumes: Task 1's Step 7 result (every duplication fragment is in a `_test.go` file).
- Produces: a `qodana.yaml` whose next CI run reports zero `DuplicatedCode` results while
  still reporting duplication in production Go files. Task 5 pushes it; Task 7 confirms
  the count.

**Context the executor needs:** the current file ends with

```yaml
linter: jetbrains/qodana-go:2026.2
include:
  - name: DuplicatedCode
```

`DuplicatedCode` is not in the `qodana.starter` profile, so the `include:` is what turns
it on at all. The `exclude:` key takes the same `name` plus an optional `paths:` list.
**Glob support in `paths:` is the one uncertain part of this task** — Qodana's own
documentation shows directory paths, and per-file globs are not guaranteed to be honored
by the `2026.2` linter. The task therefore ends with a CI-verified result, not with a
local claim.

- [ ] **Step 1: Read the current file end**

```bash
tail -5 /Users/ronin/Projects/picfetch/qodana.yaml
```

Expected: the `linter:` line followed by the two-line `include:` block.

- [ ] **Step 2: Append the exclusion**

Append to `qodana.yaml`, preserving the existing content exactly:

```yaml
# Every DuplicatedCode fragment in this repository lives in a _test.go file:
# shared table-test scaffolding that reads better repeated than extracted.
# Production duplication is still reported. See
# finished_refactorings/2026-08-29-qodana-duplication-close.md.
exclude:
  - name: DuplicatedCode
    paths:
      - "**/*_test.go"
```

- [ ] **Step 3: Check the file is still valid YAML**

```bash
cd /Users/ronin/Projects/picfetch && python3 -c "import sys,yaml;d=yaml.safe_load(open('qodana.yaml'));print(d['include'],d['exclude'])"
```

Expected: `[{'name': 'DuplicatedCode'}] [{'name': 'DuplicatedCode', 'paths': ['**/*_test.go']}]`.
If `python3` has no `yaml` module, fall back to
`ruby -ryaml -e 'p YAML.load_file("qodana.yaml")["exclude"]'`. If neither parser is
available, report that fact and rely on the Task 5 CI run to validate the syntax.

- [ ] **Step 4: Record the fallback in the working notes**

Append to `plans/2026-08-29-qodana-evidence.md` a section `## Exclusion fallback` stating:
if the Task 5 run still reports `DuplicatedCode` results, the glob was not honored, and
the fallback is to replace `paths:` with the explicit list of the **29 test files** that
hold flagged fragments:

```bash
jq -r '.runs[].results[] | select(.ruleId=="DuplicatedCode") | .locations[].physicalLocation.artifactLocation.uri' "$SARIF" | sort -u
```

**The fallback must list files, not directories.** Those 29 files sit in only 10
directories, and every one of those directories also holds production Go code — excluding
`internal/imaging` to silence `internal/imaging/raw_test.go` would suppress duplication
reporting for `internal/imaging/loader.go` as well, which defeats the point of keeping the
inspection enabled. Record that reasoning in the section.

Also record that the SARIF holds only the 63 fragments CI actually wrote out, so a
file-level fallback derived from it would miss `internal/imaging/loader_test.go` — one of
the two files hit by the serialisation defect Task 2 found. Add that file to the fallback
list explicitly, giving 30 files, and say why it is not in the `jq` output.

- [ ] **Step 5: Confirm nothing else changed**

```bash
cd /Users/ronin/Projects/picfetch && git diff --stat
```

Expected: `qodana.yaml` and `plans/2026-08-29-qodana-evidence.md` only (plus Task 3's
`scripts/wingettag/tag_test.go` if it is not yet committed).

- [ ] **Step 6: Report, do not commit**

Report the diff and the Step 3 output. Proposed commit message:

```text
ci: stop reporting duplicated code in test files
```

### Primary-agent review gate after Task 4

Read `qodana.yaml` in full and confirm the `include:` block survived — deleting it would
silently disable duplication reporting everywhere, which is the failure mode this task
must not produce. Confirm the comment above `exclude:` names the reason and points at the
archived plan path Task 6 will actually create.

---

## Task 5: External gate — user commits and pushes, agents observe

This task is not delegated. The user performs the git operations; a subagent then reads
the result.

- [ ] **Step 1 (user):** review the working tree from Tasks 1–4 and commit, ideally as
      two commits: the `tag_test.go` fix and the `qodana.yaml` change.
- [ ] **Step 2 (user):** `git push origin main`.
- [ ] **Step 3 (agent, `general-purpose`, `sonnet`):** wait for the push-triggered run and
      capture its identity:

```bash
gh run list --workflow=qodana_code_quality.yml -L 1
gh run watch "$RUN_ID" --exit-status
gh run view "$RUN_ID" --json headSha,conclusion,event
```

- [ ] **Step 4 (agent):** download and count, using Task 1's Step 3 and Step 4 commands
      against the new artifact.

Expected after the change: `DuplicatedCode` absent from the SARIF entirely, and
`GoTypeAssertionOnErrors` absent as well, leaving **0 SARIF results**. The summary CSV may
still show a nonzero `DuplicatedCode` count, because that tally is pre-filter — say so
explicitly rather than treating it as a failure.

- [ ] **Step 5 (agent):** append the new run's identity and counts to
      `plans/2026-08-29-qodana-evidence.md` under `## Confirmation run`.

### Primary-agent review gate after Task 5

If `DuplicatedCode` results survive, do not adjust the count expectation — apply Task 4's
Step 4 fallback, and return to Task 5. If the SARIF is empty, verify that duplication
reporting is not simply off by checking that the summary CSV still shows
`DuplicatedCode` was performed a nonzero number of times.

---

## Task 6: Reconcile the backlog and archive

**Files:**
- Modify: `todos.md` — move both Qodana headings from `## TODO` to `## Done` → `#### Internal`
- Modify: `AGENTS.md` — add one bullet under `## Build and Verification`
- Modify: `finished_refactorings/2026-08-29-qodana-ci-duplication-reporting.md` — add a
  superseded-by line directly under its title, changing nothing else
- Move: `plans/2026-08-29-qodana-duplication-close.md` and
  `plans/2026-08-29-qodana-evidence.md` into `finished_refactorings/`

**Interfaces:**
- Consumes: the verdicts from Tasks 1, 2 and 5.
- Produces: a `todos.md` whose `## TODO` section is empty of Qodana items.

**Context the executor needs:** `todos.md`'s Done entries are written as dense prose
paragraphs that state what changed, what the earlier diagnosis got wrong, and how the
result was verified. Match that register exactly — read three existing entries first.
Two claims in the current TODO text are now known to be wrong and must be corrected
rather than deleted: "the next CI run should show 30 fewer problems (155 -> 125 on a full
scan)", and the framing that CI "under-reports duplication".

- [ ] **Step 1: Read the target register**

```bash
sed -n '1,40p' /Users/ronin/Projects/picfetch/todos.md
```

- [ ] **Step 2: Write the false-positive entry**

Under `## Done` → `#### Internal`, as the new first bullet, record: the 12
`//goland:noinspection` suppressions are confirmed in CI at `210fee5`, because the
summary CSV still counts all 12 findings while the SARIF contains none of them; the other
18 suppressions were `_` renames that never produced a finding; and the predicted
"155 -> 125" figure was wrong because it compared an IDE Project Default scan against a
`qodana.starter` CI scan.

- [ ] **Step 3: Write the duplication entry**

As the next bullet, record: CI does not under-report duplication. Quote the four numbers
and their units (33 SARIF cluster results, 71 location slots, 63 distinct fragments,
75 CSV rows), state that the slots are not disjoint because 8 fragments belong to 2
clusters each, and state that Qodana emits one result per cluster. Then give the closed
arithmetic: 75 CSV rows = 71 test-file fragments + 4 production fragments suppressed at
source in the orientation pixel loops, and 71 − 8 serialisation losses = 63 SARIF
fragments, so the 12-fragment CSV-to-SARIF gap is 8 + 4 with nothing unexplained. Then state that all
33 clusters were test-only, that `qodana.yaml` now excludes `_test.go` from
`DuplicatedCode`, and cite the confirmation run ID from Task 5. Include Task 2's verdict
sentence on the IDE delta.

- [ ] **Step 3a: Supersede the two now-outdated notes in the evidence file**

`finished_refactorings/2026-08-29-qodana-evidence.md` (after Step 7 moves it; before that
it is under `plans/`) contains two statements that later runs overtook. Do not delete
them — append a clearly marked superseding note directly beneath each, so the archived
document shows what was believed and when it changed.

Under `## Suppression accounting`, beneath the sentence saying the commands do not
establish what the 12 fragments are, append a note stating: run `33274422606` at `ed3d4e6`
excluded the 30 test files and the CSV's `DuplicatedCode` count fell from 75 to 4, not to
0; 4 is the count of source-local `//goland:noinspection` suppressions on the orientation
pixel loops; therefore 75 CSV rows = 71 test-file fragments + 4 production fragments, and
the 12-fragment gap = 8 serialisation losses + 4 source suppressions, with nothing left
unexplained.

Under `## GoLand comparison at 210fee5`, beneath the sentence saying 4 of the 12 remain
unexplained, append a one-line pointer to the note above.

- [ ] **Step 3b: Write the serialisation-bug entry**

As a third bullet, and under `## TODO` rather than `## Done` — this one is not closed —
add a heading `### Qodana drops detected duplicates during serialisation (upstream)`.
Record: at `210fee5`, run `33270269940`, the IDE reports 71 `DuplicatedCode` fragments and
the CI SARIF 63, CI being a strict subset; the 8 missing fragments are 7 in
`internal/imaging/loader_test.go` and 1 at `internal/update/tufroot_test.go:173`; the run's
own `log/idea.log` carries exactly 3 `#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find
duplicate problem in db" warnings naming exactly those two files, emitted after
`The Project analysis stage completed in 41s`, so detection succeeded and serialisation
failed. State that `qodana.yaml`'s new `_test.go` exclusion makes this invisible in this
repository — every dropped fragment is in a test file — which is why the finding is
recorded here rather than being allowed to disappear with the rule. Point at
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded offsets and at
Task 8's report file for the upstream text. Note that 4 fragments of the CSV-to-SARIF gap
remain unexplained.

- [ ] **Step 3c: Write the exclusion-mechanics entry**

As a fourth bullet under `## Done` → `#### Internal`, record what it took to make the
exclusion work, because the next person to edit `qodana.yaml` needs it. State: `exclude:`
with a `paths:` glob compiles into a real scope — `effective.profile.xml` shows
`<scope name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION" enabled="false" />`
nested inside the `DuplicatedCode` inspection — but suppresses nothing. Both
`"**/*_test.go"` (run `33273666731`) and `"**_test.go"` (run `33274030031`) failed that
way, the second being the dialect JetBrains uses for its own built-in scopes
(`glob:**.md`, `glob:**.test.ts`). Neither failure was a delivery problem: each run's
`qodana-config.json` echoes the pattern it was given and neither log shows cache reuse.
Explicit file paths work — run `33274422606` returned 0 SARIF results. Note that the CSV
count falling to 4 rather than 0 is the proof the inspection was narrowed rather than
disabled.

- [ ] **Step 4: Delete the two TODO headings**

Remove `### Qodana: false positives are flagged in code (done, needs CI confirmation)`
and `### Qodana: CI under-reports duplication against a full scan` with their bodies from
`## TODO`, leaving the `## TODO` heading itself in place.

- [ ] **Step 5: Add the AGENTS.md bullet**

Under `## Build and Verification`, after the existing Makefile bullets:

```markdown
- **Reading a Qodana report:** `qodana.sarif.json` is the post-suppression result set and
  counts one result per duplicate *cluster*; `log/qodana_inspections_summary.csv` counts
  every finding *before* suppressions and one row per *fragment*. The two disagree by
  design — compare fragment sets, never totals. CI runs the `qodana.starter` profile, not
  the IDE Project Default, so IDE and CI totals are not comparable either.
```

- [ ] **Step 6: Mark the predecessor plan superseded**

Insert immediately below the title line of
`finished_refactorings/2026-08-29-qodana-ci-duplication-reporting.md`:

```markdown
> **Superseded 2026-08-29** by `finished_refactorings/2026-08-29-qodana-duplication-close.md`.
> Chunks 1–2 landed as commit `210fee5`. Chunks 3–7 were replaced: their premise, that CI
> under-reports duplication, was disproved by run `33270269940`.
```

Change nothing else in that file.

- [ ] **Step 7: Archive this plan and its evidence**

```bash
cd /Users/ronin/Projects/picfetch
mv plans/2026-08-29-qodana-duplication-close.md finished_refactorings/
mv plans/2026-08-29-qodana-evidence.md finished_refactorings/
rmdir plans 2>/dev/null || true
```

Plain `mv`, not `git mv`: these files have never been committed, so git does not track them
and `git mv` would fail. The user commits them from `finished_refactorings/` at the end.

- [ ] **Step 8: Check every internal path reference still resolves**

```bash
cd /Users/ronin/Projects/picfetch
grep -rhoE '(plans|finished_refactorings|internal|scripts)/[A-Za-z0-9._/-]+\.(md|go|yaml)' todos.md AGENTS.md finished_refactorings/2026-08-29-qodana-*.md | sort -u | while read -r p; do test -e "$p" || echo "MISSING: $p"; done
```

Expected: no `MISSING:` lines.

- [ ] **Step 9: Report, do not commit**

Report the full `todos.md` diff. Proposed commit message:

```text
docs: close the Qodana reporting todos
```

### Primary-agent review gate after Task 6

Read both new `todos.md` entries against the evidence file and confirm every number in
them appears there. Confirm both corrections from the Context note are present — a Done
entry that silently drops a wrong prediction instead of correcting it fails this gate.
Confirm Step 8 printed nothing.

---

## Task 7: Independent final audit

**Files:** none modified unless a defect is found.

**Interfaces:**
- Consumes: everything above.
- Produces: a pass/fail verdict per acceptance-checklist line.

- [ ] **Step 1: Full verification suite**

```bash
cd /Users/ronin/Projects/picfetch && make fmt-check && go vet ./... && go build ./... && go test -timeout 20m -race ./...
```

Expected: all pass. Quote the final `ok`/`FAIL` summary line count, not the whole log.

- [ ] **Step 2: Confirm the scope of the change**

```bash
cd /Users/ronin/Projects/picfetch && git diff --stat 210fee5..HEAD
```

Expected files only: `scripts/wingettag/tag_test.go`, `qodana.yaml`, `todos.md`,
`AGENTS.md`, and the two archived documents plus the superseded-by line. **Any production
`.go` file in this list is a failure** — report it and stop.

- [ ] **Step 3: Confirm no behavioural surface moved**

```bash
cd /Users/ronin/Projects/picfetch && git diff 210fee5..HEAD -- go.mod go.sum translations assets FyneApp.toml Makefile .github | wc -l
```

Expected: `0`.

- [ ] **Step 4: Re-check the closed findings against the confirmation run**

Using the artifact from Task 5, re-run Task 1's Step 3 command. Expected: `0` results, and
`GoTypeAssertionOnErrors` absent from the summary CSV's nonzero rows.

- [ ] **Step 5: Walk the acceptance checklist**

Answer every line below with pass/fail and one sentence of evidence. Do not mark a line
pass on the strength of another agent's report; re-run the command.

- [ ] **Step 6: Report**

Report the checklist verdict. Do not commit. If any line fails, name the task that must
be re-run.

### Primary-agent final review

Spot-check at least three of Task 7's pass claims by re-running their commands. Then hand
the result to the user with the proposed commit list.

---

## Task 8: Draft the upstream bug report

**Files:**
- Create: `plans/2026-08-29-qodana-serialisation-bug-report.md`

**Interfaces:**
- Consumes: `## GoLand comparison at 210fee5` and `## Minimal reproduction` from
  `finished_refactorings/2026-08-29-qodana-evidence.md` (Task 6 moved it there), and the
  `todos.md` entry from Task 6 Step 3b.
- Produces: a self-contained report the user pastes into JetBrains YouTrack.

**Context the executor needs:** the defect belongs to the Qodana linter
(`jetbrains/qodana-go:2026.2`), not to `JetBrains/qodana-action`, so the destination is
the YouTrack **QD** project at `https://youtrack.jetbrains.com/issues/QD`, not GitHub.
Filing requires a JetBrains account, so this task **drafts** the report; the user submits
it. Do not attempt to file it, and do not use `gh` for it.

The report is read by JetBrains engineers who have never seen this repository. Write
normal, complete English prose — no compressed style, no project shorthand, no
unexplained internal file references. It must stand alone.

- [ ] **Step 1: Re-read the evidence**

```bash
cd /Users/ronin/Projects/picfetch
sed -n '/^## GoLand comparison at 210fee5/,/^## Reproduction of the GoLand comparison/p' finished_refactorings/2026-08-29-qodana-evidence.md
```

Use only facts that appear there. Add nothing from your own inference.

- [ ] **Step 2: Write the report**

Create `plans/2026-08-29-qodana-serialisation-bug-report.md` with these sections:

`## Title` — one line: `DuplicatedCode findings are detected but silently dropped from the SARIF ("Can't find duplicate problem in db")`.

`## Environment` — linter `jetbrains/qodana-go:2026.2`, `JetBrains/qodana-action@v2026.2`,
`ubuntu-latest`, profile `qodana.starter` with `DuplicatedCode` re-enabled through
`qodana.yaml`'s `include:`, full scan (not `pr-mode`), a public Go repository.

`## Summary` — two or three sentences: the analysis detects duplicate fragments, logs a
warning that it cannot find them when writing results, and emits a SARIF missing exactly
those fragments. The report and the SARIF agree with each other, so the loss happens
before either is written.

`## What we observed` — the 71-versus-63 fragment counts with their units, the
strict-subset relationship, the 8 missing fragments listed by file and line, and the 3 log
warnings quoted verbatim with their timestamps and the preceding
`The Project analysis stage completed in 41s` line.

`## Why we believe detection succeeded` — the warnings are emitted after the analysis stage
completes, and the decoded `line:charOffset` pairs land inside the duplicated regions the
IDE reports, anchored one line apart on the same repeating pattern.

`## Reproduction` — the minimal pair from the evidence file's `## Minimal reproduction`
section, with enough surrounding context that a reader can construct an equivalent case.

`## Impact` — a duplication count taken from the SARIF understates the real count,
silently, and the only signal is a WARN line in `idea.log` that no gate reads.

`## What we would expect` — either the fragments appear in the SARIF, or the run fails
loudly rather than emitting a quietly incomplete result set.

`## A second, separate issue: exclude globs compile but do not suppress` — a short section,
clearly marked as a distinct defect the maintainers may want to split into its own ticket.
State that `exclude: - name: DuplicatedCode / paths: ["**/*_test.go"]` and the same with
`"**_test.go"` both produce a compiled scope
`<scope name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION" enabled="false" />`
inside the `DuplicatedCode` inspection in `effective.profile.xml`, yet neither suppresses
any finding, while an explicit list of the same files as literal paths does suppress them.
Give the three run IDs. Note that the second pattern matches the dialect of Qodana's own
built-in scopes, `glob:**.md` and `glob:**.test.ts`.

- [ ] **Step 3: Check the report stands alone**

Re-read it as someone with no access to this repository. Every file path must be
introduced, every number must carry its unit, and no sentence may depend on a claim that
is only in `todos.md` or the evidence file. Fix anything that fails that test.

- [ ] **Step 4: Report, do not commit and do not file**

Report the file path and a one-paragraph summary. Proposed commit message:

```text
docs: draft the upstream Qodana serialisation bug report
```

### Primary-agent review gate after Task 8

Verify every quoted log line against `$SCRATCH/qodana/x/log/idea.log` character for
character, and every fragment line number against the SARIF. Confirm the report contains
no claim the evidence file does not support. Then present it to the user for submission —
**do not file it anywhere.**

---

## Acceptance checklist

- [ ] Run `33270269940` at `210fee5` is a full scan with a downloadable artifact.
- [ ] The 12 `//goland:noinspection` suppressions are shown effective in CI by the
      CSV-vs-SARIF difference, not by a headline total.
- [ ] The `DuplicatedCode` 33 / 71 / 63 / 75 counts are reproduced and their four units
      named, with the non-disjointness of location slots stated explicitly.
- [ ] `relatedLocations` is confirmed empty, ruling out hidden cluster members.
- [ ] All 33 clusters are shown to be test-only; production duplication is zero.
- [ ] The IDE/CI comparison is made on the same SHA, with the profile difference stated.
- [ ] Any residual IDE/CI delta is explained by a fragment-set difference or reproduced
      concretely; no total is reconciled by assertion.
- [ ] `scripts/wingettag/tag_test.go` uses `errors.As`, and the new branch is shown
      reachable by mutation.
- [ ] `qodana.yaml` excludes `_test.go` from `DuplicatedCode` and still `include:`s it.
- [ ] A confirmation CI run shows the expected post-change counts.
- [ ] `todos.md`'s TODO section holds no Qodana items, and both previously wrong claims
      are corrected rather than deleted.
- [ ] `AGENTS.md` records the report-reading semantics.
- [ ] No production Go file, dependency, translation, golden, or workflow file changed.
- [ ] `make fmt-check`, `go vet`, `go build`, and the race test suite all pass.
- [ ] `todos.md` carries an open entry for the Qodana serialisation defect, and that entry
      survives the `_test.go` exclusion that hides it.
- [ ] The upstream report stands alone, quotes the log verbatim, and was NOT filed by an
      agent.
- [ ] Both plan documents are archived under `finished_refactorings/`.
- [ ] No commit was created by a subagent.

---

## Open decisions

### D1 — Who commits?

**Recommended:** the user, as in the predecessor plan. Subagents stage and report;
Task 5 is an explicit external gate. Alternative: authorize agents to commit on a branch
and open a PR, which would also exercise the `pr-mode: true` path of the revised workflow
and give a second data point for free.

### D2 — Leave the `qodana.starter` profile as is?

**Recommended:** yes, out of scope. Moving CI to `qodana.recommended` or to the IDE
Project Default would make CI and IDE totals comparable but changes what the gate
enforces, and would surface an unknown number of new findings. Worth its own todo, which
Task 6 could add if you want it.

### D3 — Remove the inert quick-fix configuration?

**Recommended:** leave `push-fixes: pull-request` and the `contents: write` /
`pull-requests: write` permissions alone. They are inert without `--apply-fixes`, but
narrowing workflow permissions is adjacent security cleanup and deserves its own change.
