# Toast.go Import Grouping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user.

**Goal:** Put a blank line between the `fyne.io/...` and `github.com/frathe/picfetch/...` import groups in `internal/ui/toast.go`, matching neighbouring `internal/ui` files, and move the todo to Done.

**Architecture:** House style in `internal/ui` is three blank-line-separated import groups when all three kinds are present: stdlib, then `fyne.io/...`, then local `github.com/frathe/picfetch/...`. `gofmt` preserves those blank lines but does not insert them. `toast.go` is the only `internal/ui/*.go` file that currently merges fyne and local into one group. This is a one-blank-line style fix, not a behavior change.

**Tech Stack:** Go 1.26 (see `go.mod`), Fyne v2, `gofmt`. Do not add `goimports` unless the user explicitly chooses the parked Task 3.

## Status check (2026-08-25) — this todo is old but not done

Checked against the tree before writing this plan.

| `todos.md` TODO | Still open? | This plan |
|-----------------|-------------|-----------|
| `internal/ui/toast.go` import grouping | **Yes.** Two groups: stdlib, then fyne+local merged. Neighbours that import all three kinds use three groups (47 files under `internal/ui/*.go`). Present since `9eda18c`. | **This plan.** |
| `finishLoad` (`internal/ui/load.go:192-305`) | Yes. Still one ~114-line pipeline. | **Out of scope.** Bullet says decompose only if it needs to change anyway. |
| `internal/imaging/exif.go` (~687 lines) | Yes. Still one file. | **Out of scope.** Parse/format split is cosmetic. |

If the user instead wants `finishLoad` or `exif.go`, stop; do not execute this plan.

## Approaches considered

1. **One blank line in `toast.go` only (this plan).** Matches the todo text. `gofmt` keeps the extra group. Nothing in CI will re-merge it unless someone later runs unconfigured `goimports`.
2. **Add `goimports -local github.com/frathe/picfetch` to `make verify` and CI.** Would lock the grouping and catch future merges. The todo names the missing check as *why nothing catches it*, not as work to do. It would also rewrite files that currently put `fyne.io` in its own group even when other third-party imports sit beside it (e.g. `internal/imaging/save.go` groups `fyne.io` with `golang.org/x/image`). Parked as Task 3; do not run it unless the user says so.
3. **Repo-wide import rewrite.** Overkill. `toast.go` is the only `internal/ui/*.go` merge of fyne+local. Imaging files that mix `fyne.io` with `github.com/gen2brain` / `golang.org/x` are a different convention (third-party together).

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not change `finishLoad`, `internal/imaging/exif.go`, `Makefile`, `.github/workflows/ci.yml`, or any file other than those listed in the active task.
- Do not install or run `goimports` in Tasks 1–2.
- Do not add an import-grouping unit test. This is a style blank line; a linter test would be a new CI policy (Task 3).
- Do not add `TODO`/`FIXME` comments to source. Open work stays in `todos.md`.
- Do not update `ARCHITECTURE.md` (no package/file move).
- Preserve `gofmt` formatting. Tabs, not spaces.
- TDD does not apply: there is no failing production behavior. Verification is `gofmt -l` plus a compile/test of `./internal/ui`.
- Subagents must not start Task N+1 themselves. They stop after their task’s verification and report.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `composer-2.5-fast` | One blank line in one file; complete replacement in the brief. |
| Task 2 implementer | `composer-2.5-fast` | Docs-only `todos.md` move. |
| Task reviewer (each task) | `cursor-grok-4.5-high-fast` | Mid-tier floor for reviewers. Diff is tiny; Opus is wasted. |
| Parent review / fix after each task | this session (do not dispatch) | User asked the parent to review and fix after every step. |
| Task 3 implementer (only if unparked) | `cursor-grok-4.5-high-fast` | Makefile + CI + tool install; integration, not design. |
| Final whole-branch review | `cursor-grok-4.5-high-fast` | Same reason: mechanical style diff. Do **not** use Opus. |

Do not use `claude-opus-5-thinking-high` for any task in this plan. The work cannot be made complex enough to need it.

## File structure

- Modify: `internal/ui/toast.go` — import block only.
- Modify: `todos.md` — move the toast bullet from `## TODO` to `## Done`.
- Parked: `Makefile`, `.github/workflows/ci.yml` — only if Task 3 is unparked.

---

### Task 1: Split the toast.go import groups

**Model:** `composer-2.5-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)

**Files:**
- Modify: `internal/ui/toast.go` (import block, currently lines 5–14)
- Test: no new test file. Cover with `gofmt -l internal/ui/toast.go` and `go test -count=1 -run TestReportClipboardError_ShowsToast ./internal/ui/`

**Interfaces:**
- Consumes: nothing from later tasks
- Produces: three import groups in `toast.go` (stdlib / `fyne.io` / `github.com/frathe/picfetch`). Task 2 only needs this file to already be grouped; it does not import Go symbols.

- [ ] **Step 1: Confirm the bug is still present**

Read `internal/ui/toast.go` lines 5–14. The current block must still be exactly:

```go
import (
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/ui/widgets"
)
```

If a blank line already sits between the `fyne.io/fyne/v2/container` line and the `github.com/frathe/picfetch/internal/completion` line, report `DONE` with no edits (todo already landed). Do not “improve” anything else.

- [ ] **Step 2: Insert the missing group separator**

Replace the import block so it matches `internal/ui/components.go` (stdlib, blank, fyne, blank, local):

```go
import (
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/ui/widgets"
)
```

The only change is one blank line after `"fyne.io/fyne/v2/container"`. Do not reorder, add, or drop import paths. Do not touch any other lines in the file.

- [ ] **Step 3: Verify gofmt keeps the groups**

Run:

```bash
gofmt -l internal/ui/toast.go
```

Expected: empty stdout (file is already `gofmt`-clean; `gofmt` preserves existing import-group blank lines).

If `gofmt -l` prints `internal/ui/toast.go`, run `gofmt -w internal/ui/toast.go` and inspect the import block. It must still have three groups. If gofmt collapsed them, stop and report `BLOCKED` (that would contradict how `gofmt` works in this repo).

- [ ] **Step 4: Compile/test a toast-using helper**

Run:

```bash
go test -count=1 -run TestReportClipboardError_ShowsToast ./internal/ui/
```

Expected: `PASS`. This does not test import grouping; it proves the file still builds inside package `ui`. Do not run `go test -race ./...` in this task.

- [ ] **Step 5: Do not commit**

Leave the working tree dirty. Report:

- Status: `DONE` or `DONE` (already grouped) or `BLOCKED`
- Diff summary: one blank line in `internal/ui/toast.go` (or none)
- Commands run and their exit codes
- Concerns, if any

---

### Task 2: Move the toast bullet to Done in todos.md

**Model:** `composer-2.5-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)

