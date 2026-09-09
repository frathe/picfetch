# Maintainability ticket breakdown

Status: tickets 01–30 complete; 31/32 remain accepted dependency watches
Date: 2026-09-06

Prepared from the [specification](spec.md) and its [phased plan](../2026-09-06-maintainability-plan.md).
There are **32 published tickets: 27 required, 3 conditional, and 2 accepted-watch reminders**.
All 24 MA items are accounted for. The parent specification and historical audit evidence are preserved.

The user requested `/implement sdd tdd` on the parent specification. The prepared bodies are published in `issues/` with their activation conditions preserved. Tickets 01–26 are resolved with evidence in their issue files, including the ticket 15 manual-test correction and the completed Windows/Store native guards. Ticket 26 has macOS Retina and user-operated Windows ARM64 renderer evidence, with independently recorded 100% monitor scaling and DPI 96. Ticket 27 has reviewed inputs, all required artifact builds, native macOS, both Windows ARM64 and both Linux package evidence (Linux amd64 emulated); both x64 attempts remain failed: a matching runtime resolves context creation but crashes during drawing in the ARM64 VM. On 2026-09-09 the user reported successful Windows 11 ARM and x64 testing and accepted the remaining detailed checks as edge cases. Ticket 27 / MA-020 is closed by that acceptance, with remaining Windows SDK/WACK evidence waived for this audit; no WACK pass is claimed. See the [acceptance record](windows-test-todo.md). The failed VM attempts remain historical evidence.

Tickets 28–30 are now resolved with measured preview capacity and the tested
command matrix. Ticket 24's user-reported loss of progressive updates is fixed
and reverified. The shared Phase 6 common gate passes all 667 root UI tests and
the complete race suite. Ticket 27 is closed under the acceptance above;
31/32 remain accepted watches awaiting their explicit dependency-upgrade triggers.
The separate MA-025 decoded-map retention follow-up remains in the open backlog.

## Published breakdown

1. **[Reject malformed TIFF spans safely](issues/01-safe-tiff-spans.md)** — MA-001; required.
   **Blocked by:** None. **Delivers:** Open malformed EXIF-bearing images and RAW previews without a panic while retaining valid metadata and orientation.

2. **[Bound mosaic preparation without changing composition](issues/02-bounded-mosaic-preparation.md)** — MA-002; required.
   **Blocked by:** None. **Delivers:** Generate a mosaic from extremely wide or tall source images under an explicit scratch-memory budget, preserving placement and edge quality.

3. **[Reconcile confirmed deletions by file identity](issues/03-deletion-target-identity.md)** — MA-004; required.
   **Blocked by:** None. **Delivers:** Delete the confirmed files and reconcile only successful removals with the current Grid result even when its ordering or generation changes.

4. **[Construct complete shared image-cache records](issues/04-complete-image-cache-records.md)** — MA-006; required.
   **Blocked by:** None. **Delivers:** Show consistent size, EXIF information and canonical pixels regardless of whether foreground loading, preloading or comparison first cached an image.

5. **[Save truthful rotated JPEG dimensions](issues/05-saved-jpeg-dimensions.md)** — MA-007; required.
   **Blocked by:** None. **Delivers:** Save Changes writes JPEG dimension metadata that agrees with the encoded frame after rotation and orientation normalization.

6. **[Preserve exact native chooser paths](issues/06-exact-native-chooser-paths.md)** — MA-005; required.
   **Blocked by:** None. **Delivers:** Open exactly the selected files and save to exactly the confirmed destination, preserving every path boundary supported by each native backend.

7. **[Preserve clipboard temporary-file errors](issues/07-clipboard-temp-errors.md)** — MA-011; required.
   **Blocked by:** None. **Delivers:** Report the original temporary PNG write or close error rather than apparent success after cleanup.

8. **[Decode Windows copied-file lists explicitly as UTF-8](issues/08-windows-file-list-encoding.md)** — MA-012; required.
   **Blocked by:** None. **Delivers:** Copy Windows file references containing accented, CJK and emoji names without changing their paths at the PowerShell boundary.

9. **[Respect ordinary XDG Trash configuration](issues/09-trash-xdg-configuration.md)** — MA-013; required.
   **Blocked by:** None. **Delivers:** Honor the user's normal Trash data directory while retaining the identified sandbox-redirection workaround.

10. **[Invalidate favorite previews on subsecond edits](issues/10-preview-subsecond-identity.md)** — MA-018; required.
   **Blocked by:** None. **Delivers:** Refresh a favorite preview after a same-size source edit whose available modification timestamp differs within one second.

