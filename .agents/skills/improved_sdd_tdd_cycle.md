# The Improved SDD → TDD Cycle

**A cost-aware working agreement for agentic development in picfetch.**

`AGENTS.md` says *what* the code must look like. This says *how the work gets
done*: who does which step, what may be delegated, what must never be, and what
counts as proof that a step is finished.

**Priority order — not negotiable, and not a tie:**

1. **Quality.** Correctness, then clarity, then the repo's conventions.
2. **Cost.** Of the ways that reach (1), take the cheapest.

Cost is an objective, never a constraint on (1). No check is skipped to save
tokens. The savings in this document all come from *not paying twice for the
same context* — never from looking less carefully.

---

## 1. Where the money actually goes

Cost is not proportional to how hard the problem is. It is proportional to **how
many times the problem gets understood**. Every one of these pays for the same
understanding more than once:

| Pattern | What you pay for twice | Fix |
|---|---|---|
| **Post-review delegation** | Lead reviews (builds full context), spawns a fixer (rebuilds it cold, weaker), re-reviews the fix (rebuilds again). One finding, three context builds. | Lead fixes what the Lead finds. §6, §8 |
| **Implementation-fidelity plans** | Lead writes the code into the plan (expensive output), then pays an agent to read and retype it (input + output again). | Write it *or* delegate it, never both. §11 |
| **Cold-start fan-out** | N agents each re-orient in the repo to do one small thing. | Delegate breadth, not depth. §7 |
| **Peer-tier delegation** | An agent as strong as the Lead redoes what the Lead already knows. | Delegate *down* or not at all. §4 |
| **Reading to answer a grep** | Whole files loaded to establish one fact. | Ask the shell first. §10 |
| **Regex work sent to a model** | A model reasons through what `sed` does for free. | Script it. §7, rule S |
| **Full-suite runs while iterating** | `go test -timeout 20m -race ./...` × every loop. | Targeted packages until the final gate. §10 |
| **Verifying by re-reading** | Re-derive from the diff what one command would have told you. | Evidence is command output. §10 |

The one delegation that *saves* Lead context rather than spending it is
read-only search fan-out (§4, Scout). Everything else is a trade: you pay tokens
to keep the Lead's window clean and to run work in parallel. Make that trade
deliberately.

---

## 2. The two laws

**Law 1 — Hot context stays home.**
Context the Lead already holds is the cheapest resource in the session; it has
been paid for. Re-deriving it in a fresh agent is the most expensive. So: the
moment you have understood something well enough to specify the fix, you are
also the cheapest thing in the room that can *apply* it. Apply it.

**Law 2 — Delegate only what survives amnesia.**
A task may leave the Lead only if a competent stranger with no memory of this
conversation could complete it from the prompt alone, and a single command
proves they did. If the task needs taste, needs the conversation, or is judged
by "does this look right" — it is the Lead's.

Everything below is these two laws made operational.

---

## 3. SDD → TDD, and why the seam matters

Spec-driven and test-driven are the same activity at two altitudes. The
improvement over ad-hoc practice is one rule at the seam:

> **Every acceptance criterion carries the command that proves it.
> A criterion with no command is not a criterion — it is a wish, and it goes
> back to the spec.**

This is what makes the rest affordable. When criteria are commands:

- The test file is derived from the criteria list 1:1 — no invention step.
- Delegation gets a pass/fail oracle, so a weaker model can be trusted with it.
- Review becomes *criteria vs. output*, not a re-reading of the whole diff.
- "Done" is falsifiable, so nobody has to argue about it.

A spec that cannot be written this way is not ready. That is a finding about the
spec, not a reason to start coding.

---

## 4. Roles and tiers

Route by **tier**, never by model name. The tier→model map lives in one place
(Appendix A) so it can change without rewriting plans.

| Tier | Who | Owns | Never touches |
|---|---|---|---|
| **T0 — Lead** | This session's model | Framing, spec, plan, architecture, **all review**, **all post-review fixes**, anything cross-cutting, the final gate | — |
| **T1 — Implementer** | Sonnet-class subagent (`go-expert` for Go) | One task, one package, tests already specified, interfaces already fixed | Spec decisions, cross-package design, review |
| **T2 — Mechanical** | Haiku-class subagent | Exact-spec transforms that need reading comprehension but no judgement | Anything whose success is a matter of opinion |
| **T3 — Scout** | `Explore` (read-only) | Bounded search fan-out; returns conclusions + `file:line`, never file dumps | Writing anything |

