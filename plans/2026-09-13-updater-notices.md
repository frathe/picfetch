# Updater dependency notices

Route: Deep (six desktop targets and final release/Store packaging).
Deliverable: reconcile the pinned updater source licenses and preserve their
complete applicable LICENSE/NOTICE text in every distribution.

## Scope and decisions

- Audit the production `internal/update` dependency union for Darwin, Linux and
  Windows on amd64/arm64, with cgo enabled and release tags. Tests/tools are excluded.
- Keep dependency versions and the upstream verifier unchanged. No new dependencies.
- Record exact module versions, upstream source locations and selected license
  files in a reviewed manifest. Generate only a bounded updater section in the
  existing notices document; preserve the existing non-updater inventory.
- Include upstream notices and selected-source exceptions, not just SPDX names.
  Apache 2.0 section 4 requires distribution of its license and applicable NOTICE
  attribution: https://www.apache.org/licenses/LICENSE-2.0 . MIT/BSD notices retain
  their original copyright, conditions and disclaimer. File-specific licenses
  and exact provenance will be recorded below after source inventory.
- Verification seams are the tooling command, its generated document, and final
  archive contents. The implementation request authorizes these routine tests.
- Final MSIX generation/signing requires Windows SDK CI; local ZIP fixtures prove
  the notice checker, not native package validity. No publishing or commits.

## Acceptance criteria

1. Every module in the six-target production updater union has a matching reviewed
   version and complete selected license/NOTICE text. `make check-updater-notices`.
2. Missing/stale notices, changed dependency versions or source license bytes fail
   closed. `go test ./scripts/updaternotices -run TestNotices`.
3. Final macOS ZIP, Windows ZIP, Linux tar.gz and each x64/ARM64 MSIX payload contain
   byte-identical root notices. Missing/stale/duplicate entries fail.
   `go test ./scripts/updaternotices -run TestArtifact` and
   `go run ./scripts/updaternotices -artifact <finished artifact> ...`.
4. Release publication and Store upload run final artifact checks, and ordinary CI
   checks the source inventory. `go test ./scripts/updaternotices -run TestWorkflow`.
5. `make verify`; inspect every changed Go file in GoLand including weak warnings.
   Report environment-blocked gates explicitly.

## Tasks

| Task | Owner | Files | Verification | Budget |
|---|---|---|---|---|
| Inventory | Lead + read-only scout | evidence in `.scratch/updater-notices/` | six-target go-list inventory and source file audit | 1 spawn, 1 review, no full suite |
| Source notices | Lead | `scripts/updaternotices/main.go`, `main_test.go`, `manifest.json`, `THIRD-PARTY-NOTICES.md` | AC1/AC2 | 0 spawns, 2 reviews, no full suite |
| Artifact delivery | Lead | `scripts/updaternotices/artifacts.go`, existing test file, release/Store/CI workflows, Makefile | AC3/AC4 | 0 spawns, 2 reviews, no full suite |
| Final gate/docs | Lead | architecture, Qodana exact test exclusion, todos, this record | AC5 | 0 spawns, 1 full gate |

Graph: inventory -> source notices -> artifact delivery -> final gate/docs.
Packaging reconnaissance runs alongside the independent source inventory.

Delegation G1-G5: bounded read-only production-source inventory; go-list/file
evidence is the oracle; no shared edited files; package/file exception discovery
needs a broader sweep than the lead's packaging work; lead has not built this
source context. S: scripts collect bytes, scout identifies file-level exceptions.
W: no implementation handed off. All design, review and fixes remain with lead.

## Source audit and distribution obligations

The six successful cgo-enabled production inventories resolve the same 66
module/version pairs, comprising a 322-package union. The complete versions,
package names, source archive URLs, source/header ranges and hashes are retained
in `scripts/updaternotices/manifest.json`: 97 notice-source entries, including two
historical license supplements. Direct verifier pins remain Sigstore v1.10.9,
sigstore-go v1.3.0 and go-tuf/v2 v2.4.2. No dependency changed.

The source audit includes complete LICENSE, NOTICE and COPYRIGHT files, including
go-tuf's NOTICE. Selected exceptions include denco's MIT license, Go BSD headers
in OpenAPI middleware, certificate-transparency, in-toto and grpc-gateway,
Google's generated protobuf headers, YAML's Simonov/Canonical MIT notice, and
explicit Go adaptations in timestamp, ULID and OpenTelemetry. Complete common
Apache/BSD text is linked where source headers contain only references. The
generator deduplicates identical bytes, never license conditions or attribution.
Non-shipped Python/Ruby/TypeScript bindings, documentation licenses and unselected
Prometheus/casing packages were excluded. Upstream-retained go-pathspec
`GO-LICENSE` was documented for the subsequently removed `chefignore.go`; it is
not a license for the selected matcher.