11. **[Handle declining RSS in the optional HEIC check](issues/11-safe-rss-growth.md)** — MA-019; required.
   **Blocked by:** None. **Delivers:** Treat declining or unchanged RSS as no positive growth instead of unsigned-underflow evidence of a leak.

12. **[Pace animation after queued frame application](issues/12-queued-animation-pacing.md)** — MA-003; required.
   **Blocked by:** None. **Delivers:** Play heterogeneous animation frame delays correctly when UI dispatch is delayed, and stop obsolete playback observably.

13. **[Reject late picture-frame advances](issues/13-queued-picture-frame-advances.md)** — MA-003; required.
   **Blocked by:** None. **Delivers:** Leaving picture-frame mode prevents queued advances from changing the closed or subsequent session.

14. **[Keep chooser admission and results on UI](issues/14-chooser-ui-admission.md)** — MA-003; required.
   **Blocked by:** None. **Delivers:** Native file choosing remains background work while admission checks, comparison refusal and result handling run on UI.

15. **[Cancel capture-date sorting inside source reads](issues/15-capture-sort-cancellation.md)** — MA-010; required.
   **Blocked by:** None. **Delivers:** Cancel a capture-date sort during an ancillary source read so obsolete work releases the sort request promptly where the reader supports it.

16. **[Cancel favorite thumbnail reads through imaging](issues/16-favorite-thumbnail-cancellation.md)** — MA-010; required.
   **Blocked by:** None. **Delivers:** Cancel a favorite preview pass through thumbnail read/probe/decode boundaries while retaining previews the pass never visited.

17. **[Cancel grid thumbnails and native-size backfill](issues/17-grid-read-cancellation.md)** — MA-010; required.
   **Blocked by:** 16. **Delivers:** Superseding or closing grid work cancels obsolete thumbnail reads, hash work and native-size backfill without admitting cancelled decode-slot waiters.

18. **[Admit thumbnail facts only to their source generation](issues/18-generation-safe-facts.md)** — MA-009; required.
   **Blocked by:** None. **Delivers:** Keep old hashes, failure markers and native dimensions out of a replacement/reset file set, including same-URI replacement.

19. **[Bound aggregate map requests and failure state](issues/19-bounded-map-lifetime.md)** — MA-015; required.
   **Blocked by:** None. **Delivers:** Map navigation and window closure keep aggregate fetch concurrency and failure metadata bounded while retaining useful deduplication and cache behavior.

20. **[Make position-poller shutdown observable](issues/20-position-poller-completion.md)** — MA-016; required.
   **Blocked by:** None. **Delivers:** Stopping window-position polling prevents queued native reads from updating a closed target and exposes actual worker completion.

21. **[Move whole-image clipboard encoding off UI](issues/21-background-clipboard-encoding.md)** — MA-014; required.
   **Blocked by:** None. **Delivers:** Copy the captured displayed image while input and redraw remain responsive throughout PNG encoding and clipboard dispatch.

22. **[Load EXIF panel data without blocking UI](issues/22-background-exif-refresh.md)** — MA-014; required.
   **Blocked by:** None. **Delivers:** Open or refresh the EXIF panel without blocking input while a slow source is read, and show only metadata for the current displayed request.

23. **[Run original-file mutations in serialized background work](issues/23-background-original-file-mutations.md)** — MA-014; required.
   **Blocked by:** None. **Delivers:** Save Changes and metadata removal leave the UI responsive and cannot overlap unsafely on the same original source.

24. **[Reuse cancellable duplicate-group snapshots](issues/24-reusable-duplicate-groups.md)** — MA-008; required.
   **Blocked by:** 18. **Delivers:** Searching a warm Grid result reuses unchanged duplicate groups, while changed grouping computes away from UI and cannot install obsolete results.

25. **[Execute omitted native and Store regression guards](issues/25-native-and-store-ci-guards.md)** — MA-017; required.
   **Blocked by:** None. **Delivers:** Release validation actually executes the existing Windows, macOS and Store-specific guards with visible evidence of test selection.

26. **[Maintain and exercise production comparison GL smoke checks](issues/26-native-comparison-gl-smoke.md)** — MA-017; required.
   **Blocked by:** None. **Delivers:** Verify comparison's actual shader output and interactions in a documented, repeatable native desktop smoke procedure.

27. **[Pin and validate reviewed packaging inputs](issues/27-reviewed-packaging-inputs.md)** — MA-020; required.
   **Blocked by:** None. **Delivers:** Local, release and Store builds use reviewed packaging tool/image inputs and produce inspectable, smoke-tested artifacts without publication.

28. **[Measure foreground contention during favorite prewarming](issues/28-measure-preview-contention.md)** — MA-021; conditional.
   **Blocked by:** 15, 16, 17. **Delivers:** Determine whether current favorite prewarming materially delays visible image work using a repeatable cold-favorite workload.

