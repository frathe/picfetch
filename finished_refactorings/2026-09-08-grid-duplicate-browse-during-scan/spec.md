# Show known duplicate groups during Grid View analysis

Status: complete — AC1–AC8 verified; see [implementation evidence](evidence/verification.md)
Date: 2026-09-09
Source: /grill-with-docs discussion, published with /to-spec
Delivery: implementation, manuals, SDD/TDD regressions, and canonical verification complete

## Problem Statement

While Grid View is analyzing images for duplicates, pressing Shift+D on a
highlighted image with already known duplicates shows an unfiltered list of
all images instead of that image's duplicate group. This prevents users from
working with a group already identified while the rest of a large file set
is still being analyzed.

The reporter confirmed that the feature works as intended after scanning
finishes. This regression concerns access to known groups during analysis,
not the correctness of the completed scan.

## Solution

Shift+D on a highlighted image with a known duplicate group immediately shows
that group, including copies hidden by hide-duplicates. It uses the duplicates
known at that point in time and does not wait for unrelated analysis. Other
groups and unrelated images stay outside this variants view.

Analysis continues in the background. As accepted group updates arrive, the
variants view follows the same source image's current group. Waiting for a
fresh update must not temporarily replace an established variants view with
the full file list. Navigation, opening a variant, and leaving browse continue
to work during analysis.

### Settled decisions — do not relitigate

| Decision | Contract |
| --- | --- |
| Entry timing | A known group opens immediately while remaining images are being analyzed. |
| Initial contents | Show the highlighted source's known group at the time of the request, including hidden extras. |
| Filtering | One source's group, not all duplicated images or the complete file set. |
| Subsequent updates | Keep the source fixed by file identity and follow accepted membership updates for that source. |
| Completed analysis | Preserve the existing working behavior. |
| No known group at request time | Preserve existing behavior. Redesigning this case was discussed but not approved and is excluded. |

The reporter accepted immediate browsing with "yes show that we know at that
PIT" and confirmed that "when the scan is complete the feature works as
intended". Updating the displayed group as analysis progresses was part of
the proposal; this spec does not introduce a frozen historical snapshot mode.

## User Stories

1. As a Grid View user, I want Shift+D to open an identified group immediately, so that I can inspect duplicates before the rest of the scan finishes.
2. As a Grid View user, I want the highlighted image to determine the group, so that browsing follows the thumbnail I am working with even when another image is open underneath the grid.
3. As a Grid View user, I want only that source's group to appear, so that unrelated images and other groups do not obscure it.
4. As a user hiding duplicates, I want the group's hidden copies to appear in variants view, so that I can inspect every known copy.
5. As a user with a large file set, I want analysis to continue while I browse, so that inspecting one group does not interrupt discovery elsewhere.
6. As a user browsing variants, I want accepted matches to update the same source's group, so that the view reflects current analysis without switching subjects.
7. As a user browsing variants, I want the existing group to remain visible while another grouping result is pending, so that unrelated thumbnails do not flash into view.
8. As a user browsing variants, I want to navigate and open a known copy during analysis, so that I can work with results already available.
9. As a user browsing variants, I want the title, counts, and duplicate badges to follow the existing variants presentation, so that the mode and contents agree.
10. As a user leaving variants, I want Escape or a second Shift+D to restore the appropriate filter, so that my hide-duplicates setting remains effective.
11. As a user closing the grid, I want later analysis completions to respect that action, so that canceled browsing does not reopen itself.
12. As a user whose file list changes, I want browsing to follow the original source by identity and end if it disappears, so that an index change cannot select another shot.
13. As a user changing duplicate sensitivity, I want existing group-dissolution behavior preserved, so that a former group does not leave stale variants behind.
14. As a user returning after analysis completes, I want established Shift+D behavior preserved, so that the fix does not regress completed scans.
15. As a reader of either manual, I want the explanation to distinguish available known groups from pending analysis, so that the documentation matches the feature.

## Implementation Decisions

- Keep the change within the existing grid feature and viewer integration.
  The duplicate model retains ownership of matching facts and accepted groups.
  This is a presentation and browse-lifecycle correction.
- A source with an accepted group of at least two members can enter variants
  while other hash jobs remain pending. An empty global work queue is not a
  prerequisite for presenting that known group.
