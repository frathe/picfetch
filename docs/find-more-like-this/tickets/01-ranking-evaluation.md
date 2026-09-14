# 01: Evaluate real-image search quality

Ticket: FML-001
Status: qualitative acceptance recorded; quantitative judgments waived
Progress: evaluator and real measurements recorded; Ronin approved proceeding from his overall review. Per-item judgments and P@10 remain unmeasured.
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Produce a reproducible local report showing whether the existing image model finds useful content matches, with cold/warm timings and resource costs, before building the full interface.

**Blocked by:** None; real corpus/assets and relevance judgments are required to complete the evaluation gate.

**Specification:** AC-01 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] At least 20 varied, labeled content references produce exact cosine top-30 reports, including duplicates, hard negatives, unreadable inputs, weak matches and small collections. Missing result slots count as non-relevant in precision at 10. ([V01](../ticket-execution.md#v01))
- [ ] Record model/preprocessing and corpus identities, reproducible judgments, median content precision, and separate appearance results. Compare with the initial 0.6 median precision target; record a proceed/revise decision before FML-002. ([V01](../ticket-execution.md#v01))
- [x] Record cold preparation, first partial, complete preparation, warm query p50/p95, Favorite reuse, retained vectors and native RSS with machine details. Compare the initial 30-second usability and 200-ms warm p95/10,000-vector targets without promising universal performance. ([V01](../ticket-execution.md#v01))
- [ ] Use the existing pinned assets, decoding and isolation policy. Synthetic fixture tests prove report correctness; real images and recorded judgments prove usefulness. General-cache and final production-session measurements remain explicitly pending until FML-008. Technical checks passed; real usefulness judgments remain open. ([V01](../ticket-execution.md#v01))

**Demo / completion evidence:** A report can be regenerated from the same corpus and identifies both useful matches and measured limitations.

**Execution:** [FML-001 file map, contracts and verification](../ticket-execution.md#fml-001).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
