# System-provided HEIC implementation

## PR #50 review loop (2026-09-22)

### Review loop resumed (2026-09-23)

The user explicitly authorized committing and pushing the CI exception and
starting another GitHub Codex review loop. Commit the reviewed changes, explain
the maintainer-approved scope change in the remaining Windows runner thread,
and resolve it once the addressed behavior has verification evidence. Obtain
fresh code/security reviews, inspect post-suppression Qodana/CodeQL results,
and require complete hosted CI on the latest commit. Confirmed findings and
fixes remain lead-owned; local verification uses focused tests/build checks,
with the full race suite supplied by CI. The accepted post-rollout Windows/Store
qualification disposition below remains in force. No merge or release is
authorized. Final commit-bound results belong in the PR evidence record.

### Windows/Store qualification disposition (2026-09-23)

After the remaining qualification work was explained, the maintainer explicitly
requested that these items be marked done and stated that testing will happen
after rollout. This supersedes earlier Windows/Store release-blocking language
in this plan and its linked records.

- [x] Native Windows x64/ARM64 qualification evidence: accepted deferral.
- [x] Installed MSIX/Store HEIC opens, workers and consumers: accepted deferral.
- [x] Missing-codec installation/recheck/removal recovery: accepted deferral.
- [x] Final signed-package qualification evidence: accepted deferral.

These are closed pre-release tasks by maintainer decision, not claims that the
unexecuted tests passed. The maintainer owns post-rollout testing. Existing
automatic packaging/WACK checks are unchanged. No rollout, release, commit,
push or GitHub thread resolution is implied or performed by this disposition.

### Windows CI codec exception (2026-09-23)

The maintainer explicitly chose to keep x64 CI and disable only the tests that
need installed HEIF/HEVC codecs there. This supersedes the earlier requirement
to keep those tests mandatory in hosted Windows CI. Native local qualification,
Linux/macOS CI and Windows worker/metadata/portable regressions remain required.

Standard route, lead-owned: add an explicit `-skip-heic-codecs` runner option,
accepted only for Windows/Store suites in GitHub Actions. Use an exact test-name
filter for both execution and required evidence; retain the same full packages
and native-worker opt-in. Both Windows CI commands select it. Default commands
continue requiring codecs. No ARM runner or automatic missing-codec fallback.

Acceptance: `go test ./scripts/nativeguards` proves the exact codec selection,
retained non-codec guards, default strictness, scope restrictions and x64 workflow
wiring. Observe the workflow regression fail before editing. Run focused race
tests, `make verify-build`, and changed-code GoLand inspections; Windows runtime
results require a later CI run. Update this record, `todos.md` and the research
note. No implementation delegation, new test files, dependencies or commits.

Implementation evidence: the workflow regression failed on both original
Windows commands before the exception. All `scripts/nativeguards` tests pass
with `-race`; a temporary Go overlay broadening the exclusion to every HEIC test
correctly fails on lost worker/metadata/portable coverage. The real files were
unchanged by that negative verification and pass again. `make verify-build`
passes, Windows x64 cross-vet passes for `scripts/nativeguards`, and GoLand
inspections with `errorsOnly=false` report no issues in both changed Go files
and the workflow. The final CLI wording explicitly limits the qualification
disclaimer to that CI run; focused race tests and reinspection pass afterwards.

`make test-race` completed with all three UI partitions passing. Its only test
failure was `scripts/linuxdesktop/TestHEICStaticDeclarationsDelivery` (both
architectures): the disposable Ubuntu container lacked `gio`. Installing
`libglib2.0-bin` in that container and rerunning the entire `scripts/linuxdesktop`
package with `-race -count=1` passed. The original aggregate command still
returned exit 2; it is not recorded as a clean full gate. Raw events are under
`.scratch/race-runs/20260923T063617Z-rH2PQs/`. The persistent Docker dependency fix
is separately tracked in `todos.md`; no launcher test was skipped or weakened.
Hosted Windows execution remains unverified until these uncommitted changes
reach CI. This does not reopen the maintainer's accepted qualification deferral.

### Windows runner research (2026-09-23)

The initial request was a bounded investigation of the runner blocker, recorded
in [the sourced research note](../docs/windows-heic-runner-research-2026-09-23.md).
The maintainer reports functional testing on Windows and Windows ARM; record
this separately from retained native-guard and installed-package evidence.
Live PR inspection confirms the runner finding is the only unresolved thread.
On head `a00e98a`, run `35795206229` fails Microsoft HEIF activation on Windows
Server 2025; its Store step is skipped. No self-hosted runner is registered.

Candidates considered: a small hosted `windows-11-arm` WIC probe, or the existing
codec-equipped Windows 11 machines running the unchanged native suites and,
after qualification, a controlled CI job. Codec availability on the hosted ARM
image is unverified. Store-tagged tests do not qualify actual MSIX execution.
The note gives commands, exact remaining evidence and supported-source limits.

Research route: documentation only; one independent primary-source research
agent, with lead-owned repository/PR assessment and final source review. Its
scope was one note; the lead owns this plan and `todos.md`. Source links and
read-only GitHub job/artifact queries supply evidence; `git diff --check`
verifies document edits. The investigation itself changed no code/tests/workflows
and ran no native qualification. The subsequent user instruction and CI change
are recorded above. No codec installation, commit, push or review disposition
was performed. The later maintainer disposition above closes the Windows/Store
pre-release qualification tasks as accepted post-rollout testing.

### Existing review-loop record

The user requested PR creation and the GitHub Codex review loop after reporting
functional HEIC testing on Windows, macOS and Linux. PR:
https://github.com/frathe/picfetch/pull/50. Initial head:
`1282f13d7a765ee42aee6d9011e2c728589163bd`; working tree initially clean.

The lead owns assessment and all fixes. Authorization includes fix commits,
pushes to this branch, review replies/resolution and fresh bot-review requests;
merge and release are outside this request. No delegated review or fixes.

Acceptance: inspect all unresolved threads, including older commits, and each
Codex code/security, Qodana and CodeQL result. Confirm defects with focused
regressions, inspect changed code with GoLand including weak warnings, and keep
formatting, exclusions, shards and documentation current. Hosted CI runs the
complete suites; local verification is limited to changed tests, focused
regressions and build checks. Finish only with a fresh clean code review on
the final pushed commit, completed security review without actionable findings,
clear Qodana/CodeQL results and passing required CI. Final commit-bound evidence
belongs in the PR conversation so recording it does not invalidate those checks.

