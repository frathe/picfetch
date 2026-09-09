# 01: Browse known duplicate groups while analysis continues

**What to build:** Shift+D immediately shows the highlighted image's already
known duplicates while the remaining scan continues. Keep that source's group
current and usable, preserve normal exits and completed-scan behavior, and
document the corrected behavior in both manuals.

**Blocked by:** None (can start immediately).

Status: complete — AC1–AC8 pass; see [verification evidence](../evidence/verification.md)
Parent: [Show known duplicate groups during Grid View analysis](../spec.md)
Coverage: AC1–AC8; user stories 1–15
Route: Standard
Owner: lead for framing, design-bearing tests, implementation, review, and final gate

## Context and contract

The reported failure occurs in an open grid while duplicate analysis is still
running: an image already has known duplicates, but Shift+D shows all images.
The reporter confirms completed scans work correctly and wants the duplicates
known at the point of invocation to appear immediately.

The highlighted grid item is the source, even if the underlying image view
has a different current file. Show only its accepted group, including hidden
extras. Other duplicate groups and unique images must remain outside variants.
Global scan completion must not gate an already known group.

Continue analysis using the existing workers. Keep the browse source fixed by
file identity and update its group as valid membership results arrive. While
another result is pending, preserve the established group rather than showing
the whole list. Ignore results belonging to obsolete file sets or canceled
browse sessions.

Preserve the existing interaction contracts: selection/search keep their
Escape precedence; Escape leaves browse before disabling hide; a second
Shift+D leaves browse; G/Close leave browse without disabling hide. Opening a
hidden extra must still open that chosen file and retain the existing inspect
and return-to-grid behavior. Reordering retains the source by identity;
removing it, or an accepted update dissolving its established group, ends browse.

Requests made before any source group is known retain their existing waiting
and pending-grid behavior. The unanswered proposal to open automatically upon
the first discovered match is not part of this ticket.

## Acceptance criteria

Each checkbox carries the parent criterion's verification command. These are
requirements, not evidence of completed work. The new root test and its
subtests must be implemented; a command with no matching test, a skipped
guard, or a guard failing for an unrelated reason does not satisfy a criterion.

- [x] **AC1 — Immediate source-specific results.** With reads still held, Shift+D
  shows exactly the highlighted source's known group, including hidden extras.
  Check hide both on and off. Distinguish the highlighted source from the
  underlying current image, another group, and the full list.

      go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^known_group$' -count=1 -v

- [x] **AC2 — Continuing group updates.** Hold the next grouping delivery and assert
  the established group stays visible. Accept a new matching member while other
  reads remain held and verify it joins; an unrelated completed input does not.
  Moving the highlight within variants or changing the representative must not
  retarget the source. Completing the scan leaves the correct final group.

      go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^group_updates$' -count=1 -v

- [x] **AC3 — Cancellation survives late delivery.** While reads or delivery remain
  held, exercise Escape, second Shift+D, G, and Close in separate cases. Release
  and drain later work. Browse stays canceled, closed grids stay closed, and
  the original hide setting determines ordinary grid contents.

      go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^exit$' -count=1 -v

- [x] **AC4 — Identity and dissolution.** During partial browsing, reorder the file
  set and verify the same source's group by URI identity. Removing the source
  ends browse. An accepted sensitivity change dissolving an established group
  also ends browse. Stale queued results cannot restore the previous group.

      go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^source_identity$' -count=1 -v

- [x] **AC5 — Usable partial variants.** Before unrelated analysis completes, navigate
  and open a known hidden extra through normal key/click paths. Verify the
  chosen file opens and existing inspect/return behavior works. Check title and
  badge presentation against the actual group rather than the whole file set.

      go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^open_variant$' -count=1 -v
      go test ./internal/ui -run '^TestGridHighlight_Variants' -count=1

- [x] **AC6 — Completed scans and requests without known groups.** Retain settled
  Shift+D entry/exit, unique-source no-op, and existing pending-request behavior
  when no group was known at invocation. The older pending-browse test covers
  that distinct case; it is not a substitute for AC1.

      go test ./internal/ui/grid -run '^(TestHandleKey_ShiftDTogglesBrowseDuplicates|TestApplyFilter_BrowsePendingDoesNotCollapseGrid|TestSetBrowsingDuplicates_HashesRemainingWithoutWarm|TestDuplicateDistancePreservesPendingBrowse)$' -count=1
      go test ./internal/ui -run '^(TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup|TestHandleKeyEvent_ShiftDNoopOnUniqueDoesNotOpenGrid)$' -count=1