- Use accepted membership belonging to the current file set. Do not expose
  unaccepted worker results or reuse an obsolete file-set snapshot.
- Capture the highlighted source before filtering. Track its file identity
  across reordering, representative changes, and later group updates.
- Preserve an established group's contents while the next valid grouping
  result is pending. Apply accepted updates for the same source; never fall
  back to the full gallery merely because analysis continues.
- Preserve exits: selection/search retain their Escape precedence; Escape
  then leaves browse before hide; a second Shift+D leaves browse; G/Close
  leave browse without disabling hide. A late completion cannot restore a
  canceled browse or reopen the closed grid.
- Preserve source-removal and dissolution behavior. Removing the source ends
  browse. If an accepted update leaves an established group with fewer than
  two members, use the existing browse exit behavior.
- Opening a variant still opens that chosen file, including a hidden extra.
  Preserve the existing inspect and return-to-grid behavior, and keep variants
  title and badge presentation consistent with the group shown.
- Use existing cancellable workers and UI queue. Do not introduce synchronous
  hashing/grouping on key handling, mutable package-level test seams, or a new
  production interface for this fix.
- Keep the existing path for requests made before a group is known. Do not
  choose among the unanswered alternatives for that separate interaction.
- Update both manuals to describe immediate known-group browsing. Reuse
  existing presentation and localized strings; new UI copy is not required.

## Testing Decisions

### Test seam and prior art

Use the highest existing seam: a viewer built with newTestUI/newTestViewer,
an open grid, the existing modifier stub, and Shift+D sent through the normal
viewer key handler. Assert visible file identities by mapping ResultIndexes()
through the viewer's file accessors. Browse flags or counts alone cannot
distinguish the correct group from another group of equal size.

Use instance-owned uitest.ReaderURI fixtures and a retained drainable UIQueue
to keep unrelated reads blocked while delivering accepted partial groups.
The partial-hide test supplies controlled-progress precedent. Its private
decode-pool setup must not become a new public interface. Use observable
worker/callback completion and synctest where applicable, never sleeps or an
in-flight counter as proof that the callback after it has been delivered.

The fixture contains two distinct known groups, a known unique image, and
held reads. Highlight a member of a different group from the viewer's current
image. Check the selected pair before releasing unrelated reads. Then release
individual matching/unrelated inputs while other work remains held.

The regression must fail on the current behavior for the reported reason
before the fix, then pass after it. Do not call Settle while held reads are
needed for the during-analysis assertion. Release inputs and settle tracked
work during cleanup, including failure paths.

Prefer one new root test, TestGridBrowseDuringAnalysis, with the behavioral
subtests below. Existing grid tests remain compatibility checks. Add lower
coverage only if a required ordering case cannot be expressed at the viewer
seam.

### Acceptance criteria and verification commands

These commands are implementation requirements, not executed evidence. The
new test and subtests do not exist yet. Check the build-selected inventory and
confirm each required subtest actually runs: no matching tests, or a skipped
guard, is not a pass.

**AC1 — Immediate source-specific results.** With reads still held, Shift+D
shows exactly the highlighted source's known group, including hidden extras.
Check hide both on and off. Distinguish the highlighted source from the
underlying current image, another group, and the full list.

    go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^known_group$' -count=1 -v

**AC2 — Continuing group updates.** Hold the next grouping delivery and assert
the established group stays visible. Accept a new matching member while other
reads remain held and verify it joins; an unrelated completed input does not.
Moving the highlight within variants or changing the representative must not
retarget the source. Completing the scan leaves the correct final group.

    go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^group_updates$' -count=1 -v

**AC3 — Cancellation survives late delivery.** While reads or delivery remain
held, exercise Escape, second Shift+D, G, and Close in separate cases. Release
and drain later work. Browse stays canceled, closed grids stay closed, and
the original hide setting determines ordinary grid contents.

    go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^exit$' -count=1 -v

**AC4 — Identity and dissolution.** During partial browsing, reorder the file
set and verify the same source's group by URI identity. Removing the source
ends browse. An accepted sensitivity change dissolving an established group
also ends browse. Stale queued results cannot restore the previous group.

    go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^source_identity$' -count=1 -v

