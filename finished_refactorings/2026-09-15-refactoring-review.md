# Refactoring follow-up: GitHub Codex review loop

Status: implementation complete; PR review and CI govern acceptance. No merge
or release is authorized. Route: Standard for review coordination; promote any
cross-package finding to an explicit contract before implementation.

## Scope and authorization

The user requested opening a PR for `feature/refactoring` and invoking the
GitHub Codex review loop with SDD/TDD. This authorizes PR creation, fix commits,
pushes, review requests/replies and thread resolution. Follow `AGENTS.md`'s loop;
the lead owns assessment, implementation, review and fixes. Use focused local
tests; GitHub CI owns the complete race suite during this workflow.

The five implementation commits were rebased onto `dd46855` (v1.1.3):
`8f30c68` browsing transitions; `2827165` cache-maintenance handoff; `739f31f`
warm-path coverage; `33e6bf4` continue-in-memory policy; `0d917e6` bounded decoder.
`git range-diff` confirms their patches match the previously verified versions
apart from release-reset todos. Comparing the previous tested head `a4b5e40`
with `0d917e6` changes only release notes, `FyneApp.toml` and todos; Go code is
identical. Existing implementation plans retain the original local evidence.

## Acceptance and loop

1. Publish a reviewable PR against current `main`, with the concrete behavior,
   scope, local evidence and benchmark limits. Check clean worktree and branch
   ancestry first; use a normal push.
2. Read every unresolved thread and all code/security summaries, including
   outdated threads. Validate findings against current code and contracts.
   For each confirmed defect, specify its invariant and add a failing useful
   regression before the smallest fix. Use deterministic queues/barriers.
3. Run focused tests and negative guard checks. Inspect changed Go files in
   GoLand, including weak warnings; keep formatting, exclusions, shard mapping,
   architecture and todos current. Commit/push, explain each disposition with
   evidence, resolve addressed threads and obtain another fresh Codex review.
4. Finish only with a clean Codex code review on the latest pushed head, completed
   security review with no actionable findings, required CI passing, and no
   actionable Qodana/CodeQL results. Inspect Qodana's post-suppression
   `qodana.sarif.json`, even if its checks are green. Inspect failed job logs.
   Never reuse a clean review of an older head.

Commands: `gh pr view`, GraphQL review-thread pagination, issue/review comments,
check runs and Actions artifacts bind each observation to its head/run ID.
The live connector advertises `@codex review`; security runs alongside when
enabled and also advertises `@codex security review`. Request once only when no
appropriate review is queued/running. Security summary status and `headSha` must
match the PR head. Qodana uses the `qodana-report` artifact.

Initial focused verification uses `go test -race -tags no_emoji -count=1` for
`internal/fileidentity`, `internal/ui/analysiscache`, `internal/ui/visualsearch`,
similarity's `TestAnalysisCache|TestSearch`, and root UI's
`TestFindMoreLikeThis|TestLoad`. Formatting and shard checks run before publication.

Graph: initial verification -> PR -> findings/dispositions -> fresh review/CI;
repeat until acceptance criteria pass. Do not merge, release or archive plans
before branch acceptance. PR comments/checks are the authoritative live delivery
record; local retrieval snapshots and full gate evidence live under
`.scratch/refactoring-review/`, avoiding commits solely to update polling state.

## Delegation and budget

One read-only scout identifies artifact names and bot completion formats while
the lead prepares the PR. G1 bounded question; G2 exact workflow/API references;
G3 no writes; G4/G5 independent evidence retrieval map. No implementation or
review is delegated. Budget: one scout, focused local gates per confirmed fix,
zero repeated broad local race suites; remote review rounds are outcome-driven.

## Initial verification

The stated focused race commands passed on the rebased code. Logs:
`/tmp/picfetch-refactoring-review-{features,similarity,root}.log`.
`make fmt-check check-test-shards-direct` passed and confirmed all 686 root UI
runnables assigned; log `/tmp/picfetch-refactoring-review-static.log`.
`git diff --check` passed. No Go files changed during PR preparation, so the
existing per-file GoLand and full local gate records remain applicable evidence.