- [x] **AC7 — Accurate manuals.** Both manuals describe immediate known-group
  browsing and continuing analysis. Their no-known-group explanation matches
  the unchanged path. Review the manual diff for semantic agreement; automated
  guards enforce existing rendering constraints.

      git diff -- internal/ui/help/manual.md internal/ui/help/manual_de.md
      go test ./internal/ui/help -run '^(TestManualIsEmbedded|TestManualHasNoMarkdownTables|TestManualHasNoUnicodeArrows)$' -count=1

- [x] **AC8 — Repository completion gate.** Register the new root UI test in its
  shard and maintain exact Qodana exclusions if adding a test file. Run focused
  checks while iterating and the canonical make verify gate once for handoff.
  Retain actual output and do not claim success from inventory alone.

      make check-test-shards
      make verify

## Test design and execution

Use one end-to-end regression family, TestGridBrowseDuringAnalysis, built
through newTestUI/newTestViewer. Send Shift+D through the normal viewer key
handler using the existing modifier stub. Assert the visible URI identities
obtained from ResultIndexes and the viewer file accessors, not only counts or
browse-mode flags.

The controlled fixture contains two distinct known groups, a known unique
image, and held source reads. The highlight belongs to a different group from
the underlying current image. Use instance-owned ReaderURI inputs and a
retained drainable UIQueue to deliver accepted groups while unrelated reads
remain held. Release individual matching and unrelated inputs to exercise
progress without completing the full scan.

Prior art includes TestHideDuplicatesPublishesWhileSourceReadsRemainPending,
TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup, and
TestActionsMenu_ShowVariantsOpensGridOnPairAfterHide. Existing identity and
dissolution tests establish the behavior to preserve. Keep the regression at
the viewer seam; use a lower seam only for an ordering case that cannot be
expressed there. No separate prefactoring ticket or new production interface
is required by the spec.

Before the fix, run AC1 and capture a failure showing the wrong visible
identities while unrelated reads remain held. After the fix, run every
required behavioral subtest and compatibility command and inspect actual
output. Check the build-selected inventory before using focused filters.
The expected subtests are known_group, group_updates, exit, source_identity,
and open_variant.

Use observable callback/worker completion and synctest where applicable.
Do not sleep or infer delivery from a cleared in-flight counter. Do not call
Settle before an assertion that requires work to remain held. Release held
inputs and settle all tracked work during cleanup, including failures.

Keep every intermediate change formatted and the UI shard inventory valid.
Register the new root test when it is added; update exact Qodana exclusions if
adding a test file. Prefer extending existing test files where appropriate.
Inspect the English/German manual diff for behavior agreement: rendering
guards alone do not prove the wording is accurate.

## Boundaries and completion

The grid feature and viewer integration continue using the existing duplicate
model, accepted group snapshots, cancellable workers, and UI queues. Do not
change matching, hashing rules, distance thresholds, representative selection,
unknown-group UX, shortcut meanings, or platform behavior. Do not add mutable
package-level test seams, synchronous grouping/hashing to key handlers, or
unrelated refactors.

Before implementation, record the concrete file map, task contract, test
commands, and budget in the Standard implementation plan. Suggested budget:
one cohesive implementation task, zero implementation subagents, one lead
review followed by inline fixes, and one full make verify gate at handoff.
Read-only exploration is optional only for newly discovered breadth. Record
any justified routing or budget change in the plan.

This ticket is complete only when all AC1–AC8 evidence is recorded, required
guards have been observed failing appropriately before passing, both manuals
are consistent, the canonical verification gate passes, and open-work tracking
reflects the result. The lead owns review and any fixes arising from it.
Do not run git commit.

## Comments

Published from the approved specification in response to /to-tickets on
2026-09-09. One vertical slice owns all behavior, tests, manuals, and handoff
checks. No implementation has started; acceptance checkboxes remain open.

Implemented with the Standard SDD/TDD cycle on 2026-09-09. All required
behavioral slices, compatibility checks, negative guards, manuals, shard
inventory, and the canonical verification gate pass. The completed
[implementation record](../../../finished_refactorings/2026-09-09-grid-duplicate-browse-during-scan.md)
and [verification evidence](../evidence/verification.md) retain the result.
No commit was made.