**The Lead is not a router.** It does the hardest thinking and most of the
typing. Delegation is an exception that must be argued for, task by task, in
writing (§7).

**T2's real niche** is mechanical work that needs *comprehension* — writing eight
table-driven cases from a list of behaviours, adding doc comments in the repo's
style, filling a translation catalogue. Mechanical work that a regex does is not
a T2 task; it is a `sed` command (§7, rule S).

---

## 5. Pick the route before doing anything

Most work is Thin. Choosing Deep by default is the single most expensive mistake
available.

| Route | When | Artifacts | Delegation | Reviews |
|---|---|---|---|---|
| **Thin** | ≤ 3 files, no new interface, no user-visible string, behaviour already understood | None — commit message is the record | None | One, at the end |
| **Standard** | One feature or bugfix, ≤ 8 files, ≤ 2 packages | `plans/YYYY-MM-DD-slug.md`, spec + task list | 0–2 tasks | Gate per task + final |
| **Deep** | New subsystem, cross-package refactor, platform-specific behaviour, anything shipping to Windows | Full plan with file map, task graph, routing table, cost ledger | Per plan | Gate per task + final |

**Promote, don't pre-empt.** Start Thin. Promote to Standard the moment you find
a second package involved or a decision you cannot make alone. A Thin run that
gets promoted has lost nothing; a Deep run that should have been Thin has burned
a plan document nobody needed.

Today's five-item Windows fix (§13) was correctly **Standard**, and cost roughly
one Deep plan's worth of writing less than it would have under the old habit.

---

## 6. The cycle

| # | Phase | Owner | Exit gate |
|---|---|---|---|
| 0 | Frame | T0 | Route chosen; deliverable named in one sentence |
| 1 | Spec | T0 (+ user) | Every criterion has a command; non-goals written |
| 2 | Recon | T3 or shell | Every open question answered with `file:line` |
| 3 | Plan | T0 | Each task has files, test, command, owner, budget |
| 4 | Red | per plan | Test fails **for the stated reason** |
| 5 | Green | per plan | Test passes; nothing else broke |
| 6 | Review | **T0 only** | Criteria checked against command output |
| 7 | Fix | **T0 only, inline** | Findings closed or explicitly deferred |
| 8 | Land | T0 | Docs, `todos.md`, memory, ledger |

### 0 — Frame

Restate the request in one sentence naming the deliverable. Choose the route.
Name what is explicitly *not* in scope. If two readings of the request lead to
materially different work, ask now — a question here costs one message; the same
question discovered at phase 6 costs the whole cycle.

### 1 — Spec

Use `superpowers:brainstorming` for anything creative or ambiguous. Produce:

- **Problem** — the observed behaviour, with evidence (log line, screenshot, error).
- **Decisions** — resolved questions, in a table, marked *do not relitigate*.
- **Acceptance criteria** — numbered, each with its verification command.
- **Non-goals** — what this deliberately does not do.
- **The honest limit** — what will still be broken afterwards, and why that is acceptable.

A criterion looks like this, and nothing weaker counts:

```
AC3  No Unicode arrow reaches the renderer.
     go test ./internal/ui/help/ -run TestManualHasNoUnicodeArrows &&
     go test ./internal/ui/ -run TestTranslationsHaveNoUnicodeArrows
```

### 2 — Recon

Answer every open question the plan depends on, cheapest tool first:

1. **Shell** — `grep -rn`, `git log --oneline --grep`, `go doc`. Most questions
   die here for a few hundred tokens. *Always try this first.*
2. **Targeted read** — the specific function, via `sed -n 'A,Bp'`, not the file.
3. **T3 Scout** — only when the answer needs a sweep across many files or naming
   conventions and you want the conclusion without the dumps. Give it a bounded
   question and demand `file:line` back.

Never open a Scout for something a grep answers. Never read a whole file to
learn one signature.

### 3 — Plan

Standard and Deep only. Per task, exactly this much and no more:

```
### Task N — <name>
Owner:   T0 inline | T1 go-expert | T2 mechanical
Files:   create / modify / test
Depends: task numbers
Contract: the exact identifiers and signatures later tasks rely on
Test:    the behaviour the test must pin (prose or assertion, not a full body)
Verify:  the one command that proves the task is done
Budget:  ≤ N spawns · ≤ M review rounds · full suite: yes/no
```

**Contracts, not implementations.** The plan pins names, signatures and
behaviour so parallel tasks agree. It does not contain the code — see §11, rule W.

