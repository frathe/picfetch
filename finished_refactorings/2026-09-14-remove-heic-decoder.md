# Remove HEIC decoding pending distribution qualification

Ronin requested removal of the previous HEIC decoder on 2026-09-14 while
permission to distribute its embedded payload remains unresolved. Remove
`gen2brain/heic` and its fork replacement; do not select another HEIC decoder.
Preserve AVIF and the remaining formats. This is a removal decision, not a
legal conclusion that every use of the library is prohibited.

Route: Deep, because support declarations span imaging, packaging, and the
embedded manuals. No new runtime interface or platform behavior is introduced.
Initial checkout: current feature branch, one existing unpushed commit,
untracked prior research note and generated Fyne initializer. Preserve those.
The removal request initially covered local implementation. Ronin subsequently
authorized committing and pushing these changes to the current feature branch;
release and upstream contact remain outside this handoff.

## Acceptance criteria

1. No HEIC decoder or fork exists in the selected module graph or compiled
   application dependency graph. `go list -m all` and `go list -deps .` must
   contain none of `gen2brain/heic`, `frathe/heic`, `gen2brain/h265`.
2. HEIC/HEIF extensions and MIME types are unsupported, and actual HEIC bytes
   cannot use a registered HEIC decoder. Existing imaging admission and image
   decode tests, updated to assert refusal, must fail before implementation
   and pass afterward. Retain real HEIC test data for the refusal regression.
3. AVIF decode/metadata, ordinary images and RAW previews retain their behavior.
   Run imaging and file-scan tests plus focused affected UI regressions.
4. Generated macOS associations and other current packaging declarations do
   not advertise HEIC. Run `go test ./scripts/plistdoctypes ./scripts/msixstage`
   and inspect association consumers discovered during reconnaissance.
5. README, embedded manuals, Store listing, current agent conventions and
   dependency notices reflect the removal. Retain historical audit evidence
   with an explicit current disposition. Run manual/locale guards, Qodana
   inventory, formatting and `make verify` as the final gate.

## Tasks and ownership

| Task | Files / contract | Verification | Owner |
| --- | --- | --- | --- |
| Admission and dependency removal | `internal/imaging/{loader,exif,save}.go`, existing tests, remove HEIC-only RSS files, `go.mod`, `go.sum`; no supported HEIC path | Imaging tests, module and app graph | Lead |
| Association and format exposure inventory | Read-only search through packaging/scripts/platform integrations and live manuals; identify generated consumers and remaining format claims | Lead reruns reported searches and affected generator tests | Scout |
| Exposure and notice cleanup | `scripts/plistdoctypes`, README, manuals, Store listing, current comments, Qodana exclusion, notices, AGENTS/architecture, todos/audit | Packaging tests, manual guards, complete remaining-reference scan | Lead |
| Final qualification | Focused race suite, Make gate, changed-file GoLand inspections | Actual commands and inspection output recorded below | Lead |

Task dependencies: inventory runs beside imaging changes; exposure cleanup
uses its findings; final qualification follows both. No code edits, reviews,
licensing decisions or user-visible text are delegated.

Scout gate: G1 the question is a bounded exposure inventory; G2 exact paths
and consumer references are verifiable with `rg`; G3 no writes; G4 no prior
conversation needed; G5 lead has not traced platform generation consumers.
Shell has identified literal mentions, but does not resolve which generated
manifests inherit the imaging list. Rule S does not replace that tracing;
Rule W holds because no implementation is prescribed to the scout.

Budget: one read-only scout, lead review/fixes, one final complete Make gate.
No additional dependencies or replacement codec are in scope.

## Evidence

Implemented in the working tree. Removed the dependency and replacement,
HEIC image registration and metadata call, HEIC/HEIF extensions and MIME
admission, native leak-test helpers and their Qodana entry, and the macOS UTI
mappings. No decoder was substituted and no dependency version was upgraded.
`make tidy` removed only the HEIC requirement/replacement and its two sums.
The real HEIC fixture remains as refusal coverage, with its existing MIT
attribution moved beside the test data. It is image data, not decoder code.