29. **[Protect foreground capacity if prewarm measurements justify it](issues/29-foreground-preview-capacity.md)** — MA-021; conditional.
   **Blocked by:** 28. **Delivers:** Apply the smallest measured prewarm share/yield policy that preserves interactive capacity and still completes disk previews when idle.

30. **[Consolidate repeated command admission when routes are touched](issues/30-command-admission-policy.md)** — MA-022; conditional.
   **Blocked by:** 12, 13, 14. **Delivers:** Preserve command behavior across menus, keyboard and direct entry while simplifying repeated admission decisions in touched routes.

31. **[Reassess the HEIC fork at its next dependency update](issues/31-heic-fork-retirement-watch.md)** — MA-023; watch.
   **Blocked by:** 11. **Delivers:** At a future authorized HEIC update, determine whether an official release can replace the fork while preserving its leak mitigation.

32. **[Measure verifier footprint at its next major upgrade](issues/32-verifier-footprint-watch.md)** — MA-024; watch.
   **Blocked by:** None. **Delivers:** At a future authorized major verifier upgrade, measure dependency cost and retain the update trust/provenance contract through any justified reduction.

## Why these boundaries and edges

Each required ticket includes the owning behavior, its callers and meaningful tests. The broad audit work packages are split only where a smaller independently verifiable user action can land green:

- MA-003 becomes animation pacing (12), picture-frame advancement (13), and chooser admission (14). Their state owners are independent.
- MA-010 becomes capture-date sort cancellation (15), favorite thumbnail cancellation (16), and grid/native-size cancellation (17). Ticket 16 adds context-bearing thumbnail APIs and migrates a complete favorite-preview path while retaining compatibility for grid. Ticket 17 depends on that new API and migrates the grid. No separate unused abstraction or arbitrary API-removal ticket is needed.
- MA-014 becomes clipboard encoding (21), EXIF reads (22), and original-file mutations (23). Save Changes and metadata removal remain together: both can rewrite the same resolved source, and moving either to a worker introduces the need to serialize the entire read-transform-write transaction.
- MA-017 becomes native/Store CI guards (25) and production-GL smoke evidence (26). Both are required to close that finding.
- The unrelated entries in plan groups G/H have separate tickets. Plan F's chooser and Windows clipboard transports are also separate.
- MA-021 has a measurement ticket (28) before a conditional policy ticket (29); a decision to retain the current policy leaves 29 explicitly unactivated.
- MA-023/024 have future-trigger reminders (31/32), retaining their accepted-watch scope.

Hard edges are **17 <- 16**, **24 <- 18**, **28 <- 15/16/17**, **29 <- 28**, **30 <- 12/13/14**, and **31 <- 11**.
They represent a new consumed thumbnail API, generation-safe fact inputs, the spec's cancellation-before-measurement requirement, evidence before a scheduling change, the plan's I-before-Q ordering, and corrected arithmetic before a future HEIC RSS validation, respectively.

Useful queue patterns do not force map, poller or unrelated action tickets to wait for a new framework. Saved-dimension correction need not wait for TIFF reader hardening. Native coverage can start early; each platform-affecting implementation still records its own native evidence rather than treating an earlier baseline smoke as proof of a later change.

Shared file ownership affects scheduling, not the dependency graph. Do not have concurrent implementers edit the same owning module just because two tickets have no logical blocker. Follow the repository delegation limits and lead-owned review.

## Activation and frontier

Start with **01 and 02**. Other required tickets with no blockers can be worked independently when ownership permits.
After publication, a ticket is eligible only when every numbered blocker is complete **and** its activation condition is satisfied.

- Tickets 28-30 are conditional and do not block required audit work. Ticket 28 starts only if contention measurement is selected; 29 also requires measured need and an accepted bounded policy; 30 starts only during selected work on command routes.
- Tickets 31-32 are accepted watches, outside the ordinary frontier. They activate only under a separately authorized HEIC update or major verifier update. This ticketing request creates no upgrade, standing monitor or external notification.
- AC25 is a gate on each mergeable implementation ticket, not a final testing-only ticket that lets earlier work land without verification.
- Native Windows/macOS/GL, Linux RSS and packaging/WACK requirements remain explicit. `ready-for-agent` will describe the published specification state; it will not waive a native environment or activation condition.

## Coverage