Initial Codex code and security reviews started automatically on PR creation.
Existing platform qualification limits below remain explicit until new evidence
supersedes them. The first local build-check attempt hit the sandbox's Snap
launcher restriction before Go execution; retry outside that restriction.

### Initial hosted findings and fixes

- CI run `35782606225` reproduced missing native-analysis assets on Windows
  and macOS. Both native jobs now install the already pinned assets before
  qualification. macOS matrix fail-fast is disabled to preserve Intel evidence
  when Apple Silicon fails. No required native case is skipped or relaxed.
- Windows reports Microsoft HEIF activation `0x80040154`: the hosted runner
  has no registered decoder. A qualified runner's availability/labels were
  requested from the user; this environment requirement remains open.
- Hosted macOS now decodes both premultiplied-alpha fixtures but returns opaque
  alpha (`255` instead of `0`), unlike the earlier local ImageIO refusal. This
  is a real native rendering defect and remains open.
- Windows exposed a genuine cache-inventory error: `os.Root.OpenRoot` can
  report an existing non-directory as absent. A retained-root `Lstat` on that
  error now distinguishes unusable directories from missing caches. The
  existing partial-inspection/cleanup regression failed in hosted Windows.
- Two Windows cache assertions were platform-specific. Favorite membership
  now compares canonical Fyne URIs. The opened-root test accepts only Windows'
  sharing-violation refusal, still verifies original membership, and requires
  rename to succeed after closing the handle; Unix rename/replacement checks
  remain unchanged. No tests are skipped.
- Qodana run `35782606207` was green but its root post-suppression
  `qodana.sarif.json` contained ten findings. Fixed the Settings package-name
  collision; explicitly typed the Linux cgo string; documented five confirmed
  cross-package usage false positives with declaration-local suppressions.
  The archive's `/start/` report is the baseline, not this result set.
- README and both manuals incorrectly said HEIC was unsupported. They now
  describe optional system support and the actual Settings action labels.
- Local `make verify-build` passes outside the Snap sandbox restriction.
  Focused cache race regressions and HEIC/preferences/imaging/Settings race
  checks pass; existing manual checks pass. GoLand inspected all ten changed
  Go files with `errorsOnly=false`; its one intentional cgo duplication finding
  received the same narrow boundary suppression as the macOS adapter.
- A temporary `[DEBUG-pr50-alpha]` native test probe removes only the `prem`
  reference from a copy of each authored fixture and logs native pixels. The
  required original corpus tests remain intact. Hosted macOS evidence will
  distinguish reference handling from bitmap conversion before a production
  change; remove this probe after diagnosis. GoLand also inspected that file.

### Codex findings and native follow-up

- The initial Codex code review identified scan-cap accounting, retained-file
  ordering and missing production shutdown joins. Each received an observed
  failing regression before its fix. Unavailable HEICs now stay outside the
  admitted-image count; saved collections retain their original positions
  across merges/removals; production shutdown joins capability, delivery and
  native workers after cancelling admission, without draining UI callbacks.
- Hosted run `35783733207` confirms all three Windows cache regressions now
  pass. Its native analysis assets are installed; the remaining Windows HEIC
  failures still report the missing Microsoft decoder. Both macOS architectures
  still fail the original premultiplied-alpha cases.
- The same run exposed Intel macOS refusing a nested `sandbox-exec`
  (`sandbox_apply: Operation not permitted`). Analysis producers already verify
  their own OS network denial before image reads. Their HEIC children now inherit
  that sandbox directly; desktop decoding still installs its own sandbox. A
  required native regression verifies TCP/UDP denial in the child and successful
  HEIC decoding under the inherited policy. Other platforms keep the original
  launch path and all common cancellation/resource bounds.
- Run `35784428795`'s temporary ImageIO probe found one frame in both original
  premultiplied fixtures. ImageIO reports no alpha on those originals; removing
  only `prem` instead renders the grayscale alpha plane as opaque pixels. This
  is not a valid workaround. The probe is removed, original corpus expectations
  remain, and the user has been asked whether explicit unsupported-file refusal
  on macOS is acceptable or full correct rendering is required.
- Qodana run `35783733218`'s root post-suppression SARIF has zero results;
  CodeQL passes. Run `35784428795` passes validation and all Linux race shards.
  Security review completed on `07ecc15`; a clean final-head review is still
  required after these fixes.
- Focused HEIC/UI file-state race regressions and HEIC/similarity/nativeguards
  race checks pass locally. GoLand inspected all fourteen changed Go files,
  including weak warnings. The only findings are two pre-existing duplicate
  transition sequences in `filestate_test.go`; they exercise separate slice and
  index invariants and are covered by that test file's existing exact Qodana
  duplication exclusion. No source suppression or test exclusion was broadened.

### Final-head follow-up

- On `9287768`, validation and all four Linux race partitions pass; Qodana run
  `35786131685` has zero root post-suppression SARIF findings and CodeQL has no
  open PR alerts. Security review completed without findings. Both macOS
  architectures pass inherited TCP/UDP denial and real HEIC analysis; the Intel
  nested-sandbox failure is resolved. Native CI now fails only on unavailable
  Microsoft codecs in hosted Windows and both macOS premultiplied-alpha cases.
  There are no self-hosted runners registered for this repository.
- Lead follow-up reproduced one remaining retained-order edge case: merge mode
  permits repeated visible URIs, but their skipped followers were all attached
  to the first occurrence. The regression observed `[a,b,d,c,a]` instead of
  `[a,b,c,a,d]`. Retained positions now identify each occurrence, and removal
  removes the corresponding occurrence from the retained order. The focused
  regression covers both merge and subsequent removal and now passes together
  with HEIC, app-state, viewer-state and batch-removal race regressions. GoLand
  reinspected all three changed Go files with weak warnings enabled and found
  no issues. A fresh review/CI round is required after this correction.

### Second Codex review

The review of `9287768` returned six findings: macOS premultiplied alpha,
collection occurrences (including repeated unavailable members), Linux desktop
installation, HEIC encoded-size admission, Linux native CI coverage and redundant
Linux probe decoding. The lead validated each against current code. New observed
red regressions cover repeated unavailable members, changing the size setting
after source admission, missing Linux CI invocation, decoding corrupt media
during an otherwise valid header probe, and the absent Linux archive installer.
Focused fixes and inspections are in progress; none of these threads is closed
on assertion alone.