Draw the task graph. Mark which tasks are independent. Independent does not mean
"spawn them all" (§7).

### 4 — Red

Write the test first, always — `superpowers:test-driven-development`. Then **run
it and read the failure**. A test that fails for the wrong reason (typo, missing
import, wrong package) is not a red test; it is a broken test that will go green
for the wrong reason too.

Design-bearing tests are T0's. Table expansion over an already-designed test is
T2's.

### 5 — Green

Minimal implementation that passes. No speculative generality, no adjacent
cleanup — that is a separate task with its own criterion.

### 6 — Review — T0, always

**Never delegated.** Not to a peer model, not to a "reviewer" agent, not to a
second opinion. The Lead wrote or accepted the spec; the Lead is the only party
that can say whether the spec was met.

Order matters — mechanical first, because it is nearly free and catches most of it:

1. `gofmt -l .` — empty.
2. `go vet ./...` — clean.
3. Every AC command from the spec — run, output read.
4. Each guard test **negatively verified**: break the thing on purpose, confirm
   the test fails, restore. A guard never seen to fail is not known to guard.
5. `git diff --stat` — does the change's shape match the plan? Unexpected files
   are the cheapest bug signal you will ever get.
6. *Then* read the diff, against the criteria — not line by line for its own sake.

For delegated tasks, add: **verify the agent's report mechanically, never take
it at face value.** Agents report success they did not achieve. Check the claim
against a command, every time.

### 7 — Fix — T0, inline, always

**The fix never leaves the Lead.** This is the rule with the largest single
effect on cost, and it is the one the priority order in §1 and Law 1 exist to
justify.

At the moment a finding exists, the Lead holds the file, the diff, the test, the
spec and the reason — the most expensive state in the session, already paid for.
Spawning throws it away, rebuilds it cold in a weaker model, and forces a
re-review to check the rebuild. One finding becomes three context builds and two
review passes. Fixing inline is one edit and one command.

**The one exception**, and it is not "delegate my review": if a review produces a
*large batch of independent mechanical edits* (say, thirty call sites after a
rename), that batch may be written up as a **new task** with its own spec, own
verification command, and own trip through the delegation gate (§7). It is a new
delegation that happens to have been discovered during review — not a fixer
agent, and it is only worth it when the batch is too large to script (rule S).

Re-run the AC commands after fixing. A fix is not done because it was applied; it
is done because the command says so.

### 8 — Land

- Update `todos.md` — and correct anything in it the work proved wrong.
- Update `AGENTS.md` if a *convention* was established (not if a bug was fixed).
- Write a memory only for what the repo does not already record: the surprising
  cause, the rule learned, the thing that will otherwise regress again.
- Append the cost ledger (§11).
- Move the plan to `finished_refactorings/` when the branch is accepted.

---

## 7. The delegation gate

A task leaves the Lead only if **every** answer is yes. Write the answers in the
plan — an unwritten gate is a gate that was not applied.

| # | Question | If no |
|---|---|---|
| **G1** | **Amnesia.** Can I specify it in ≤ 25 lines so a stranger with no memory of this conversation could do it? | Inline |
| **G2** | **Oracle.** Is there one command whose output proves success, with no judgement call? | Inline |
| **G3** | **Blast radius.** ≤ 3 files, and no file another live agent touches? | Inline, or re-cut the task |
| **G4** | **Cold-start economics.** Is the context it needs genuinely smaller than the context I would have to hand it? | Inline — if the prompt carries half the plan, spawning saves nothing |
| **G5** | **Hot context.** Is this something I have *not* just built the context to do myself? | **Inline. Always.** This is Law 1, and it is what forbids post-review delegation. |

Then two tie-breakers that override a passing gate:

- **Rule S — Script before spawn.** If `sed`, `python3`, `gofmt` or a `go`
  generator can do it deterministically, do that. Today's 116-arrow substitution
  across two manuals passed G1–G5 cleanly and was still a five-line script:
  cheaper, exactly repeatable, and reviewable as a diff. A model is for tasks
  needing comprehension, not for regex.
- **Rule W — Write it or delegate it, never both.** If the plan already contains
  the implementation, paste it yourself. Paying T0 output tokens to write the
  code and then T1 tokens to retype it is the most expensive way to produce a
  line of Go that exists.

**Parallelism.** Independent tasks may run concurrently only when they share no
file and no contract still in flux. Two agents editing one package is not
parallelism; it is a merge conflict with a bill attached. Cap: **two concurrent
subagents**, and only when the Lead has real work to do meanwhile.

