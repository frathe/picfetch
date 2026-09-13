# FML-009 — Explore positive and negative examples

Status: deferred; not part of the MVP.
Type: prototype/research.
Owner: T0 lead.
Depends on: FML-008 and separate selection of this follow-up.
Budget when activated: zero spawns; one experiment and up to two lead reviews.

## Goal

Refine a search with **More like these** and **Less like these** examples. Keep
the original proposal available without expanding the initial single-reference
implementation or adding a text model, training, or a preference database.

## Candidate files and contract

- Extend the local evaluation runner/report and its fixtures first.
- If the experiment succeeds, extend `internal/similarity` query/ranking types
  and `internal/ui/visualsearch` visit/history state, plus feature tests and
  translations. Update the plan with final contracts before implementation.

Compare a normalized positive centroid with a bounded negative-centroid penalty
against the single-reference baseline. Evaluation must determine weighting,
example limits, and useful behavior before these become product defaults.
Examples remain source identities within the same prepared session; exclude
example paths from matches and permit removing/resetting feedback. Contradictory
examples, zero/degenerate vectors, and a negative example also marked positive
need explicit handling. History must restore feedback sets with their results.

## Acceptance if activated

- A labeled real-image evaluation records whether feedback improves the chosen
  intent and identifies regressions; no benefit is claimed from math tests alone.
- The final chosen rule has deterministic fixture tests for positive/negative
  examples, removal/reset, invalid input, and exclusion.
- Existing single-reference results stay unchanged when no feedback is supplied.
- Repeated feedback uses the existing index; Back restores captured results
  and examples without inference or training.

Prospective verification after the experiment chooses a contract:

```sh
go test ./scripts/explorereval -run '^TestSearchFeedbackEvaluation' -count=1
go test -race ./internal/similarity ./internal/ui/visualsearch -run '^TestSearchFeedback' -count=1
```

Done when the experiment yields a recorded proceed/reject decision. A proceed
decision requires separately specified implementation work before activation;
this deferred ticket does not block FML-001 through FML-008.
