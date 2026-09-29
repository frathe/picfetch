# Native Location Map gesture timing

Deliverable: identify requested pan/zoom transforms from captured pixels before
accepting native latency samples. Route: Deep, because the capture adapter is
macOS-specific. Branch: `feature/location-map-gesture-timing`.

## Scope and decisions

The current helper refuses gestures because body hashes cannot distinguish input
responses from tile delivery. Keep hashes for stable-frame admission and the
existing closed-viewer exit boundary. Use independent image registration for
gestures, without reading application camera state or using worker timestamps.
No application behavior, dependencies, performance promises or existing maintainer
acceptance changes. Ronin subsequently authorized commits for the macOS handoff;
the feature branch will be pushed for that handoff. No merge/release or native
input automation has been requested.

Use the existing frame-observation and Go protocol/evidence seams documented in
the prior PR58 plan. A portable Swift matcher consumes sampled luminance frames;
the macOS adapter alone owns capture/input. Match distributed textured patches
under translation and the keyboard's 2x/0.5x scales. Reject insufficient texture,
stationary content, wrong direction, ambiguous registration and unrelated repaint.
Preserve the first verified frame's WindowServer timestamp and retained PNG pair.
Persist a versioned transform witness; old boolean-only reports cannot qualify.

Honest limit: textureless/ambiguous maps, changed content with no remaining visual
correspondence, clamped zoom and unstable baselines fail closed. Synthetic matcher
tests do not qualify native performance. This Linux host has no macOS SDK or
native capture permissions; any unavailable native gate remains unverified.

## Acceptance and tasks

1. T0: pure matcher and synthetic frame tests, existing capture policy tests,
   Make/CI wiring. Identify both pan directions and zoom directions; reject
   stationary, repainted, ambiguous, wrong-direction and insufficient frames.
   Verify: `make location-map-capture-test` (portable Swift toolchain); native
   adapter typecheck remains in both macOS CI jobs.
2. T0, depends on 1: connect capture to matcher, retain transform witness and
   first qualifying frame time, preserve stable admission/exit and failed samples.
   Update Go protocol and report validation at their existing seams.
   Verify: `make location-map-qualification-test`; synthetic Swift policy tests.
3. T0, depends on 2: review complete diff, update docs/todos and record evidence.
   Verify: changed-file GoLand inspections (including weak warnings),
   `make verify`, plus the Swift command above. Native SDK compilation and live
   screen/input checks must be reported separately if unavailable.

Graph: 1 -> 2 -> 3. Budget: one read-only scout, no delegated implementation or
review; one final complete suite. Lead owns design, tests, changes and review.
Scout G1: bounded CI/native-host reconnaissance; G2: verify returned locators;
G3: no writes; G4/G5: unread build/native-evidence breadth; S/W: relationship search,
not a deterministic edit or delegated implementation. Scout found both macOS
CI architectures, no local Swift and no retained Location Map screen fixtures.

## Checklist and evidence

- [x] Frame/spec/recon/plan and delegation gate.
- [x] Red/green at the established observation and evidence seams.
- [x] Lead review, negative guards, focused verification and inspections.
- [x] Linux final gate and docs/todo handoff; native qualification pending below.

Initial Go protocol suite passes with `/snap/go/current/bin/go test -count=1
./scripts/locationmapqualify`. The `/snap/bin/go` launcher is unavailable inside
the sandbox; the installed toolchain binary works. Docker reports native
`linux/x86_64`. No shipped dependency or license closure changes are planned.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 | Lead locator check | no |
| Implementation | 0 / 0 | 1 + prerequisite repair | no |
| Final gate | 0 / 0 | passed locally; macOS pending | 2 (first exposed missing Git) |

## Implementation and verification

`native/transform.swift` searches translations and 2x/0.5x scales independently
of the requested input. It requires at least eight textured patches, 65% match
coverage, distribution across both axes and separation from competing transforms.
Capture uses averaged 4x4 luminance blocks; stable admission and exit still use
the existing body hash. The first identified captured frame supplies its original
WindowServer timestamp. Timeouts retain the failed observation and PNG pair.
Schema 2 requires `patch-grid-v1` scale/translation/count witnesses. Collection
also checks the witness against the requested key and rejects wrong response kinds.

Red/green evidence:

- Swift's initial empty matcher failed `Requested pan 123 was not identified`.
  Positive pan/zoom fixtures subsequently passed in both directions. Negative
  fixtures cover stationary, wrong direction/scale, unrelated repaint, one
  arriving tile, repeated patterns, insufficient texture and malformed frames;
  partial tile occlusion still accepts coherent pan. Deliberately removing the
  pan direction/zero-motion guard failed `Accepted stationary as pan`; restoring
  it returned both portable suites to green.
- `TestCheckReportRejectsInvalidTransformEvidence` failed all nine invalid-witness
  cases before report validation; all passed afterwards. The focused `^TestCheck`
  suite passed. `TestNativeProtocolRejectsMismatchedTransform` failed all five
  cases before collection validation, then passed.
- `PATH=/snap/go/current/bin:$PATH make location-map-qualification-test`: PASS,
  `scripts/locationmapqualify` 3.262s, `internal/locationtrial` 1.038s, both race.
