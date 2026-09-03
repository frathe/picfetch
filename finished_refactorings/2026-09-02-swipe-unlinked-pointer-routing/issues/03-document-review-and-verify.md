# 03 - Document, review, and verify the Swipe routing fix

Status: resolved
Blocked by: 01, 02

## Contract

Record the approved terminology and implementation invariant, review the two
vertical TDD slices, negatively verify their guards, and run the final gate
once.

Add **Linked comparison** and **Unlinked comparison** to `CONTEXT.md` and mark
locked/unlocked comparison as avoided terminology. Update `ARCHITECTURE.md` to
state that Swipe input bounds mirror the reveal while wheel coordinates are
translated back into the full viewport. Add the bugfix to `todos.md` and
normalize its existing locking/unlocking wording to linking/unlinking.

Complete the local spec and ticket comments with observed evidence, record the
Standard-route plan and cost ledger, and move the completed plan to
`finished_refactorings/` after the final gate.

## Verification

1. Run `go test ./internal/ui/compare -count=1`.
2. Run
   `go test ./internal/ui -run 'Compare(LinkToggle|SwipePointer)' -count=1`.
3. Temporarily restore full-width pane inputs and confirm Ticket 01 fails for
   the original Right-over-left symptom; restore the fix.
4. Temporarily remove scroll-coordinate translation and confirm Ticket 02
   fails for lost cursor anchoring; restore the fix.
5. Rerun both focused ticket commands on the restored tree.
6. Run `make verify` once and record its actual result.

## Acceptance

- Every spec acceptance command passes on the final tree.
- `rg -n 'Linked comparison|Unlinked comparison' CONTEXT.md` finds both
  canonical terms.
- `rg -n 'reveal|revealed' ARCHITECTURE.md todos.md` finds the architecture and
  release-note records.
- `make verify` passes.
- No diagnostic files or debug instrumentation remain.

## Constraints

- Leave the already-correct manuals and translations unchanged.
- Do not create an ADR or claim a manual native UI smoke test.
- Do not commit; provide the suggested commit message at handoff.

## Comments

- `go test ./internal/ui/compare -count=1` passed, as did the assembled
  `Compare(LinkToggle|SwipePointer)` selection and both focused guards.
- Deliberately restoring full-width inputs reproduced `Unlinked: Right` over
  the left reveal. Deliberately removing coordinate translation reproduced the
  wheel-anchor drift from `0.625` to `0.5774`. Both fixes were restored and
  both guards passed again.
- `CONTEXT.md`, `ARCHITECTURE.md`, and `todos.md` now record the approved terms,
  invariant, and bugfix. Manuals, translations, and ADRs were left unchanged.
- `make verify` passed: formatting, embedded TUF-root check, vet, build, and the
  complete Linux/amd64 race suite were green (`internal/ui` 676.609s;
  `internal/ui/compare` 28.486s).