| Spec finding / acceptance | Tickets | Scope |
| --- | --- | --- |
| MA-001 / AC01 | [01: Reject malformed TIFF spans safely](issues/01-safe-tiff-spans.md) | required |
| MA-002 / AC02 | [02: Bound mosaic preparation without changing composition](issues/02-bounded-mosaic-preparation.md) | required |
| MA-003 / AC03 | [12: Pace animation after queued frame application](issues/12-queued-animation-pacing.md); [13: Reject late picture-frame advances](issues/13-queued-picture-frame-advances.md); [14: Keep chooser admission and results on UI](issues/14-chooser-ui-admission.md) | required |
| MA-004 / AC04 | [03: Reconcile confirmed deletions by file identity](issues/03-deletion-target-identity.md) | required |
| MA-005 / AC05 | [06: Preserve exact native chooser paths](issues/06-exact-native-chooser-paths.md) | required |
| MA-006 / AC06 | [04: Construct complete shared image-cache records](issues/04-complete-image-cache-records.md) | required |
| MA-007 / AC07 | [05: Save truthful rotated JPEG dimensions](issues/05-saved-jpeg-dimensions.md) | required |
| MA-008 / AC08 | [24: Reuse cancellable duplicate-group snapshots](issues/24-reusable-duplicate-groups.md) | required |
| MA-009 / AC09 | [18: Admit thumbnail facts only to their source generation](issues/18-generation-safe-facts.md) | required |
| MA-010 / AC10 | [15: Cancel capture-date sorting inside source reads](issues/15-capture-sort-cancellation.md); [16: Cancel favorite thumbnail reads through imaging](issues/16-favorite-thumbnail-cancellation.md); [17: Cancel grid thumbnails and native-size backfill](issues/17-grid-read-cancellation.md) | required |
| MA-011 / AC11 | [07: Preserve clipboard temporary-file errors](issues/07-clipboard-temp-errors.md) | required |
| MA-012 / AC12 | [08: Decode Windows copied-file lists explicitly as UTF-8](issues/08-windows-file-list-encoding.md) | required |
| MA-013 / AC13 | [09: Respect ordinary XDG Trash configuration](issues/09-trash-xdg-configuration.md) | required |
| MA-014 / AC14 | [21: Move whole-image clipboard encoding off UI](issues/21-background-clipboard-encoding.md); [22: Load EXIF panel data without blocking UI](issues/22-background-exif-refresh.md); [23: Run original-file mutations in serialized background work](issues/23-background-original-file-mutations.md) | required |
| MA-015 / AC15 | [19: Bound aggregate map requests and failure state](issues/19-bounded-map-lifetime.md) | required |
| MA-016 / AC16 | [20: Make position-poller shutdown observable](issues/20-position-poller-completion.md) | required |
| MA-017 / AC17 | [25: Execute omitted native and Store regression guards](issues/25-native-and-store-ci-guards.md); [26: Maintain and exercise production comparison GL smoke checks](issues/26-native-comparison-gl-smoke.md) | required |
| MA-018 / AC18 | [10: Invalidate favorite previews on subsecond edits](issues/10-preview-subsecond-identity.md) | required |
| MA-019 / AC19 | [11: Handle declining RSS in the optional HEIC check](issues/11-safe-rss-growth.md) | required |
| MA-020 / AC20 | [27: Pin and validate reviewed packaging inputs](issues/27-reviewed-packaging-inputs.md) | required |
| MA-021 / AC21 | [28: Measure foreground contention during favorite prewarming](issues/28-measure-preview-contention.md); [29: Protect foreground capacity if prewarm measurements justify it](issues/29-foreground-preview-capacity.md) | conditional |
| MA-022 / AC22 | [30: Consolidate repeated command admission when routes are touched](issues/30-command-admission-policy.md) | conditional |
| MA-023 / AC23 | [31: Reassess the HEIC fork at its next dependency update](issues/31-heic-fork-retirement-watch.md) | watch |
| MA-024 / AC24 | [32: Measure verifier footprint at its next major upgrade](issues/32-verifier-footprint-watch.md) | watch |

## Draft validation

Passed document checks: 32 sequential unique ticket numbers; 10 existing, non-self blocking references ordered before dependents; an acyclic graph; all 24 MA mappings; verification commands and common completion gates in every draft; valid local links and whitespace. The parent spec hash is unchanged, and no final issue files have been published. These are documentation checks, not execution of the future regressions.

Production source, dependencies and verification inputs have no diff from the preceding spec publication, whose `make verify` passed. Reuse that code-baseline result for this draft-only step; do not label it as a new run or as proof of any MA fix.

Drafting budget: one read-only call-path scout, lead-owned synthesis/review, no implementation delegation. The scout inspected independent cancellation/mutation call paths while the lead prepared the breakdown; reported symbols were checked by repository search. G1-G5 were satisfied for that bounded read-only task; no edit, spec decision or review was delegated.

## Publication

Published on the user’s implementation instruction. Work the required frontier, starting with 01 and 02; conditional work and watches retain their separate activation rules.