- Both Swift suites pass with warnings as errors, using the Make target's compile
  commands in an offline container (the official image lacks Make). Toolchain:
  Swift 6.2.4, x86_64 Linux; image `swift:6.2`, digest
  `sha256:9bea530093ffff8cf6c259991715ee843fe4d0f932e612f7f4b79cca6e00db87`.
  Output: `Native Location Map visual transform tests passed` and
  `Native Location Map response-frame policy passed`. The AppKit adapter is
  excluded on Linux and remains unverified against the macOS SDK.
- The actual `make location-map-capture-test` target also passed after mounting
  `/usr/bin/make` read-only at `/usr/local/bin/make` in that same offline Swift
  container, with the repository mounted read-only at `/work`.

No new shipped dependencies or assets. Swift/Git changes are development tooling;
the macOS helper uses the existing system frameworks and repository-owned matcher.

### Full-suite prerequisite repair

The first `make verify` passed format/assets/notices/vet/build and shard inventory,
then found the existing MA-033 provenance tests require Git in the Docker runner.
Evidence: `.scratch/race-runs/20260929T063542Z-MK5T29/console.log` and raw JSON.
`TestLaunchPolicyNativeSuite/evidence_metadata` reproduced in 0.007s with
`exec: "git": executable file not found in $PATH`. Installing Git exposed the
second environment boundary: Git rejected `/work` as differently owned.
Exact-path process-scoped Git configuration fixed that; the same regression
passed under race in 1.086s. No application/provenance test was weakened.

The diagnosis workflow used the explicit missing-executable error and tight
single-test reproduction; speculative hypothesis/instrumentation phases were
unnecessary for that direct environment failure. Existing Docker setup tests
were extended first and failed for missing Git. Race, ordinary test and coverage
containers now install Git and export only `safe.directory=/work` through Git's
process-scoped config. Host/global Git configuration is untouched.
This repair adds `scripts/testshards/docker-race.sh` and its existing test file
to scope. A second full verification is justified by this gate failure; it is
the only planned broad-suite repeat. Sandboxed host testshards tests separately
hit read-only Go cache/VCS access; container verification supplies that evidence.

Final `PATH=/snap/go/current/bin:$PATH make verify`: PASS, exit 0. Fresh artifacts:
`.scratch/race-runs/20260929T064642Z-tWQBoB/`. Both runner/container exit codes
are 0, with zero OOM events. All non-UI packages passed, including nativeguards,
testshards and locationmapqualify; UI shards passed in 637.873s, 428.694s and
473.562s. Formatting, TUF/assets/notices, exact Qodana exclusions, vet, build and
the complete 740-runnable UI shard inventory passed. `bash -n` and
`git diff --check` passed. The separate corrected-container testshards race
package passed in 3.500s. No test exclusion or worker isolation policy changed.

### Inspections

Analyzed working tree based on `1c08654`, carried into the handoff commit without
further Go changes: GoLand `get_file_problems(errorsOnly=false)` on all five
changed Location Map Go files returned no findings. The changed testshards Go
file has four weak duplicate-fragment warnings at lines 90/174 and 225/275,
all in unchanged intentional fixture/assertion sequences covered by its existing
exact `qodana.yaml` exclusion. No duplication refactoring is warranted.
The shell file, Makefile and CI YAML returned no diagnostics. All four Swift files returned no
diagnostics through the same API, but batch lint returned an empty item list;
do not count that as Swift language/SDK inspection evidence. Portable Swift
warnings-as-errors compilation is the verified check; macOS SDK/IDE inspection
is still required. This is the documented IDE fallback, not a Qodana SARIF pass.
No PR was opened; this repository's CI triggers on main pushes and PRs, so the
feature-branch push alone does not provide fresh CI, CodeQL or Qodana evidence.

## macOS handoff

Stay on `feature/location-map-gesture-timing` and read this plan before continuing.
Commit/push authorization persists for the handoff; no merge/release authorization.
The todo remains open until native verification resolves the remaining work.

1. Fetch/switch to the branch and confirm a clean tree. Run
   `make location-map-capture-test SWIFTC='xcrun swiftc'`, then type-check the
   production adapter with
   `xcrun swiftc -parse-as-library -warnings-as-errors -typecheck scripts/locationmapqualify/native/capture.swift scripts/locationmapqualify/native/transform.swift`.
   Run `make location-map-qualification-test` and inspect changed Swift code with
   a suitable native tool. Fix any adapter/compiler issue and repeat its checks.
2. Obtain the explicitly chosen image collection and a fresh evidence directory,
   plus approval for the foreground CGEvent-driven trial. The helper preflights
   existing Screen Recording and Accessibility permissions; do not grant them
   automatically. Run `make location-map-qualify` with those explicit paths.
3. Inspect whole-window before/after PNGs against retained witnesses and the
   recorded direction/latency. Verify 40 pan/zoom samples, closure matching and
   preservation of any failed sample. Use actual admitted image count when
   checking smoke evidence, and demonstrate rejection under 10k/30k counts.
   If registration refuses real imagery, retain the failed evidence and improve
   the matcher against a reproducible fixture; never weaken it to hash changes.
4. A small native smoke validates the helper, not the 10k/30k performance promise.
   Use the documented exact-count protocols if formal qualification is requested.
   Ronin alone supplies the 30k verdict. Keep the maintainer acceptance separate.
5. Record native results, update this plan/todos and commit/push. On Apple Silicon,
   do not run or claim an emulated amd64 full-suite pass; use the recorded native
   Linux verification or native amd64 CI for that gate. Both macOS CI jobs now
   run the portable matcher/policy target and type-check the production helper.
