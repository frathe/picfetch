# goimports CI Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** Parent implements (user unparked Task 3). Do not commit (`AGENTS.md`).

**Goal:** Make `make fmt`, `make verify`, and GitHub CI enforce `goimports -local github.com/frathe/picfetch`, so a toast.go-style merge of fyne and local imports fails the gate.

**Architecture:** Pin `golang.org/x/tools/cmd/goimports` as a Go `tool` in `go.mod` (`go tool goimports`). One Makefile `fmt-check` target is the format gate; `verify` and CI both call it. `make fmt` writes with the same `-local` prefix. No new Go tests; proof is a red/green check against `toast.go`.

**Tech Stack:** Go 1.26 `tool` directive, `goimports` v0.48.0 (the version already verified against this tree: `-l .` is empty after the toast.go split).

## Global Constraints

- `-local` prefix is exactly `github.com/frathe/picfetch`.
- Do not commit.
- Do not decompose `finishLoad` or split `exif.go`.
- Do not add an import-grouping unit test.
- Do not mass-rewrite source: current tree must already pass `goimports -local … -l .`.
- `go tool goimports` (pinned in `go.mod`), not `go install @latest` in CI.
- Keep `install-tools` for fyne/fyne-cross/govulncheck only; goimports comes from `go tool`.
- Docs that name the CI format check (README, CONTRIBUTING, PR template, AGENTS.md, todos.md) must match the new gate.

---

### Task 1: Pin goimports and wire Makefile + CI

**Files:**
- Modify: `go.mod`, `go.sum` (`go get -tool golang.org/x/tools/cmd/goimports@v0.48.0`)
- Modify: `Makefile` (`fmt`, new `fmt-check`, `verify` help text, `.PHONY`)
- Modify: `.github/workflows/ci.yml` (Check formatting step)

- [ ] **Step 1:** Add the tool; confirm `go tool goimports -local github.com/frathe/picfetch -l .` prints nothing.
- [ ] **Step 2:** `make fmt` → `go tool goimports -local github.com/frathe/picfetch -w .`
- [ ] **Step 3:** `fmt-check` fails when `-l` is non-empty, message tells the user to run `make fmt`. `verify` runs `fmt-check` then vet/build/race tests.
- [ ] **Step 4:** CI Check formatting runs `make fmt-check` (same gate as verify; no duplicated shell).
- [ ] **Step 5:** Red/green: drop the toast.go blank line, `make fmt-check` fails listing `internal/ui/toast.go`; restore; `make fmt-check` passes.

### Task 2: Docs that describe the gate

**Files:**
- Modify: `README.md` (fmt/verify table)
- Modify: `.github/CONTRIBUTING.md`
- Modify: `.github/pull_request_template.md`
- Modify: `AGENTS.md` (Match CI line)
- Modify: `todos.md` (Done bullet)

- [ ] Name `goimports -local github.com/frathe/picfetch` (or `make fmt` / `make fmt-check`) wherever those files currently say `gofmt` is the CI format check.
