# Configurable Favorite preview limit

User request: expose the automatic original-decode limit in Settings' Cache
tab, explain it in one sentence, and default to 1000. Existing installations
without the new preference also receive 1000. Positive whole numbers follow
the existing numeric-entry convention. Existing previews beyond the limit
remain reusable; original decoding remains serial.

Route: Deep because the value crosses preference storage, viewer lifecycle,
the settings surface and the preview worker. Lead owns all implementation and
review. No new dependency, worker, native integration or source-file mutation.
One existing read-only scout continues hosted-check evidence collection.

1. Add the saved value and default, then wire startup, settings snapshots,
   shutdown persistence and ApplySettings. Capture the value before launching
   a pass; changing it cancels the old pass and applies to the next Favorite
   open/save. Files: preferences.go and tests; UI favthumbs, memlimits, features,
   run, startup and existing wiring/preview tests. Verify focused preferences
   and viewer tests for default/custom values, invalid input and cancellation.
2. Pass an explicit positive limit to favthumbs.Sync and bound preparation by
   it. Keep the complete-list reuse and sweep behavior. Update existing call
   sites and test custom small/large limits, cancellation and cached tail reuse.
   Verify the Favorite-cache race suite and viewer preview regressions.
3. Group the preview toggle and numeric limit in Cache, with wrapped explanatory
   text. Update English/German translations, both manuals and architecture.
   Verify settings tree membership, live edits, invalid input, close cleanup,
   locale/manual guards and rendered layout. Assign new UI tests to shards.

Dependencies: value contract first; worker and UI wiring follow; final
verification/review last. Each new behavior gets a failing regression before
the fix; use temporary overlays to verify any additional guards. Run GoLand
on every changed code file, formatting/exclusions/shards, vet/build, focused
local race tests, then full CI and a fresh Codex review on the pushed head.
The bounded native memory comparison remains a separate pending check.

## Verification

- The initial acceptance tests failed for the missing saved default, absent
  Cache control, missing viewer wiring and worker ignoring a selected limit.
  They pass with a 1000 default and custom positive values. Preparation tests
  cover limits of 1, 256 and 1000 without reading an unused tail.
- Preferences, preview-worker and Settings race suites passed (1.033s, 3.340s,
  3.389s); viewer startup/apply/persistence/preview coverage passed (7.923s).
  Subsequent review fixes also pass the targeted viewer regressions (7.999s).
- English and German Cache tabs were rendered and inspected at 520x520. The
  checkbox, number and wrapped explanation fit without clipping. The test
  walks the actual Cache surface and verifies close cleanup and invalid input.
- Locale parity and manual guards passed. No new test files or dependencies.
  The new root UI test is assigned to ui-1; the manifest covers 694 tests.
- Lead implemented/reviewed all changes. One existing scout collected hosted
  checks for the previous head; fresh exact-head review is required after push.
- Formatting, Qodana exclusions, the Docker shard inventory, vet/build and
  GoLand inspections of all 18 changed Go files passed. Final cleanup ownership
  regressions passed under the race detector (2.980s). Full hosted verification
  follows the push; no native desktop comparison is claimed complete.
