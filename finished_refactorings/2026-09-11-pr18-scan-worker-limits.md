# PR 18: trial rejection and worker limits

Route: Standard. Two findings from Codex's review of `c1a3369`.
Root owns both assessments, tests and fixes; no delegated implementation.

## Acceptance criteria and tasks

1. A truncated explicit trial records rejection, admits no analysis/partial
   files, restores the empty viewer surface, and spends pending launch actions.
   Files: `internal/ui/drop.go`, existing `explorer_test.go`.
   Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/trial_truncated$'`.
2. Each production analysis captures the caller's encoded-size limit and sends
   it to the worker, which installs it before any image reads. A later analysis
   captures an updated limit. No new viewer configuration or parallel load path.
   Files: `internal/similarity/client.go`, existing `explorer_local_test.go`.
   Verify: `go test -tags explorertrial ./internal/ui -run '^TestVisualSimilarityExplorerLocal$/configured_file_size_limit$'`.

These tasks are independent and implemented inline. Budget: no spawns; focused
local tests only; full CI remains GitHub's responsibility under the user's
review-loop instruction. No new UI strings, packages, or top-level UI tests.

## Evidence

Both regressions failed before their fixes: the truncated trial left its empty
surface hidden, and the worker accepted a padded 2 MiB image despite the viewer's
1 MiB limit. After the fixes, the focused trial rejection/launch and real-worker
tests passed together in 2.111s:

```sh
go test -tags explorertrial ./internal/ui -run '^TestVisualSimilarityExplorer$/(trial_truncated|trial_launch)$|^TestVisualSimilarityExplorerLocal$/configured_file_size_limit$' -count=1 -timeout 3m
```

The worker regression also starts a second analysis after raising the limit to
3 MiB and verifies that both images succeed. `git diff --check` passed. Subsequent
external reviews and full CI results are recorded on PR 18. Findings:
- https://github.com/frathe/picfetch/pull/18#discussion_r3984185513
- https://github.com/frathe/picfetch/pull/18#discussion_r3984185515
