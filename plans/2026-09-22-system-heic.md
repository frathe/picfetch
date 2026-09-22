# System-provided HEIC implementation

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

Apple adapters use system frameworks only. The Windows experiment uses public
SDK WIC declarations, tied to the upstream source revision in its evidence.
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