Windows MSIX associations and Explorer's format choices inherit
`SupportedExtensions()`. Native chooser results, saved sessions and Favorite
opens flow through the scanner's admission. Existing cached Favorite preview
images may still be reused; no saved images, Favorites or caches were erased.
The existing AVIF box parser remains for generic ISOBMFF metadata. This does
not introduce a replacement HEIC pixel decoder. Historical audit records are
retained and the obsolete fork-upgrade watch is closed by removal.

Observed red before implementation, in
`/private/tmp/picfetch-remove-heic-red.log`: five HEIC/HEIF extension/MIME
cases incorrectly returned true; real HEIC header and pixel decoding and
canonical loading still succeeded, including the renamed `.jpg`; generated
macOS associations still included HEIC. The same focused tests passed after
removal (`internal/imaging` 1.080s; `scripts/plistdoctypes` 0.782s).

Final local verification:

- `go list -mod=readonly -m all`: 442 selected modules, none of
  `gen2brain/heic`, `frathe/heic`, or `gen2brain/h265`.
- `go list -mod=readonly -tags no_emoji -deps .`: 812 application dependencies,
  none of those decoder paths. `go version -m bin/picfetch` independently
  confirms their absence from the rebuilt executable; AVIF v0.6.0 remains.
- `go test -race -tags no_emoji -count=1 ./internal/imaging ./internal/filescan
  ./internal/favthumbs ./internal/explorerpresets ./scripts/plistdoctypes
  ./scripts/msixstage`: imaging 24.494s, filescan 1.858s, favthumbs 1.817s,
  plistdoctypes 2.679s and msixstage 7.077s all pass; explorerpresets has no
  tests. Includes AVIF, RAW, ordinary image and generated association coverage.
- Root locale guards pass (1.660s) and help/manual guards pass (2.851s).
  The first shared test-name filter also selected the unrelated
  `TestManualUpdateCheck_BypassesSettingAndDailyGate`, whose local HTTP server
  was denied by the execution sandbox. A corrected UI-only selection of
  Favorite opens/preview sync and export tests passes (34.088s). This is not
  recorded as a passing updater test. The changed `TestSuggestedExportPath`
  also passes separately under the race detector (1.586s).
- `make fmt`, `make verify-build build`, Windows/amd64 no-cgo
  `go vet -tags no_emoji ./internal/...`, and `git diff --check` pass. The Make
  gate includes Qodana's exact test-file inventory and updater notice checks.
  Native linking emits the existing duplicate `-lobjc` warning; a sandbox
  write warning for the repository's Go module stat cache did not prevent
  either build from succeeding.
- `make verify` correctly stops at the native Linux/amd64 platform guard:
  the selected daemon is `linux/aarch64`. Complete Linux/amd64 CI and actual
  Windows execution are unverified. No isolation test or platform guard was
  bypassed. No root UI top-level test was added or renamed, so shard
  assignments are unchanged.

GoLand inspected all 13 changed Go files, `go.mod` and `qodana.yaml`, including
weak warnings. No new actionable findings remain. Existing duplicated test
setup retains exact-file Qodana exclusions; the plist fixture's HTTP DTD
identifier is test text and is not fetched. The unchanged `LoadedImage`
padding suggestion is outside this removal. Three allegedly unused indirect
modules are required by the updater, confirmed with `go mod why -m`; version
upgrade suggestions concern unchanged dependencies and are outside scope,
including the explicitly deferred Fyne upgrade and Intel runtime pin.

Logs and graph inventories are under `/private/tmp/picfetch-remove-heic-*`.
No commit or push was performed during implementation and verification.
Ronin subsequently authorized the feature-branch commit and push. No upstream
contact or release was performed.

| Task | Spawns budget/actual | Lead review | Full suite |
| --- | --- | --- | --- |
| Exposure inventory | 1 / 1 | Source/consumer trace confirmed | No |
| Implementation and documentation | 0 / 0 | Red/green guards, diff and IDE findings assessed | Focused race tests |
| Final gate | 0 / 0 | Build, vet, graphs and notices pass | Attempted once; native amd64 guard blocks |