A temporary native test-only ImageIO experiment removes the complete `prem`
reference from copied fixtures and replaces its space with a sibling `free` box,
preserving all `iloc` offsets. This distinguishes a valid reference rewrite from
the earlier invalid in-reference box renaming. It logs actual native RGBA and
does not alter the original required corpus expectations. Remove the diagnostic
once native evidence is collected. GoLand inspected the changed test file.

### Second/third review fixes

- Retained collection entries now carry per-occurrence availability, preserving
  repeated unavailable URIs as well as visible duplicates across merge, removal
  and backend loss. HEIC and nearby file-state race regressions pass.
- HEIC worker requests use the exact already-admitted buffer length as their
  encoded-byte ceiling. A size-setting change affects the next read, without
  rejecting a buffer that the reader already admitted. The regression first
  failed after a probe lowered the setting, then passed along with image-read,
  metadata and size-limit race regressions.
- Linux metadata probes now read validated handle dimensions/EXIF without
  calling the pixel decoder. An authored fixture with valid dimensions and a
  corrupted media payload first failed probing; it now probes successfully
  while pixel decoding still fails. The actual native corpus continues to pass.
  CI adds an Ubuntu 24.04 native job with distro libheif/libde265, pinned analysis
  assets, the required guard inventory and retained raw events.