---

## 8. Never delegated

- **Review** of any kind (§6).
- **Fixes to review findings** (§7) — modulo the batch exception.
- **Spec and acceptance criteria.**
- **Architecture**: package boundaries, interface shape, dependency choices.
- **Anything touching a user-visible string**, because it must land in every
  `translations/*.json` and obey the conventions in `AGENTS.md`.
- **Platform-specific behaviour that cannot be tested here** — Windows apply
  paths, macOS packaging. Untestable *and* delegated is two blind spots stacked.
- **The final gate** (§10).
- **Anything you would not be willing to defend to the user line by line.**

---

## 9. Prompt templates

A vague prompt is the most expensive thing in this document: the agent explores
to fill the gap, and you pay for its confusion by the token.

**T1 — Implementer**

```
Task: <one sentence>

Context you need (do not go looking for more):
  - <file:line> — <what it is>
  - Contract you must implement exactly: <signatures>

Constraints:
  - Repo conventions: AGENTS.md §Project Conventions. Read it; it is short.
  - Do not touch: <files>
  - Do not add dependencies. Do not refactor anything not named here.

Test first: <the behaviour to pin>. Run it, confirm it fails for that reason,
then implement.

Done when this prints clean:  <command>

Report: the command's actual output, the files you changed, and anything you
could not do. Do not report success you have not seen a command confirm.
```

**T2 — Mechanical**

```
Transform: <exact, unambiguous rule>
Files: <explicit list — no globs, no discovery>
Do not change anything else in these files.
Done when: <command> prints <exact expected output>
Report: the command output verbatim.
```

**T3 — Scout**

```
Question: <one bounded question>
Breadth: medium | very thorough
Return: conclusions with file:line. No file contents, no full listings,
no recommendations. If the answer is not in the repo, say so.
```

---

## 10. Verification and the final gate

**Evidence is command output.** Not the diff, not an agent's summary, not
"should work". State only what a command has confirmed; where you did not verify,
say so plainly.

| Claim | Acceptable evidence |
|---|---|
| "Tests pass" | The `go test` line, with the package name |
| "Nothing else broke" | Targeted packages green, then the final gate |
| "The guard works" | The guard seen **failing** on a deliberate violation, then green |
| "The subagent did it" | Your own command, not its report |
| "It renders correctly" | A screenshot, or a measurement — fonts and layout are not deducible |
| "It works on Windows" | A Windows run, or an explicit statement that it is unverified |

**While iterating:** targeted packages only — `go test ./internal/update/`,
`go test ./internal/ui/ -run TestUpdateFailure`. The full `-race` suite is 20
minutes; running it every loop is pure cost with no new information.

**Final gate, once, before handoff** (matches CI — `AGENTS.md §Build and Verification`):

