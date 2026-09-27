# Help privacy policy

Route: Standard. Add Help -> Privacy policy as an offline Markdown window,
using the existing Help singleton and command-admission behavior.

## Decisions and scope

- Embed the canonical root `PRIVACY.md` in `main.go` and pass it through
  `ui.Run` to `Help.SetPrivacyPolicy`, following the notice-document wiring.
- Reuse the existing translated `Privacy policy` label; retain the policy's
  published English text, as release notes retain their published language.
- Place the action after Release Notes. Repeated activation raises its window;
  Escape closes it. Fresh actions respect Help admission and shutdown.
- Policy wording, Explorer's existing browser link, dependencies and packaging
  behavior are outside this change. The bundled policy follows the installed
  build and does not automatically fetch later policy revisions.
- Standard scope is retained despite the startup wiring and documentation
  increasing the file count: there is no new subsystem or platform behavior.

## Acceptance and tasks

1. Help offers the action and renders the full policy with Markdown headings,
   word wrapping and vertical scrolling. Existing Help menu/visible-window
   boundaries are reused for testing; routine verification is authorized by
   the requested feature and the repository working agreement.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/help`.
2. Singleton, Escape/reopen, refusal and shutdown behavior match existing Help
   windows; the document contains no runtime image loads or unsupported arrows.
   Verify: the same Help package tests and
   `go test -tags no_emoji,nodynamic ./internal/ui -run TestMenuCallbacksRecheckAdmission`.
3. Startup supplies the exact canonical policy without a duplicated Markdown
   source. Existing locale and manual checks remain green.
   Verify: `go test -tags no_emoji,nodynamic .` and `make verify`.

Implementation owner: T0, including tests, review and fixes. Files: main startup,
UI startup, Help implementation/tests, manuals, architecture, command inventory,
Qodana exact test exclusion, and todos. No top-level root UI tests are added.
Task order: menu regression -> policy window/startup -> documentation -> gates.
Budget: one read-only scout, one lead review, one final full suite.

The scout traces startup call sites and menu-index dependencies while the lead
works in Help. G1: bounded prompt; G2: repository search verifies reported paths;
G3: read-only; G4: breadth outside the lead's Help context; G5: not yet inspected
by the lead at dispatch. No implementation or review is delegated.

## Evidence

Implementation verified against base revision `c194a53`, with the feature
source hashes below. Ronin accepted the feature on 2026-09-27 and authorized
its branch, commit, push, PR and GitHub Codex review loop.

- Red: `TestHelpMenu` failed with `expected 7 help items, got 6` before
  implementation. It passed after adding the action.
- Negative verification: temporarily embedding README instead of PRIVACY made
  `TestEmbeddedPrivacyPolicyMatchesShippedDocument` fail; temporarily rendering
  an empty policy made `TestPrivacyPolicyMenuShowsOfflineDocument` fail on the
  missing visible content. Both deliberate faults were restored.
- `go test -tags no_emoji,nodynamic . ./internal/ui/help` passed, including
  locale parity, both manuals, the canonical policy, and existing Help windows.
- Focused root UI menu-refusal and startup-failure regressions passed.
- After restoring the negative probes, `go test -race -tags no_emoji,nodynamic
  . ./internal/ui/help` passed (root 1.731s, Help 41.739s). The focused root UI
  race run also passed (7.561s), covering `TestMenuCallbacksRecheckAdmission`,
  `TestRun_RejectsUnavailableFavoriteStorageBeforeBuildingViewer`,
  `TestBuildMainMenu_Structure`,
  `TestCompareMenuState_DisablesOrdinaryCommandsAndRestoresThemOnExit`, and
  `TestCommandAdmissionModalOwnership`.
- `make verify-build` passed: formatting, TUF root, generated assets, notice
  checks, exact Qodana test exclusions, `go vet`, and `go build`.
- `make verify` stopped at its platform guard: Docker reports
  `linux/aarch64`, whereas the complete suite requires native Linux/amd64.
  The local full suite could not run; no isolation guard was waived. The
  native Linux CI results below subsequently verified the source revision.
- GoLand `get_file_problems(errorsOnly=false)` inspected all eight changed Go
  files listed below, using the active Project Default profile, with no timeout.
  Seven files had no findings. `main.go:125` had one weak duplicate-code warning
  against `.scratch/ma-028/macos-2026-09-27/overlay/main.go:120`, the retained
  native-qualification source copy. An unignored repository search confirmed
  the duplicate. This is an analysis-scope artifact, not duplicated shipped
  implementation; no production suppression or refactor was added. Main and
  privacy rendering were re-inspected after restoring the negative probes.
- These IDE inspections are the documented local fallback. IDE-local Qodana
  was not run through the available connector. CI Qodana SARIF and CodeQL
  evidence was obtained in the PR round below. Native desktop visual
  qualification is not claimed.

Analyzed source SHA-256 values:

```text
5800ba792d908e5ea59b40d78246fa96c787327f9656d9e4bbd0fe55631d37db  main.go
3bb208d842dc558246ce74de2601db5a66eaf7a38eb35d3930d4f8d34d6162b9  main_test.go
12125a20b527816005bce08643c90522f518aeaa66031a77dd48903dbf58c2e5  internal/ui/run.go
4ed162565674c0e0b29a3461701fb33f6bed00a2b596ca64d75704be33715068  internal/ui/preferences_wiring_test.go
1e30cc019f0acccb6363d2136aca92373a62d288f4dcbbc501f2bd227ea21e34  internal/ui/help/help.go
da128b4a688dd6ba9af4f0f3dffbb773043af15d93101a570631d6a220f8b98b  internal/ui/help/manual_test.go
f88024484be04eec64408695e3309fe21bcc53aacf0e60974aac0ff4cf369608  internal/ui/help/privacy.go
9091ecccaa5aa595a33b93425cbc783d1bcc1cbbd9a5b0c86e8e63661b411d65  internal/ui/help/privacy_test.go
```

Cost ledger: one scout budgeted/used, one lead review, one full-suite attempt
(blocked before execution). Implementation and fixes stayed with the lead.
No dependencies changed. The initial implementation handoff was uncommitted.

## GitHub review loop

Branch: `feature/help-privacy-policy`.
PR: [67](https://github.com/frathe/picfetch/pull/67).
Source commit: `bf6152403a2c1ba373cac884239d0b199f88975c`
(`Add offline privacy policy to Help menu`).
The review workflow uses focused local tests; the complete race suite runs in CI.

One additional read-only scout maps CI artifact names and completion evidence
while the lead publishes the branch. The task is bounded, independently
verifiable against workflow files, touches no files, and needs only CI context.
All review assessments and any fixes remain with the lead.

### Source review round

- Codex's [review summary](https://github.com/frathe/picfetch/pull/67#issuecomment-5859440241)
  marks code review complete at 20:15:35 UTC and security review complete at
  20:17:02 UTC on 2026-09-27, both naming `bf61524`. No findings or inline
  review threads were posted; the connector's clean-review thumbs-up was
  recorded at 20:17:05 UTC. No fixes or finding dispositions were needed.
- All 17 PR checks passed. [CI run 36347216466](https://github.com/frathe/picfetch/actions/runs/36347216466)
  passed validation, all four native Linux/amd64 race partitions, Linux native
  guards, Windows tests, and macOS arm64/amd64 native guards.
- [CodeQL run 36347216473](https://github.com/frathe/picfetch/actions/runs/36347216473)
  passed Actions and Go analysis. Analyses `1847999868` and `1848011333` report
  zero results and no error on PR merge revision
  `6367a2febd6cc95a8045e423c80f9f2ae61afd7e`, which GitHub binds to source head
  `bf6152403a2c1ba373cac884239d0b199f88975c`. The PR-ref open-alert query was empty.
- [Qodana run 36347216459](https://github.com/frathe/picfetch/actions/runs/36347216459)
  passed. Artifact `10940828112` (`qodana-report`) contains
  `/end/qodana.sarif.json` with source revision `bf6152403a2c1ba373cac884239d0b199f88975c`,
  profile `starter`, zero post-suppression results, `executionSuccessful: true`,
  exit code 0 and no execution/configuration notifications. This was a
  differential PR scan; the pre-suppression CSV totals were not used.
- FOSSA dependency-quality, license-compliance and security-analysis checks
  passed. Local source hashes and GoLand evidence above remain unchanged.

This evidence-only closure commit removes the verification TODO. It still needs
fresh Codex code/security reviews and all remote gates on its own head. Their
terminal results will be recorded in the PR conversation, avoiding another
documentation commit that would restart the checks. The PR remains open for
Ronin; this workflow does not authorize merging or releasing.