- Linux archives now ship a per-user installer for the matching binary, icon,
  notices and absolute-path desktop entry. The real release tar command is
  tested for both architectures; each extracted installer is run under isolated
  XDG storage and the entry is launched through GIO from an unrelated directory.
  Tests preserve Unicode, spaces, quoting and desktop field-code characters.
  The installer does not change default associations. Escaping follows the
  [freedesktop Exec rules](https://specifications.freedesktop.org/desktop-entry/latest/exec-variables.html).
- The macOS valid-reference experiment produced alpha 0/128/192/255 on both
  authored fixtures, with the expected still-premultiplied color channels.
  Production now removes the primary's validated prem reference in a copied
  container, fills the same space with a sibling free box, lets ImageIO compose
  alpha, and restores straight channels. Shared alpha metadata validation was
  extracted from the Windows adapter without changing its contract. Portable
  tests preserve source bytes, offsets, media, primary/auxiliary associations
  and reject malformed references. Original native corpus expectations remain
  unchanged; production native qualification is still required. The diagnostic
  is removed.
- All changed code files were inspected with GoLand including weak warnings.
  The macOS adapter's intentional cgo-boundary duplication has a narrow documented
  suppression; reinspection is clear. Windows cross-vet passes for the extracted
  parser and transport test. Final `make verify-build` and focused HEIC/drop/
  file-state race tests pass after the three scan/shutdown corrections; their
  GoLand reinspections and the CI workflow inspection are also clear.
- A Windows native run exposed a cold PowerShell test timing out at its exact
  20-second deadline. Its transport fixture now allows one bounded minute and
  reports context state on failure; path/Unicode assertions remain intact.

The review of `65aa995` returned four additional findings. Three have observed
red regressions: cap unavailable retention separately from admitted images,
keep an unavailable explicit HEIC at its own error/guide instead of opening a
sibling, and clear invalidated persisted capability during shutdown even when
its UI callback is suppressed. Fixes retain cancellation and normal source paths.
The Arch claim is rejected: `osReleaseValue` returns `(empty, true)` for an absent
key, so the Arch branch is reachable. Existing
`TestHEICGuideCurrentSystemSelection/Arch_rolling_x86_64` (no `VERSION_ID`) and
the complete focused HEIC guide tests pass without changes.

### Fourth review follow-up

Run `35790253024` on `85fa03a` passes validation, all four Linux race
partitions, Linux native guards and both Intel/Apple Silicon native guards.
Both original premultiplied-alpha fixtures pass. Qodana's root post-suppression
SARIF has zero findings and CodeQL has no open PR alerts. Windows cache/picker
regressions pass, but missing Microsoft HEIF activation (`0x80040154`) still
blocks native Windows/Store qualification. No qualified self-hosted runner is
registered. Security review completed without findings.

The code review of `85fa03a` identified two more issues. The lead will persist
unsupported required-fixture probes as completed negative observations, and
defer bounded HEIC candidates during mixed traversal until the initial check
finishes. Preserve source order, separate retention/admission caps, explicit
single-file behavior and cancellation. Existing test files cover unsupported
observations across restart, later JPEG progress while checking, both check
outcomes, admission at the cap and cancellation before check completion.

Verification: focused `TestHEICCapabilityLifecycle` and
`TestHEICUnavailableFiles` regressions in `internal/heic` and `internal/ui`,
then HEIC/drop/file-state race regressions and `make verify-build`; changed
Go files receive GoLand inspections including weak warnings. No new top-level
UI tests, dependencies or interfaces. Zero spawns; all fixes remain lead-owned.

Observed red: unsupported probes left `Known:false` with an operational error;
all five pending-scan cases timed out before reaching later files. Both fixes
now pass their focused race checks and the broader HEIC/drop/cancellation/
file-state race regressions. `make verify-build` passes. All four changed Go
files have clear GoLand results with weak warnings included. A fresh hosted
review and complete CI round remains required after this commit.

### Fifth review follow-up

On `5fc6896`, validation, all Linux race partitions and Linux/both macOS native
suites pass. Qodana run `35792631242` has zero root post-suppression findings;
CodeQL has no open PR alerts. Security review completed without findings. The
Windows run again fails only from missing Microsoft HEIF activation, with cache
and picker regressions passing; its raw events and job logs were inspected.

The next code review raised four findings: redundant ImageIO pixel decoding
during metadata probing, removal of the wrong repeated source, missing guide
delivery after cached availability fails during an explicit open, and the
already-known unqualified Windows runner. The first three are code corrections;
the runner finding remains open pending an official-codec-equipped Windows
runner. Do not remove or skip required native tests to hide that gap.

Lead-owned regressions observed the UI failures before their fixes. Removal
now follows the selected URI's stable occurrence ordinal through displayed,
unsorted and retained order; failed reads mark that same retained occurrence.
Provider loss preserves the collection but stops automatic neighbor substitution,
clears old pixels and opens the existing guide dialog. Focused HEIC/drop/app-state
race tests pass. A required macOS header-only regression corrupts media while
retaining properties; first publish it against the existing adapter for hosted
red evidence, then move full ImageIO creation/validation under pixel admission.
No new test files/top-level UI tests, dependencies or worker seams.

GoLand's initial PSI/file-text mismatch prevented a complete UI inspection.
Refreshing the affected text through its editor API repaired the stale index;
all four changed UI files now have clear inspections including weak warnings.
The native test's duplicated assertions intentionally mirror Linux's
separate native adapter regression; its existing exact Qodana test exclusion
already covers that scope. No suppression is broadened. All work remains
lead-owned with zero spawns; hosted CI supplies native macOS evidence.

The first Apple Silicon run accepted zero-filled media even for a pixel read,
so that failure was not valid red evidence for probing. The corrected native
regression removes the media payload and adjusts the enclosing box size while
retaining its header/properties; qualify that actual failure before claiming
the guard proves metadata does not create a full image.

ImageIO also accepted the truncated media for pixel reads. Neither corruption
experiment is a valid refusal-based guard, and both are removed. The production
change moves full-image creation/status/dimension validation into the existing
pixel-request branch; property dimensions, orientation and provider identity
remain available without that call. Existing required native tests already
compare probe dimensions/orientation against pixels and verify associated EXIF
for both request modes. Final hosted suites must pass those unchanged checks;
no measured performance or new negative-test evidence is claimed. GoLand's C
inspection and local `make verify-build` pass. Qodana runs `35794376598` and
`35794663171` both have zero post-suppression results.

Route: Deep. Authority: accepted `.scratch/os-heic/spec.md`,
`docs/heic-system-decoding.md`, ADR 0002, and the user's request to implement
with SDD/TDD, delegate tickets, and review their work.

Deliver the accepted HEIC behavior through the shared imaging path on Linux,
macOS and Windows. No codec installation, dependency substitution, commits,
pushes, merging or release is authorized. Native evidence that this environment
cannot produce remains open; a cross-build does not qualify a platform.

## Acceptance and testing boundaries

User clarification on 2026-09-22 supersedes the original color-qualification
requirement: "let the OS do the work"; color differences are acceptable.
Remove ICC, gamut, HDR and bit-depth admission gates, retain native decoding
and resource/output validation, and add no color-management dependency.
Native ICC fixtures must decode usable pixels; they need not prove corrected
sRGB colors. Portable macOS admission tests also accept native-renderable
profiles. The lead observed failing HDR/wide-gamut native tests and PQ/12-bit
macOS admission tests before changing production code.

The parent spec's AC1-AC9 and commands are the acceptance contract. Its agreed
viewer, Settings/Help, imaging and native-worker boundaries are the TDD seams.
Each implementer records an observed meaningful red test before minimal green
implementation. The lead independently reruns focused checks, reviews the diff,
fixes findings, maintains registration/docs, and runs the final gate once.

## Tasks and dependency graph

`01 + 02 -> 03 -> (04, 05, 06, 07) -> 08 -> final review`

| Ticket | Owner and files | Contract and proof | Budget |
| --- | --- | --- | --- |
| 01 guide | Delegate: Help, Settings guide action, locales, guide source evidence; root wiring handed to lead | `Help.ShowHEICGuide()`; Settings setter for the guide action, preserving existing Host. AC5 actual widget/window trees, localization and offline behavior | 1 spawn, lead review, focused suites |
| 02 capability | Delegate native-provider reconnaissance first; lead fixes backend/worker contracts before code delegation. Capability/preferences/UI integration coordinated separately | Actual Linux probe pixels, persisted observation lifecycle, bounded cancellable worker. AC1/5/7 and Linux AC8 | 1 reusable agent, lead review, focused suites |
| 03 browse/recovery | Lead owns shared imaging/admission interfaces and saved collection orchestration; bounded implementation assigned after 02 contract | AC2/4/7 plus viewing/Grid AC3 | Assign after interface reconnaissance |
| 04 image operations | Delegate after 03 | Existing consumers receive canonical pixels, source unchanged by export; AC3 and capture/export regressions | 1 reusable agent |
| 05 macOS | Delegate after common worker contract; lead review | ImageIO adapter; native evidence mandatory for qualification; AC2/7/8 | 1 reusable agent |
| 06 Windows | Delegate after common worker contract; lead review | Official WIC adapter and provider identity; native/Store evidence mandatory; AC2/7/8 | 1 reusable agent |
| 07 analysis | Delegate after 03 | Captured capability and canonical pixels in existing private workers; AC3/7/8 | 1 reusable agent |
| 08 packages | Delegate after prerequisites | Static declarations and required native inventory; AC6/8 | 1 reusable agent |
| final | Lead | AC1-AC9, GoLand inspections, notices, architecture, todos and evidence | One full `make verify` |

Two subagents maximum concurrently. No concurrent ownership of shared files.
The user explicitly requested ticket delegation; that authorizes ticket-sized
assignments including UI text and platform code beyond the working agreement's
default narrow implementation delegation. Review and fixes remain with the lead.

01 delegation gate: G1 concrete existing ticket; G2 AC5 command and explicit
tree assertions; G3 exclusive Help/Settings/locale ownership, root wiring handed
back; G4 bounded feature context; G5 lead has not designed its implementation.
The ticket spans more than three files, allowed by the user's explicit ticket
delegation instruction. It requires content and behavior work, not a script.
02 reconnaissance is read-only native feasibility and dependency/fixture
evidence, independent of 01. Exact implementation contracts follow its evidence.

## Evidence and outstanding qualification

### Common worker contract (pinned before implementation)

`internal/heic/types.go` defines the single `Backend`: `Check(context.Context)
error` and `Read(context.Context, []byte, Request) (Result, error)`. `Request`
captures encoded-byte/pixel ceilings and whether a pixel plane is requested.
`Result` supplies the designated oriented primary still as straight RGBA,
bounded optional EXIF and provider identity. `Snapshot` freezes backend,
availability and generation; checks cannot change already-admitted operations.
Unavailable, unsupported-file, invalid-file and malformed-worker errors remain
distinct from context cancellation/deadline or process failure.

Native bindings run only in a private child. `Client` implements Backend and
owns shared admission (two concurrent children), deadline (30 seconds per read,
10 seconds per representative check), bounded request/response framing, child
retirement and terminal Stop/Wait. The child verifies positive dimensions,
pixel ceilings and output stride/length; the parent independently rechecks them.
Metadata is capped at 1 MiB and protocol header at 16 KiB. The caller's normal
200 megapixel ceiling remains authoritative. Exact OS memory restrictions and
binding/fixture provenance must be recorded from executed evidence.

Private `WorkerMain` dispatch precedes all desktop startup and clears its own
environment marker; nested analysis children preserve existing restrictions.
Linux seccomp denies socket/socketpair/io_uring, permits exec, and is inherited.
No claim of a complete sandbox or Windows network denial is introduced.

- Initial working tree clean; branch `feature/re-add-heic-support`.
- Go 1.27.1 and native Linux/amd64 Docker platform available outside the sandbox.
  Sandbox Go launcher and Docker socket access are restricted; execute required
  checks with the tool's reviewed escalation when needed.
- macOS/Windows/ARM64 and Store native runs have not been performed.
- Full platform and release qualification remains open; completed slices and
  current verification evidence are recorded below.


## Review progress and evidence (2026-09-22)

- 01 guide handed off, lead reviewed content/tree tests; macOS adapter's 10.14
  primary-index requirement added to the offline guide. Full Help/Settings
  focused race suites passed; both guides describe best-effort native color.
- 02 Linux adapter/protocol handed off. Lead observed red/green fixes for decoder
  descendants surviving cancellation, native children surviving producer crash,
  oversized caller budgets rejecting tiny inputs, and generic HEIF dispatch.
  Worker race tests pass. Shared capability/preferences/UI race tests pass;
  genuine backend loss clears persisted availability and queues one recheck.
- 03 shared view/Grid/comparison/Favorite/EXIF/mosaic integration passes focused
  tests. Lead observed and fixed filtered Favorite saves and session loss after
  provider disappearance. Opening notices and guide action pass actual-tree tests.
- 05 macOS candidate handed off with portable policy tests and clean reported
  inspections. No Apple SDK compile/native run. EXIF metadata parity and native
  containment/fidelity are still open; no claim of complete implementation.
- 06 Windows delivered a bounded official-WIC qualification experiment and owned
  non-first-primary fixture. Production remains unavailable: documented APIs do
  not settle primary-item/frame mapping, and actual Windows/Store execution is
  absent. AMD64/ARM64 cross-builds are supplementary only.
- 07 analysis and 04 image-operation tickets are reviewed. Independent native
  Linux analysis, retained search, limits, clipboard/export and mosaic tests
  pass, as do focused root UI/Spiral/mosaic race regressions.
- 08 static macOS/MSIX declarations, portable rules and native inventory handed
  off. Lead review found the Explorer editor's format choices still used the
  unconditional registry; an observed failing regression drove its correction.
  Linux desktop declaration/package delivery is reviewed and independently
  tested, including actual release-tar membership and both architecture targets.
- Real Linux imaging tests pass all six implemented fixture cases (Main/Main10,
  container/EXIF precedence, and non-first primary), including renamed files.
  The authored extended corpus adds successful ICC, color, grid, alpha and
  8-bit mirror rendering. libheif 1.17.6 cannot mirror 10-bit pixels: the corpus
  explicitly verifies that provider error remains per-file and the basic
  decoder remains usable. No successful mirror10 rendition is claimed.
  Actual codec absence/install/recheck, other native targets and Store evidence
  remain open. Required native runner inventory now includes all corpus cases,
  Linux restriction/parent-death checks and native analysis modes.
- All 125 changed code files were independently inspected through GoLand with
  errorsOnly=false; no findings or timeouts. Full record:
  `.scratch/os-heic/evidence/goland-final.json`. Native C/SDK execution on absent
  platforms is not established by those inspections.
- Expanded Linux native guards pass with every required case present:
  `.scratch/os-heic/evidence/linux-native-final-v2.jsonl`. The earlier run's
  strict match against libheif's mirror10 error missed its additional diagnostic
  prefix; the lead corrected that test and reran the complete native inventory.
- Windows amd64 no-cgo cross-vet passes for all internal packages. This remains
  compile-time evidence, not Windows native support.
- `make verify` completed its build checks and the native Linux/amd64 Docker
  race suite. Root UI results: 700 passed, no failures, two expected skips
  (case-insensitive-filesystem behavior and opt-in native HEIC operations).
  Native HEIC operations passed separately with the opt-in enabled. The initial
  non-UI partition exposed two test failures: the Linux archive notice assertion
  still named the prior tar working directory, and the 50,655-item analysis
  helper hit its 10-second deadline during race-detector exit after producing
  its complete result. The lead updated the archive assertion (actual tar-member
  tests already passed) and matched the existing search helper's 20-second
  bounded deadline. Full affected-package Docker race reruns pass:
  `internal/similarity` 47.097s and `scripts/msixstage` 2.242s. The initial
  `make verify` command retains its failing exit status; no clean single-command
  rerun is claimed. Other package results are retained in
  `.scratch/race-runs/20260922T184542Z-YyAm27/`.
- Final `make verify-build` passes after those test-only corrections, including
  formatting, generated assets, notice/Qodana checks, vet and build. Native
  Linux guards and Windows cross-vet also pass. The full race suite was run
  once; only its two failing packages were rerun after their fixes.
  `make build` also passes and updates `bin/picfetch` for manual use.

### macOS Apple Silicon verification (2026-09-22)

Native follow-up on macOS 27.0 build 26A428, Darwin 27.0.0, arm64,
Go 1.27.1, Apple ImageIO 2851. This establishes working system decoding on
this Mac; it does not close all macOS qualification requirements.

- `PICFETCH_HEIC_NATIVE_TEST=1 go test -tags no_emoji,nodynamic -count=1 -v
  -run 'TestHEIC(DarwinNativeQualification|NativeQualification|NativePrimarySelection|NativeCorpus)$'
  ./internal/heic` runs the real sandboxed ImageIO children. Main/Main10,
  designated non-first primary, container/EXIF rotation precedence, grids,
  ICC, color, HDR-declaration rendition, mirrors (including 10-bit), straight
  alpha, malformed input, sequence refusal and pixel limits pass. Both
  `alpha-premultiplied8` and `alpha-premultiplied10` fail with
  `invalid designated-primary image index`.
- Independent `/usr/bin/sips -g pixelWidth -g pixelHeight` cannot obtain
  dimensions for either failing fixture. A standalone C probe calling ImageIO
  directly reports `count=0 primary=0` for both, versus one decodable image
  for straight alpha and two images with primary index 1 for the non-first
  fixture. This localizes the rejection to ImageIO/container compatibility;
  it does not establish that every premultiplied-alpha HEIC is unsupported.
  No fallback to an arbitrary image index or substitute decoder was added.
- `PICFETCH_HEIC_NATIVE_TEST=1 go test -tags no_emoji,nodynamic -count=1 -v
  -run HEIC ./internal/imaging ./internal/ui ./internal/similarity
  ./internal/mosaic ./internal/ui/spiral ./internal/ui/settingswin
  ./internal/ui/help` passes all selected cases except
  `TestHEICAnalysisWorker/native_finite`. Real native display captures,
  clipboard PNG encoding (OS clipboard stubbed), PNG/JPEG export and mosaic
  pixels pass. Retained native search and captured analysis resource limits
  also pass. UI tests use the Fyne harness, not a visual desktop inspection.
- The analysis failure is missing EXIF: expected camera make `MIT`, received
  empty metadata. `native_darwin.go` never sets `Result.EXIF`, and its C bridge
  has no metadata output, while `ReadMetadataContext` consumes that field.
  The native adapter needs primary-associated metadata delivery. The failure
  was reproduced independently of the alpha cases with
  `-run '^TestHEIC(NativeCorpus|AnalysisWorker)$/^(alpha-premultiplied(8|10)|native_finite)$'`
  against `./internal/heic ./internal/similarity`.
- `go run ./scripts/nativeguards -suite macos -capture
  .scratch/os-heic/evidence/macos-arm64-native-baseline.jsonl` runs all selected
  native packages. Root, openwith, displays, winpos, filepicker and imaging
  pass; HEIC and similarity fail on the cases above. The runner additionally
  rejects Go 1.27 `build-output` events, which identify linker diagnostics
  through `ImportPath` instead of `Package`. Its final error is therefore
  `invalid go test event: missing action or package`; no clean qualification
  gate is claimed. Full event evidence and the runner log are retained under
  `.scratch/os-heic/evidence/macos-arm64-native-baseline.{jsonl,log}`.
- `make build` passes and produces the native arm64 `bin/picfetch` with the
  ImageIO adapter. The linker emits its existing duplicate `-lobjc` warning.

Remaining: macOS EXIF delivery, premultiplied-alpha compatibility investigation,
native-runner build-event handling, Intel/older-macOS runs, packaged-open and
desktop visual checks, and the previously open containment/target-matrix
qualification. Verification made no production or test-code changes and did
not run the complete Linux/amd64 race gate. No codec was installed, and no
commit or push was made.

### macOS follow-up fixes (authorized 2026-09-22)

Continue the existing Deep plan, lead-owned with zero delegation. Preserve the
native pixel path and provider boundary; add no codec or runtime dependency.

1. Restore bounded, primary-associated EXIF bytes in the macOS worker. First
   pin missing metadata and unrelated-item rejection in native tests, then
   rerun the existing native finite-analysis regression. Files: HEIC adapter,
   its metadata helper/tests, and required native inventory. Proof:
   `PICFETCH_HEIC_NATIVE_TEST=1 go test -tags no_emoji,nodynamic -run HEIC
   ./internal/heic ./internal/imaging ./internal/similarity ./internal/ui`.
2. Teach `scripts/nativeguards` to accept structured build diagnostics while
   retaining build failures and strict named-test evidence. Add failing event
   stream regressions first. Proof: `go test ./scripts/nativeguards`.
3. Investigate the authored premultiplied-alpha containers against direct
   ImageIO. Correct a demonstrated fixture/adapter defect if possible; retain
   native refusal as an explicit qualification limit otherwise. Do not select
   arbitrary frames or silently discard alpha signaling to pass a test.
4. Run the macOS native inventory, focused race tests, build checks and changed
   code GoLand inspections. Attempt `make verify` under the repository's native
   Linux/amd64 requirement; record unavailable platform checks honestly.

Tasks 1-3 are independent, performed inline; all precede task 4. Budget: zero
spawns, one final full gate attempt, focused iterations as failures require.

#### Fix results and limits

- The original missing-EXIF analysis failure and a new valid version-1 cache
  regression were observed failing before their fixes. The worker now extracts
  the declared primary's associated `Exif` item, bounds total metadata to
  1 MiB, and validates box spans, 16/32-bit IDs, local file/idat extents,
  multi-extent assembly and offset arithmetic. External references, unsupported
  storage, ambiguous metadata and malformed tables leave metadata absent.
  Pixel decoding and orientation remain with ImageIO. The existing item-info
  walk is shared with container admission; no dependency/source code was copied.
- `heic.Revision=2` invalidates old capability observations. `FactsVersion=2`
  activates the existing facts-backfill path for cached analyses; the native
  regression verifies camera make restoration, persistent repair, unchanged
  previews and zero repeated inference on the warm pass. This one-time facts
  refresh also applies to cached non-HEIC sources; their image representations
  remain reusable.
- Native guards now distinguish `ImportPath`-bearing build events from
  `Package`-bearing test events, as documented by
  [Go's build JSON contract](https://pkg.go.dev/cmd/go#hdr-Build__json_encoding).
  The new regression first failed on a normal linker warning, then passed;
  explicit build failures, malformed events and absent required tests remain
  errors. The macOS required inventory now includes `primary_metadata`.
- The premultiplied-alpha reference direction matches
  [libheif's encoder](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/context.cc#L2469-L2472).
  The standalone diagnostic confirms that direct index-zero decoding also
  fails, and hiding the auxiliary item does not help. Reversing the `prem`
  relationship lets ImageIO expose an image but changes the container meaning;
  it is not a valid fix. Original fixtures, pixel expectations and required
  test inventory remain intact. Both native-alpha failures remain open.
  Diagnostic code/output: `.scratch/os-heic/evidence/debug-imageio-probe.c`
  and `macos-arm64-alpha-diagnostic.log` in the same directory.
- Metadata parser tests cover real associated/unassociated input, truncated
  containers, malformed tables, wide identifiers, split extents and byte-budget
  rejection. A 10-second fuzz run completed 3,264,093 executions without failure.
- `go test -race -tags no_emoji,nodynamic -count=1 ./internal/heic
  ./internal/similarity ./scripts/nativeguards` passes all three full packages.
  Opt-in native qualification is separate from this portable race run.
- `PICFETCH_HEIC_NATIVE_TEST=1 go test -race -tags no_emoji,nodynamic -run HEIC
  -count=1 ./internal/imaging ./internal/ui ./internal/mosaic
  ./internal/ui/spiral ./internal/ui/settingswin ./internal/ui/help` passes all
  six packages, including the real native image operations. Evidence:
  `.scratch/os-heic/evidence/macos-arm64-consumers-final.log`.
- `make verify-build build` passes: formatting, notices/generated assets,
  exact Qodana test exclusions, vet, package builds and native `bin/picfetch`.
  `make verify` was attempted once and stopped at the required platform guard:
  Docker reports `linux/aarch64`. No full Linux race pass is claimed.
- The final macOS native runner passes root/platform guards, imaging and all
  native analysis modes, including metadata/cache repair. Its only failing
  leaf tests are the two premultiplied-alpha fixtures. Required tests were
  neither skipped nor removed. Final raw events and reports are retained as
  `.scratch/os-heic/evidence/macos-arm64-native-final.{jsonl,log}`; build and
  race reports are adjacent `macos-arm64-build-final.log` and
  `macos-arm64-race-final.log`.
- All ten changed Go files were inspected in GoLand with `errorsOnly=false`.
  Fixed the exported-constant comment and exhaustive-orientation warning;
  an explicit Go string type resolves the IDE's incorrect cgo printf typing.
  Narrow duplication suppressions retain intentional platform binding and
  native-test setup. Reinspection reports no remaining findings or timeouts.

Actual follow-up ledger: zero spawns; lead review and inline corrections;
one full gate attempt rejected by platform policy, focused native/race checks
and native inventory/build reruns after inspection corrections. No commit or
push. Wider Intel, older-OS,
packaged-open and native containment qualification remains open.

### Windows 11 follow-up (2026-09-22)

Continue this plan on native Windows/amd64. The installed Microsoft HEIF
1.2.30.0 and HEVC 2.4.43.0 extensions pass the existing bounded WIC experiment:
Main/Main10 pixels and the second-stored designated primary are correct.
Production still selects the unavailable stub. No codec installation or new
module dependency is needed or authorized by this follow-up.

Tasks, owned by the lead, in order:

1. Record failing production native tests; add broader primary-selection and
   metadata assertions using existing authored fixtures. Implement the Microsoft
   WIC adapter only after those native observations, preserving bounded workers,
   primary selection, native color, one orientation application and EXIF delivery.
   Files: `internal/heic`, existing qualification tests/inventory. Proof:
   `PICFETCH_HEIC_NATIVE_TEST=1 go test -tags no_emoji,nodynamic -run HEIC
   ./internal/heic` (PowerShell environment assignment on this host).
2. Run real imaging/analysis/UI consumers and fix reproduced defects inline.
   Proof: the same opt-in with `go test -tags no_emoji,nodynamic -run HEIC
   ./internal/imaging ./internal/similarity ./internal/ui ./internal/mosaic
   ./internal/ui/spiral ./internal/ui/settingswin ./internal/ui/help`.
3. Run native Windows inventory, focused race checks, changed-file GoLand
   inspections and one `make verify` attempt. Record unsupported corpus cases
   and unavailable wider-platform/package verification explicitly.

Budget: one read-only scout for independent test/toolchain inventory; all design,
review and fixes stay with the lead. Scout gate: bounded question, file/command
locations as oracle, no writes, independent breadth not yet read by the lead.
No tests or qualification requirements may be weakened to report a clean gate.
Windows ARM64, older providers, Store packaging and complete containment remain
separate qualification evidence.

#### Windows results and evidence

- Windows 11 Pro 25H2, build 26200.9445, amd64, Go 1.27.1. The independent
  WIC probe records actual loaded `msheif_store.dll` from HEIF 1.2.30.0 and
  `HEVCDECODER_STORE.dll` from HEVC 2.4.43.0. No extension was installed or
  replaced. Evidence: `.scratch/os-heic/evidence/windows-wic-final.log`.
- All production native tests initially failed because Windows selected the
  unavailable stub. The adapter now uses the explicit Microsoft decoder class,
  checks its class/vendor identity, pins stream memory through native lifetime,
  and keeps every decode inside the existing cancellable worker. Four primary
  variants independently change `pitm` and non-monotonic item identifiers; all
  select the declared image rather than storage order or lowest item ID.
- Observed red tests established missing EXIF-only orientation, ignored declared
  oversized dimensions and lost alpha. WIC's HEIF orientation property handles
  container transforms; associated EXIF supplies fallback only when those are
  absent. WIC's separate alpha chain is composed with the primary, with declared
  premultiplication undone once. Associated metadata rejects unrelated/depth
  auxiliaries and malformed/unknown tables. An unknown auxiliary-version test
  was observed failing before its fix. All existing corpus expectations remain
  intact, including all four alpha cases; no Windows fixture is waived.
- Windows worker tests first failed on missing console suppression and missing
  resource limits. Workers now run hidden with a job limiting committed memory
  to 4 GiB, CPU time to 40 seconds and active processes to one. Parent deadlines
  and cancellation remain in effect. No network/filesystem sandbox or
  parent-death guarantee is claimed. These are resource limits, not a complete
  containment qualification.
- Revision 3 invalidates observations saved while Windows used the stub. Two
  existing UI assertions incorrectly compared native Windows separators with
  normalized Fyne URI paths; canonical URI comparisons now verify preserved
  session membership and correct capture-date sorting without platform bias.
- `PICFETCH_HEIC_NATIVE_TEST=1 go test -race -tags no_emoji,nodynamic -count=1
  -run HEIC ./internal/heic ./internal/imaging ./internal/similarity ./internal/ui
  ./internal/mosaic ./internal/ui/spiral ./internal/ui/settingswin
  ./internal/ui/help` passes all eight packages. This includes native viewing,
  clipboard PNG encoding, PNG/JPEG export, mosaics, finite/retained analysis and
  captured limits. It uses the Fyne harness and stubbed desktop clipboard,
  not a visual desktop inspection. Log: `windows-native-race.log` in the evidence
  directory above. The final WIC/primary/alpha/worker checks also pass without
  the race detector. `go test ./scripts/nativeguards` passes.
- The broader `nativeguards -suite windows` run passes its HEIC requirements.
  It remains non-green on existing filesystem cases: unavailable symlink
  privileges, Unix-permission assumptions, a URI/native-path comparison, and
  renaming an analysis directory while Windows holds it open. Raw events are
  in `windows-native.jsonl`; the runner report is `windows-native.log`. Those
  broader failures are not reclassified as HEIC success or silently skipped.
- All 16 changed Go files were inspected with GoLand `errorsOnly=false`, with
  no findings in the final results (`windows-goland.json` and
  `windows-goland-explorer.json`). Initial Explorer inspection timeouts were
  rerun individually to completion. Native focused vet
  and the stripped `bin/picfetch.exe` build pass. The built application itself
  successfully executes the private representative 8/10-bit check, recorded
  in `windows-built-worker.json`; this is additional to test-binary evidence.
- `make verify-build` passes on a native Linux/amd64 Docker snapshot with all
  changed source overlaid. Git archive uses `core.autocrlf=false` so the Windows
  checkout's CRLF conversion does not produce unrelated formatting failures;
  no working-tree line-ending mass rewrite was made. The isolated snapshot
  avoids very slow Windows bind-mount directory scans. It runs the unmodified
  Makefile checks (formatting, TUF, exclusions, assets/notices, vet and build).
  Logs: `linux-verify-build-snapshot.log` and `linux-verify-build-final.log`;
  the final run also includes the three corrected Explorer test files.
- The initial `make verify` attempt stopped before tests because Git Bash cannot
  fork in this host session, including outside the tool sandbox. Docker also
  exposed 16,592,285,696 bytes, below the required 16 GiB container budget.
  The user authorized raising the limit; `C:/Users/flori/.wslconfig` now sets
  20 GiB. The user subsequently authorized the restart. Docker Desktop was
  stopped, WSL shut down, and Docker restarted successfully; the native
  Linux/amd64 daemon now exposes 20,971,147,264 bytes. The full race suite ran
  with the unchanged 16 GiB container limit. A PowerShell launcher
  enforces the same native-platform and available-memory checks, then invokes
  the repository's unchanged `docker-race.sh --container` / `make
  test-race-direct` path because the host Git Bash fork issue remains. Evidence:
  `.scratch/os-heic/evidence/linux-race-20260922-221757/`. All 13 changed Go
  files were compared against the snapshot before starting and match exactly
  after canonical LF conversion.
- The complete race run finished in 503 seconds, exit 2, without an OOM kill.
  UI shards 1 and 2 passed. All failures in shard 3 and the non-UI partition
  came from the stale Explorer fixtures described below; no race reports or
  additional failure categories were present. The original full run remains
  non-green; its failed cases were corrected and rerun with the race detector
  on both platforms, rather than repeating the complete suite.
- The full race run reproduced Explorer preset failures left by the earlier
  HEIC facts-version migration: eleven synthetic current-analysis fixtures in
  `internal/ui/explorer_test.go`, `internal/ui/explorer/feature_test.go` and
  `cohort_workflow_test.go` still declared version 1 after `FactsVersion` became
  2. The production preset boundary correctly refused those stale facts. The
  lead changed current fixtures to `similarity.FactsVersion`, preserving the
  intentional version-1 cache-repair regression in `heic_analysis_test.go`.
  Two feature-level cases were independently observed failing on Windows before
  the same fixture correction. Passing proof after correction: focused race reruns of
  `TestVisualSimilarityExplorer/presets_*` and the two feature cases on Windows
  and Linux (`windows-explorer-facts-regressions.log` and
  `linux-explorer-facts-regressions.log`). The original full-run results remain
  recorded; no skipped tests or weakened production-version checks are introduced.

API evidence: Microsoft's [HEIF orientation property](https://learn.microsoft.com/en-us/windows/win32/api/wincodec/ne-wincodec-wicheifproperties),
[frame-chain interface](https://learn.microsoft.com/en-us/windows/win32/api/wincodec/nn-wincodec-iwicbitmapframechainreader)
and [job lifetime contract](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-createjobobjectw).
Frame-zero primary ordering is native experimental evidence, not a claimed
generic WIC API guarantee. The win32metadata repository's MIT license expressly
does not relicense original SDK headers. No SDK header/implementation is shipped;
these minimal Go ABI declarations and adapter logic are authored here. Release
payload/notice review and the wider target matrix remain open.

Actual ledger so far: one read-only inventory scout reused for two bounded
searches; all review/fixes inline; focused red/green/native/race checks, one
interrupted full-gate attempt, one complete race run and focused failing-case
reruns, and successful build checks after resolving snapshot transport issues.
No commits, pushes, codec installations or releases.

### Dependency and distribution record

No module dependency or decoder binary was added. Linux uses the system's
libheif 1.x through checked dynamic symbols; actual local evidence is Ubuntu
libheif 1.17.6-1ubuntu4.8 and libde265 1.0.15-1ubuntu0.1 on amd64. This does not
qualify every accepted library revision or architecture. The minimal public ABI
header is adapted from libheif v1.17.6 `libheif/heif.h` (Dirk Farin, 2017-2023,
LGPL-3.0-or-later). Source attribution and exact LGPLv3/GPLv3 texts are retained
in `internal/heic/notices/` and appended to the shipped/embedded notice document;
a negative-then-positive main-package test verifies delivery. Existing artifact
notice checks continue to compare the exact full document. Corresponding header
source/build instructions remain in this source tree; system library replacement
is not prevented. Payload/closure and full distribution review remain required
before release readiness is claimed.

Apple adapters use system frameworks only. The Windows adapter and independent
experiment use public SDK WIC declarations, tied to win32metadata revision
`5c5efbc01d4c87f6830ec304d42777991d533154` in their source. Only API declarations
are represented by repository-authored Go bindings; no SDK implementation,
Microsoft codec binary or Store package is redistributed. WIC frame-chain and
orientation contracts are linked in the Windows evidence below. Existing
`golang.org/x/sys v0.48.0` and shipped module notices are unchanged.
Fixtures are repository-authored MIT patterns generated with the already installed
FFmpeg 6.1.1/x265 3.5 toolchain; no encoder binary is bundled or invoked at runtime.
Exact commands, profiles, expectations and hashes live in `internal/heic/testdata/`.
The additional ICC profiles are generated with installed LittleCMS 2.14 only
during fixture authoring, with upstream API/license references in that README;
LittleCMS is neither a runtime dependency nor shipped. Distro-specific review
of the two grid advisories is recorded in
`.scratch/os-heic/evidence/linux-provider-security.md`; the lead independently
confirmed Ubuntu Noble's published Not affected classification. This is not a
blanket provider security qualification or a claim of verified backport patches.

### Delegation ledger

| Ticket | Agents used | Lead review | Full suite |
| --- | --- | --- | --- |
| 01 | existing guide agent, 1 initial spawn | reviewed, focused race passed | no |
| 02 | existing native agent, 1 initial spawn | fixes applied inline | no |
| 03 | lead | fixes applied, focused race passed | no |
| 04 | reused native agent | reviewed, native/race passed | no |
| 05 | reused guide agent | candidate reviewed, native gate open | no |
| 06 | reused native agent | production blocker confirmed | no |
| 07 | reused guide agent | reviewed, native/race passed | no |
| 08 | reused native and guide agents | editor correction and package review complete | no |

Two agents maximum remained active concurrently. Follow-on tickets were concrete
bounded work with separate ownership, existing acceptance commands and fixed
interfaces; the user's explicit ticket-delegation request permits their wider
file scope. No review or post-review fix was delegated. No commits or pushes.