**Files:**
- Modify: `todos.md`

**Interfaces:**
- Consumes: Task 1 has grouped `internal/ui/toast.go` (or confirmed it already was).
- Produces: `todos.md` `## Done` contains the toast outcome; `## TODO` no longer lists it. `finishLoad` and `exif.go` bullets stay under `## TODO`.

- [ ] **Step 1: Read `todos.md` as it exists after Task 1**

Do not assume line numbers from this plan. The `## TODO` section currently starts with the toast.go bullet (the paragraph that cites `9eda18c` and `goimports -local`). `## Done` already has Architecture trim, Menu Window, Menu Actions, and Never-started canary.

- [ ] **Step 2: Add a Done entry**

Insert this bullet at the top of `## Done` (newest first), immediately under the `## Done` heading and before the Architecture-trim bullet:

```markdown
- Split `internal/ui/toast.go` import groups so stdlib, `fyne.io`, and
  `github.com/frathe/picfetch` are blank-line-separated like neighbouring
  `internal/ui` files (2026-08-25).
```

- [ ] **Step 3: Remove the open TODO bullet**

Delete the entire toast.go bullet from `## TODO` (the three wrapped lines that begin with `` - `internal/ui/toast.go` merges `` and end with `reason.`).

Leave these two bullets in `## TODO`, unchanged:

```markdown
- `finishLoad` (`internal/ui/load.go:192-305`) is a 114-line do-everything pipeline (vector setup, fade, overlay, zoom,
  resize, title, animation, preload). It is linear and well-commented; decompose into named steps only if it needs to
  change anyway.

- `internal/imaging/exif.go` (687 lines) holds two parsers plus IFD walking plus display formatting. Cohesive and
  well-tested; a parse/format file split is cosmetic.
```

Do not edit Menu Window, Menu Actions, Never-started canary, or the Windows grid Ctrl+click note.

- [ ] **Step 4: Sanity-check the markdown**

`todos.md` must still have exactly one `## Done`, one `## ACTIVE DEVELOPMENT`, one `## TODO`, and one `## not deemed worth implementing (edge cases)`. The toast `9eda18c` / `goimports` paragraph must not appear under `## TODO`.

- [ ] **Step 5: Do not commit**

Report status, a one-line summary of the `todos.md` edit, and concerns.

---

### Task 3 (PARKED): Enforce grouping with goimports -local

**Do not dispatch this task unless the user explicitly unparks it.**

**Model if unparked:** `cursor-grok-4.5-high-fast` (implementer and reviewer). Still not Opus.

**Why parked:** The todo calls `goimports -local` the missing detector, then says the fix is worth doing the next time `toast.go` imports change. Adding a tool to `make verify` and CI is a policy change: `install-tools`, contributor docs, and a first `goimports -w` pass that may regroup `internal/imaging/save.go` (`fyne.io` currently shares a group with `golang.org/x/image`) and any other file that currently isolates `fyne.io` from other third-party paths.

If unparked, a follow-up plan is required. Do not invent Makefile/CI text in this task list.

---

## Parent review checklist (after each task)

The parent (this session) does this before the next dispatch. Subagents do not self-approve.

**After Task 1:**

1. `git diff internal/ui/toast.go` is only the blank line in the import block.
2. `gofmt -l internal/ui/toast.go` is empty.
3. No other `.go` files changed.
4. If the diff is dirty (comments rewritten, imports reordered, unrelated cleanup), revert the extra and keep the blank line.

**After Task 2:**

1. `git diff todos.md` only moves the toast item to Done.
2. `finishLoad` and `exif.go` remain under `## TODO`.
3. No Go files besides `toast.go` are in the working tree for this work.

**Before telling the user it is done:**

```bash
gofmt -l internal/ui/toast.go
go test -count=1 -run TestReportClipboardError_ShowsToast ./internal/ui/
```

Both must succeed. Full `go test -race ./...` is optional for a blank line; skip it unless Task 3 was unparked.

## Suggested commit message (parent offers this; does not commit)

```
style: split toast.go import groups like neighbouring internal/ui files

```

## Self-review

1. **Spec coverage:** The open todo is the toast.go grouping. Task 1 implements it. Task 2 records it. `finishLoad` / `exif.go` are explicitly out of scope. CI `goimports` is parked.
2. **Placeholder scan:** No TBD/TODO in steps. Parked Task 3 refuses to invent Makefile text.
3. **Type consistency:** No new symbols. Import paths in Task 1 match the current file (`internal/completion`, `internal/ui/widgets`).