**AC5 — Usable partial variants.** Before unrelated analysis completes, navigate
and open a known hidden extra through normal key/click paths. Verify the
chosen file opens and existing inspect/return behavior works. Check title and
badge presentation against the actual group rather than the whole file set.

    go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$/^open_variant$' -count=1 -v
    go test ./internal/ui -run '^TestGridHighlight_Variants' -count=1

**AC6 — Completed scans and requests without known groups.** Retain settled
Shift+D entry/exit, unique-source no-op, and existing pending-request behavior
when no group was known at invocation. The older pending-browse test covers
that distinct case; it is not a substitute for AC1.

    go test ./internal/ui/grid -run '^(TestHandleKey_ShiftDTogglesBrowseDuplicates|TestApplyFilter_BrowsePendingDoesNotCollapseGrid|TestSetBrowsingDuplicates_HashesRemainingWithoutWarm|TestDuplicateDistancePreservesPendingBrowse)$' -count=1
    go test ./internal/ui -run '^(TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup|TestHandleKeyEvent_ShiftDNoopOnUniqueDoesNotOpenGrid)$' -count=1

**AC7 — Accurate manuals.** Both manuals describe immediate known-group
browsing and continuing analysis. Their no-known-group explanation matches
the unchanged path. Review the manual diff for semantic agreement; automated
guards enforce existing rendering constraints.

    git diff -- internal/ui/help/manual.md internal/ui/help/manual_de.md
    go test ./internal/ui/help -run '^(TestManualIsEmbedded|TestManualHasNoMarkdownTables|TestManualHasNoUnicodeArrows)$' -count=1

**AC8 — Repository completion gate.** Register the new root UI test in its
shard and maintain exact Qodana exclusions if adding a test file. Run focused
checks while iterating and the canonical make verify gate once for handoff.
Retain actual output and do not claim success from inventory alone.

    make check-test-shards
    make verify

## Out of Scope

- Changing matching, hash generation, sensitivity thresholds, or
  representative selection.
- Redesigning requests made before the first duplicate is known: automatic
  opening on first discovery, a one-image waiting view, or requiring another
  Shift+D. Preserve existing behavior rather than treating an unanswered
  question as approval.
- A frozen snapshot or separate duplicates-only gallery mode.
- New shortcuts, comparison features, selection semantics, or file operations.
- General scan performance work, worker-pool redesign, package extraction,
  or platform-specific behavior changes.

## Further Notes

### Repository evidence

- internal/ui/help/manual.md:399 specifies source-specific browsing; :411
  documents the current blanket wait. The German counterpart is in
  internal/ui/help/manual_de.md:458.
- internal/ui/grid/search.go:127 makes pending browse disable hide while
  group filtering requires no remaining hash jobs.
- internal/ui/grid/dupes.go:149 gates browse readiness on global completion;
  internal/ui/grid/groupwork.go:72 handles accepted group updates.
- internal/ui/grid/dupes_test.go:963 expects all files during a request made
  without known hashes, then the pair after settlement.
- TestHideDuplicatesPublishesWhileSourceReadsRemainPending supplies partial
  progress precedent; TestHandleKey_ShiftDTogglesBrowseDuplicates supplies
  settled grid-keyboard precedent.
- TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup supplies root keyboard
  precedent. The normal dispatcher forwards keys to an open grid.
  TestActionsMenu_ShowVariantsOpensGridOnPairAfterHide demonstrates root
  result-index assertions.
- TestFilesChanged_PreservesBrowsedSourceByIdentity and
  TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits establish identity and
  dissolution behavior to preserve during partial browsing.

These source observations identify relevant behavior and a coverage gap.
The regression has not yet been reproduced by an executed failing test.
No application code or tests were changed or run when publishing this spec.

### Delivery and honest limit

This specification is ready for implementation without another product
interview. The unresolved no-known-group interaction is bounded out of scope.
A request made before a group is known retains today's waiting behavior,
including the existing pending-grid presentation. This fix removes the
unnecessary wait when the source's group is already known.

Expected implementation route: Standard under the SDD/TDD working agreement.
The lead owns the implementation plan, design-bearing tests, review, and
final gate. Record concrete tasks and routing budgets when implementation
starts. This publication creates neither implementation tickets nor code.
Keep open work linked from todos.md. Do not run git commit.