`gitignore.go` in go-pathspec v1.3.0 has 26 matching nontrivial comment lines and
matching algorithm structure with [Python pathspec 0.2.2 at
5e891d8](https://github.com/cpburnz/python-pathspec/blob/5e891d818a94f43c79dc8962a6f4e474f3c16988/pathspec/gitignore.py).
Examples are Go lines 147-148 versus Python 56-57 (escaped pattern prefixes),
Go 191-203 versus Python 97-109 (double-asterisk cases), and Go 226-227 versus
Python 144-145 (the fnmatch attribution). Its [metadata](https://github.com/cpburnz/python-pathspec/blob/5e891d818a94f43c79dc8962a6f4e474f3c16988/pathspec/__init__.py)
identifies Copyright 2013 Caleb P. Burns and MPL-2.0. This is an inference from
primary source comparison; no upstream relicensing exception was found.

Preserve the original MPL license/copyright and provide the exact distributed Go
source archive under the matcher entry. Its inherited `fnmatch.translate()`
attribution identifies no CPython version. The complete [CPython v2.7.6
license](https://github.com/python/cpython/blob/v2.7.6/LICENSE) is retained as a
contemporaneous ancestry reference, with that uncertainty stated in the shipped
notices and a description of the matching adaptation. No Python runtime ships.
The two original license files are stored under `scripts/updaternotices/licenses`
with source URLs and full-file hashes in the manifest. This preserves identified
attribution without pretending to establish the exact historic Python pin.

Apache redistribution/NOTICE requirements follow [Apache 2.0 section 4](https://www.apache.org/licenses/LICENSE-2.0).
MIT/BSD copyright, conditions and disclaimers are reproduced; MPL matcher source
availability and the original license are explicitly delivered. The existing
non-updater inventory is retained, with its introduction corrected to distinguish
historical entries from the new verified updater inventory.

## Verification evidence

- Red: the initial missing-inventory guard and empty-archive guard failed for
  acceptance of missing notices. The real 66-module source check then failed
  because the generated section was absent. All five workflow-order requirements
  failed before workflow integration. Each became green after implementation.
- `go test -race ./scripts/updaternotices ./scripts/msixstage -count=1` passed:
  updaternotices 3.157s; msixstage 5.954s. After adding the selected-package guard,
  updaternotices passed again in 3.102s (`focused-race.log`).
- Negative overlays outside the source tree disabled version/package checking
  and artifact checking. Version and newly imported package cases failed;
  malformed ZIP/tar/MSIX and both-payload bundle cases failed. Original sources
  remained unchanged (`inventory-negative.log`, `artifact-negative.log`).
- `make verify-build` passed after the final changes: formatting, TUF root,
  generated vectors/artwork, Qodana exclusions, updater inventory, full vet and
  build. The linker retains its existing duplicate `-lobjc` warning.
- Windows amd64 `go vet ./internal/...` and the Windows build of
  `scripts/updaternotices` passed. The new helper uses only the standard library.
- GoLand inspected all three changed Go files, including weak warnings. Initial
  results were empty. Later temporary negative-test source copies inside scratch
  caused duplicate warnings; moving those deliberate copies to `/tmp` removed
  the findings. Final inspections are empty; no source suppression was needed.
- `make package-mac`, `make package-windows package-linux`, and
  `GOARCH=amd64 make package-mac BIN_DIR=.scratch/updater-notices/macos-x86_64`
  succeeded. Intel output metadata confirms `GOARCH=amd64`. All six rebuilt
  binaries contain the reviewed 66 updater module versions. Local packaging's
  FyneApp build-number increments were restored to the original 466.
- Actual release-format archives were created with the workflow's ZIP/tar
  commands. All six pass `go run ./scripts/updaternotices -artifact ...`, reading
  byte-identical LICENSE, THIRD-PARTY-NOTICES.md and PRIVACY.md. Archives, binary
  build metadata and `archive-inspection.log` remain in `.scratch/updater-notices/`.

### Remaining external qualification

`make verify` stopped at `check-test-platform`: the selected Docker daemon reports
`linux/aarch64`; complete worker-isolation/race verification requires native
Linux/amd64. No isolation test was skipped or weakened. The independent full
static/build gate above passed.

This Mac has no Windows SDK, so it did not create/sign/WACK-test the final MSIX
bundle. Tests exercise its two nested payloads, and the Store workflow now checks
the final signed native bundle before provenance recording and upload. Likewise,
local Windows ZIPs are unsigned; the release workflow inspects all six artifacts
after Windows signing and before publication. These hosted checks need to run on
the eventual committed change before the feature is declared release-ready.
`todos.md` retains that narrow qualification item. No commits, pushes, signing
credential operations or publication were performed.

The source/license implementation is complete. The exact CPython ancestry pin
remains unknown and is not presented as resolved.

## Final ledger

| Task | Spawns budget/actual | Lead review rounds | Full suite | Result |
|---|---|---|---|---|
| Inventory | 1 / 1 | 1 | no | 66 modules, 322 packages, source exceptions/provenance recorded |
| Source notices | 0 / 0 | 2 | no | generation and source/version/package guards pass |
| Artifact delivery | 0 / 0 | 2 | no | six real archives pass; final MSIX check wired |
| Final gate/docs | 0 / 0 | 1 | attempted once | static/build green; native amd64/Windows CI qualification pending |

Plan remains in `plans/` until user acceptance. Concurrent WinGet edits to README,
its workflow, its new documentation and its todo entry were preserved.

## PR #22 Codex review loop — 2026-09-13

Ronin invoked the GitHub Codex review loop, authorizing commits and pushes on
this PR branch and review-thread replies/resolution. Earlier no-commit statements
describe the original implementation session. No merge or release is authorized.

The initial head is `a3b0b6029f4af2bb125fee62c5d03954d314a687`. There are no
review threads, including unresolved older threads. Codex's code and security
reviews both completed on this head, followed by the connector's clean thumbs-up
reaction. The [review summary](https://github.com/frathe/picfetch/pull/22#issuecomment-5653103344)
records both completed reviews. Lead assessment of the tooling, archive checks,
workflow integration and deferred WinGet scope found no confirmed defect.

The [hosted CI run](https://github.com/frathe/picfetch/actions/runs/34755687251)
passes validation, all four Linux race partitions, Windows tests and both macOS
native-guard jobs. This closes the implementation session's native Linux/amd64
verification gap. The signed release archives and native MSIX bundle checks
still need their release/Store runs; PR CI does not produce those artifacts.
Their qualification remains in `todos.md`.

Local follow-up verification:

- `go test -tags no_emoji -race ./scripts/updaternotices ./scripts/msixstage
  ./scripts/wingettag -count=1` passes: 3.427s, 5.726s and 1.706s respectively.
- `go vet ./scripts/updaternotices ./scripts/msixstage ./scripts/wingettag`
  and `git diff --check` pass.
- GoLand inspections of `main.go`, `artifacts.go` and `main_test.go` in
  `scripts/updaternotices` complete without findings, including weak warnings.
- The full suite stays in native GitHub CI under the review-loop exception;
  no broad local race suite is duplicated.

Route: Thin documentation follow-up, updating this evidence record and
`todos.md`. No code change or new test is needed. Final checks and any later
review dispositions are recorded on [PR #22](https://github.com/frathe/picfetch/pull/22)
so its latest pushed commit can complete a fresh review without another
evidence-only commit restarting the checks.

Delegation: one read-only Scout collects Qodana and CodeQL artifacts while the
Lead checks threads and implementation. G1 bounded report collection; G2 retained
API JSON and post-suppression SARIF; G3 zero repository edits; G4 independent
report discovery; G5 Lead has not loaded the artifact details. Rule S scripts
extraction; the Scout follows run associations and artifact locations. Assessment
and fixes remain with Lead. The single Scout exceeds Thin's default zero-spawn
budget to collect independent external evidence concurrently. Actual: one spawn,
zero code fixes, zero local full-suite runs; final GitHub checks remain required.

### Release-note follow-up on `0c3bf8c`

Ronin restarted the loop after adding release copy and the Trane shrink-ray
illustration. The Lead confirmed that `scripts/releasenotes` preserves the
relative image path, including when writing `.github/release-notes.md` and
publishing its body on GitHub. The image now uses an absolute URL pinned to
the commit that added it. An HTML image retains descriptive alternative text
for GitHub while the existing Store plain-text conversion omits the illustration.
The incomplete optimization sentence, "Hissen" typo and trailing whitespace
are also corrected. The image bytes and release tooling remain unchanged.

Route: Thin, `todos.md` plus this record. Verification: generated release-note
preview preserves the pinned URL and corrected wording and excludes open work;
GitHub's Markdown API retains the image; downloading the raw URL and `cmp`
confirms identical bytes. `go test -tags no_emoji ./scripts/releasenotes
./scripts/storepublish -count=1` passes (0.237s and 1.065s), as does
`git diff --check`. The initial Store test attempt could not bind its local
HTTP server inside the sandbox; the rerun with loopback access passes. No Go
files changed, so the earlier GoLand inspection evidence still applies.

The existing read-only Scout collected fresh Qodana/CodeQL artifacts for
`0c3bf8c`; both contain zero results, with no open PR scanning alerts. Reuse
the same bounded evidence task for the follow-up head. Final review and CI
evidence remains on PR #22 as described above; release qualification is unchanged.
