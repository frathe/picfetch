# Location Map release test gaps

Branch: `feature/location-map-release-qualification`, from merged `main`
`d2fff17`. Standard continuation of the accepted Location Map seams: viewer
actions and rendered state, the real metadata reader, source versions, and
Favorite/live cache interfaces. The user requested closing the six documented
verification gaps, not waiving them. No new feature, dependency, merge, release,
signing operation, or publication is authorized by this work.

## Acceptance and tasks

T0 owns test design, implementation, review, and any fixes. Reuse existing
`newTestUI` and Location Map fixtures; observe tracked completion rather than
sleeping. Each case runs individually, then with the race detector. If current
behavior already passes, temporarily violate the relevant production invariant,
observe a behavioral failure, and restore it before proceeding.

1. Repeated occurrences: one source fact is reused while distinct occurrences
   retain their identity and count through removal/replacement.
2. Real metadata: encoded limits, XMP-only/malformed metadata, and read-only
   sources give honest outcomes without modifying source files.
3. Reuse: unchanged map entry and image/cluster return reuse completed GPS facts;
   changed sources do not reuse them.
4. Rebuild: source changes refresh visible thumbnails and positions without
   moving a manually selected camera.
5. Competing writes: committed writes racing collection replacement update only
   live matching sources and cannot restore retired map work.
6. Empty transitions: mutation/removal of the last located member updates mounted
   empty states and preserves source bytes during read-only browsing.

Primary oracle for items 1–6:
`go test -tags no_emoji,nodynamic -count=1 -run '^TestLocationMapReleaseQualification$' ./internal/ui`.
If the real metadata boundary needs additional fixtures, use the existing
`internal/imaging/exif_containers_test.go` with an individually named test.
Files: a focused UI test file, existing production owners only for reproduced
defects, exact Qodana exclusions and UI shard assignment, this record, and
`todos.md`. No new runtime seam or package is planned.

Release preparation is a separate check: inspect the existing workflow's CI,
packaging, signing, and final-archive notice gates and run local notice checks.
Actual six-platform release artifacts and signing remain unverified until an
authorized release run; older binaries cannot prove new notice delivery.

Task graph: bounded recon -> six sequential behavioral slices -> focused
regressions/GoLand -> `make verify` once -> evidence and handoff.

## Delegation and budget

One read-only T3 scout maps existing controlled-write and replacement fixtures
in `internal/ui/{locationmap_test.go,savework_test.go,filework.go}`. No review,
test design, edits, or subdelegation. G1: a standalone prompt under 25 lines;
G2: returned file/line locators verified by T0 with targeted reads; G3: three
read-only files; G4: independent source-write control-flow lookup while T0
handles metadata/cache contracts; G5: those fixture relationships are unread.
S: callback/worker relationships require comprehension, not a textual transform.
W: no implementation is supplied or delegated. Budget: one scout total, zero
implementation/review delegates, one final full suite. All fixes remain T0.

## Evidence

- Baseline worktree clean. PR 63 is already merged in `main` (`d2fff17`).
- Created the requested feature branch from that main commit.
- Existing performance acceptance and Windows/Store HEIC deferral are unchanged.
- Added `internal/ui/locationmap_release_test.go`, using the pre-approved
  viewer, real metadata, source-version, and committed-write seams. Assigned its
  one top-level test to `ui-2` (249 entries) and added the exact Qodana exclusion.
- Each of the following six groups passed on the existing production behavior.
  Each corresponding deliberate production violation was observed failing, then
  restored. `git diff` confirms no production-code change remains.

| Acceptance | Subtest | Deliberate violation and observed failure |
| --- | --- | --- |
| 1 | `repeated_occurrences` | Disable raw-fact reuse: two source reads instead of one. |
| 2 | `metadata_limits_xmp_read_only` | Classify unreadable as unlocated: encoded-limit failure became absent GPS. |
| 3 | `unchanged_entry_and_return` | Disable version-change detection: image/cluster return reused stale metadata. |
| 4 | `rebuilt_thumbnail_and_camera` | Reset the camera during rebuild: unchanged anchor moved. |
| 5 | `competing_writes_and_replacement` | Omit committed-source reconciliation: newer GPS remained stale in all four delivery/closure cases. |
| 6 | `mutation_to_empty_read_only` | Keep facts when membership becomes empty: all three routes retained retired facts. |

Run an individual acceptance command by appending `/^SUBTEST$` to the anchored
test expression above. There are 15 leaf cases, including direct/cluster/entry
routes, retained/removed old sources, map closure, and read-limit recovery.

- New complete acceptance group with `-race`: PASS, `internal/ui`, 12.155s.
- Existing `TestLocationMap` regressions for `automatic_rebuild`,
  `external_changes`, `gps_cache_policy`, `thumbnail_reuse`, `camera_retention`,
  and `empty_states_and_read_only` with `-race`: PASS, `internal/ui`, 20.512s.
- GoLand `get_file_problems`, `errorsOnly=false`, on the sole changed Go file:
  no errors or warnings, no timeout. This is the documented IDE inspection
  fallback, not a Qodana SARIF result. No new branch CI scan has run yet.
- `make check-test-platform`: PASS (native Linux/amd64 Docker).
- Analyzed test-file SHA-256:
  `b3ce6675cf0ae04f6ad31ee88490a90ecc8bbd055708e84b44883b89791595a1`.
- `make verify` has passed formatting, TUF root, exact Qodana exclusions,
  generated tag vectors/artwork, updater/AVIF notices, `go vet`, and `go build`.
  The complete native Linux/amd64 Docker race suite also passed; Make and
  container exit codes are both zero, with no OOM kill. Shard inventory: 707
  runnables. UI package times: ui-1 475.948s, ui-2 364.732s, ui-3 379.289s.
  The new acceptance group passed in Docker in 13.820s. The separately tracked
  similarity-protocol timeout did not recur in this run (11.510s); this alone
  does not establish that its historical load sensitivity is fixed. Artifacts:
  `.scratch/race-runs/20260926T200018Z-o6jGjk/`.
- Release-note preview (`go run ./scripts/releasenotes --prev 1.1.9 --next
  1.1.10`) passed without writing release notes, changing versions, or tagging.
- Release workflow inspection confirms CI precedes packaging, Windows signature
  verification precedes publication, and all six final archives are checked by
  `scripts/updaternotices` immediately before creating the GitHub release.
  Building/signing/publishing those future artifacts remains pending.

## Handoff

The six documented composite gaps are closed by behavioral coverage, not by a
risk waiver. No production defect was reproduced; all temporary mutations were
restored. The actual change is tests, inventory/exclusions, and evidence/docs.
No runtime package, dependency, app identity, native policy, or notice changed.
No architecture-map update is needed for a test added within an existing package.

No commit, push, new PR, merge, release, or signing operation was performed.
Fresh branch CI (including Qodana post-suppression SARIF and CodeQL) and review
remain future gates after commit/PR authorization. Local tests do not qualify
native Windows/macOS graphical behavior or future signed archives. The existing
maintainer performance acceptance and Windows/Store HEIC deferral remain intact.

## Ledger

| Task | Spawns budget/actual | Review | Full suite |
| --- | --- | --- | --- |
| Controlled-write fixture recon | 1/1 read-only scout | Lead checked returned locators | No |
| Six behavioral slices and negative guards | 0/0 | Lead-owned, all six violations caught | No |
| Final inspection and verification | 0/0 | Lead-owned; GoLand, notices, vet/build, shard inventory and race suite pass | Once, passed |