```
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

Cross-platform work adds `GOOS=windows GOARCH=amd64 go vet ./internal/...`.

---

## 11. Budgets and the cost ledger

Every Standard/Deep plan declares a budget per task and records the actual. This
exists so cost is *visible* rather than discovered on an invoice.

```
| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes           |
|------|------------------------|---------------|------------|-----------------|
| T1   | 1 / 1                  | 1             | no         |                 |
| T2   | 1 / 0                  | 1             | no         | scripted (S)    |
| T3   | 0 / 0                  | 2             | no         | hot context     |
| gate | —                      | —             | yes        |                 |
```

Rules of thumb, to be overridden with a written reason:

- **Thin route:** 0 spawns.
- **Standard:** ≤ 2 spawns total.
- **Deep:** ≤ 1 spawn per independent task, ≤ 2 concurrent.
- **Full suite:** once, at the final gate.
- **Scouts:** ≤ 2 per phase, and only after the shell has failed to answer.

Going over budget is allowed. Going over silently is not — record it and say why.

---

## 12. Stop rules

- **Two strikes.** A delegated task that fails its verification twice comes back
  inline. Never a third spawn — by then the Lead has read two failed attempts and
  holds more context than any fresh agent will.
- **No spawn to answer a question one grep answers.**
- **No spawn while a spec question is open.** Delegating an ambiguity multiplies it.
- **No second agent on a file another agent holds.**
- **Ratchet the guard.** When a bug recurs (it has, more than once), the fix is
  not complete until a test makes the *class* of bug impossible, and that test
  has been seen to fail. Then write it into `AGENTS.md`.
- **When the plan and the code disagree, the code wins.** Stop, say so, correct
  the plan. Never implement against a claim you have just watched fail.
- **Report faithfully.** If a step was skipped, say it was skipped. If a test
  fails, show the output. Unverified is a valid state; pretending is not.

---

## 13. Worked example — the Windows update session (2026-08-30)

Five reported defects: replacement glyphs in a dialog, a dialog too small for
German, a wrong download link, the same glyph bug in both manuals, and a `.old`
file never cleaned up.

| Phase | What happened | Cost note |
|---|---|---|
| 0 Frame | Route: **Standard**. Deliverable named; no plan document needed. | A Deep plan here would have been a document nobody read. |
| 1 Spec | The user's message *was* the spec; five criteria, each with a command. | Zero cost. Recognise this when it happens. |
| 2 Recon | Six greps + `git log --grep` + two font-cmap dumps via `python3`. **No Scout.** | The decisive evidence — NotoSans has no arrow glyphs at all — came from parsing two `.ttf` cmaps directly. No model could have known it; no amount of reasoning would have replaced it. |
| 3 Plan | Held in the todo list, not a document. | Standard route earns this. |
| 4–5 Red/Green | Guards written first, then the fixes. Two new guard tests. | — |
| 6 Review | Both guards **negatively verified** — arrow injected, both failed, restored. | The step that turns a guard into a guarantee. |
| 7 Fix | All five inline. Zero spawns. | Law 1, applied. |
| 8 Land | `todos.md`, `AGENTS.md` convention, one memory. | The memory is what stops the sixth recurrence. |

**The delegation that was correctly declined:** the 116-arrow manual
substitution passed G1–G5 — exact spec, `grep -c` oracle, two files, no shared
state. Rule S declined it anyway, and a five-line Python script did it in one
call. **A task cheap enough to script is cheaper to script than to delegate.**

**What the old habit would have cost:** a Deep plan document for a five-item
bugfix, a Scout to find the arrows, an agent per fix, and a re-review of each
agent's work. Roughly five cold starts and three extra review passes for the
same diff.

---

## 14. Lead checklist

Copy into the todo list at the start of a Standard or Deep run.

```
[ ] Frame     — deliverable in one sentence; route chosen; non-goals named
[ ] Spec      — every AC has a command; decisions table; honest limit written
[ ] Recon     — shell first; Scout only for genuine fan-out; questions closed
[ ] Plan      — per task: files, contract, test, verify command, owner, budget
[ ] Gate      — G1–G5 written per delegated task; S and W applied
[ ] Red       — test run, failure read, fails for the stated reason
[ ] Green     — minimal; no adjacent cleanup
[ ] Review    — T0 only: fmt, vet, AC commands, guards negatively verified, diff
[ ] Fix       — T0 inline; AC commands re-run after
[ ] Gate      — fmt-check, vet, build, full -race suite, once
[ ] Land      — todos.md, AGENTS.md if a convention, memory, cost ledger
```

---

## Appendix A — Tier map

| Tier | Model | Agent | Notes |
|---|---|---|---|
| T0 Lead | Opus 5 | — (this session) | Never spawned as a subagent |
| T1 Implementer | Sonnet 5 | `go-expert`, `refactor-planner` | Both are defined `model: sonnet` in `.claude/agents/` |
| T2 Mechanical | Haiku 4.5 | `general-purpose` with an exact spec | Comprehension, not regex (rule S) |
| T3 Scout | per agent definition | `Explore` | Read-only; conclusions + `file:line` only |
| — | Fable 5 | — | **Unrouted.** No established cost/capability profile in this repo; do not default to it without measuring one. |

Older plans in `plans/` and `finished_refactorings/` route to `gpt-5.6-*` names
from a different harness. Read those as **tiers**, not as literal models.

## Appendix B — Skills by phase

| Phase | Skill |
|---|---|
| 1 Spec | `superpowers:brainstorming` |
| 3 Plan | `superpowers:writing-plans` (Deep only) |
| 4 Red / 5 Green | `superpowers:test-driven-development` |
| any | `superpowers:systematic-debugging` — before proposing any fix to a bug |
| 6 Review | `superpowers:requesting-code-review` (Deep only; the Lead still reviews) |
| 10 Gate | `superpowers:verification-before-completion` |
| 8 Land | `superpowers:finishing-a-development-branch` |

The **Thin** route uses `test-driven-development` and
`verification-before-completion` only. Loading a planning skill for a two-file
fix is itself a cost.
