# Explorer regular-release setup

Status: complete (2026-09-10). Route: Deep (runtime download, native worker lifecycle and release
integration). Lead owns design, implementation, review, fixes and the final gate.

## Accepted scope

Ronin authorized straightforward first-run model/runtime setup, a regular
release, a GitHub Discussions link for voluntary feedback, and an accurate
privacy-policy update. No analytics, feedback collection, identifiers, or image
uploads. Assess the remaining Linux/Windows support work before claiming support.
Prepare concrete release artifacts and notes through the existing workflow;
do not commit. Preserve unrelated `:memory:.ses` files.

The accepted test boundaries remain the production viewer and similarity
engine/provider interfaces. Exercise download outcomes through these boundaries
with controlled HTTP fixtures, followed by actual installed-model execution.
Existing model/runtime digests and offline processing remain required.

## Tasks and evidence

1. **Reconnaissance (lead + one read-only scout).** Locate first-run setup,
   privacy/link/release surfaces and platform blockers. Oracle: file:line evidence
   checked with targeted reads; primary vendor sources for current external facts.
2. **Download/setup (lead).** Specify user-triggered setup, progress, cancellation,
   retry, verified installation and subsequent offline use before TDD slices.
   Pin exact files and acceptance commands after reconnaissance.
3. **Discussions/privacy/release docs (lead).** Add the existing project's
   Discussions URL; explain download providers and ordinary request metadata,
   without inventing collection by PicFetch. Keep en/de UI/manual parity.
4. **Qualification and handoff (lead).** Focused tests, actual model setup/run,
   canonical `make verify`, normal build and relevant packaging. Report platform
   support only to the extent observed. Record unresolved platform prerequisites
   in `todos.md` with release preparation status.

Graph: platform reconnaissance and lead setup/privacy reconnaissance run in
parallel; findings feed implementation and the shared final gate.

## Delegation gate and budget

One read-only scout answers a bounded cross-platform code question. G1: short
prompt with explicit files; G2: file:line findings verified by shell; G3: zero
writes; G4/G5: independent platform/CI breadth while lead owns setup and privacy.
S/W: semantic lifecycle lookup, not a scriptable transform or specified code.
Budget: one scout, two lead review rounds per slice, one final full verification
after focused checks. No delegated review or implementation.

## Concrete setup contracts

- `similarity.Client.CheckAssets(ctx)` verifies the selected local asset set;
  `InstallAssets(ctx, progress)` downloads only pinned public assets and returns
  the installed directory. A per-client HTTP transport supports controlled
  network-boundary tests. Downloaded bytes have size bounds and SHA-256 checks;
  archive extraction admits only named runtime/license files. Incomplete files
  never become loadable assets. Cancellation closes requests and removes staging.
- Default installed assets live in a per-user cache directory. Explicit developer
  asset overrides and existing verified local trial assets remain usable.
- Production first use shows Trane sorting photos, explains local grouping and
  the one-time download, and offers a user-triggered download or continuation.
  No download occurs on startup, cancellation or merely reading the explanation.
  Successful continuation persists the introductory acknowledgment. Retry and
  cancel remain available; source replacement/shutdown invalidate setup work.
- Setup workers join the existing Explorer waitgroup/UI queue and own a request
  lifecycle. User-visible strings are localized; the first-use surface includes
  plain Discussions and privacy-policy links, without attached app data.
- Test through existing engine/evaluator and production viewer seams: rejected
  HTTP/checksum responses leave no activated model; cancellation terminates an
  in-flight transfer; actual pinned local fixtures demonstrate complete install,
  verified reuse without requests, and offline analysis. Root UI cases establish
  first-use display, no download before choice, cancellation and retry.
- Verification: focused `go test ./scripts/explorereval -run TestAssetInstall`,
  root `TestVisualSimilarityExplorer` setup subtests, `make explorer-ui-test`,
  `make explorer-test`, `make verify`, build and applicable packaging checks.

## Artwork amendment

Ronin requested a new raster illustration of Trane sorting pictures into stacks
on the first-use page. Built-in image generation uses only existing public Trane
art as identity references. Retain the final selected asset in the repository,
inspect transparency and native layout, and record the prompt/tool provenance.

## Platform reconnaissance

The scout confirmed Mac-only admission, launcher/OS denial, pollable stdin and
runtime manifest. Model/grouping/cache code is shared. Existing bindings contain
Unix and Windows loaders, but native behavior on Linux/Windows is unverified.
The release setup qualifies the Apple Silicon Mac path. Linux/Windows native
support remains explicit follow-up work in `todos.md`; the shared Windows
no-cgo build regression was reproduced and fixed.

## Implementation evidence

- First-use page: RED started analysis before acceptance; GREEN pauses, includes
  Trane in the displayed tree, and persists Continue. Download/retry/cancel:
  RED missing Download; GREEN no HTTP before choice, failure remains retryable,
  cancellation ends the tracked request and removes staging. Source replacement:
  RED left setup visible; GREEN actual image drop invalidates pending delivery.
- Real installed-model UI: first-use Continue completes production offline
  analysis; reopening does not repeat the introduction. The full Explorer UI
  suite passes, including the native worker cases.
- Actual 413,387,010-byte installation: RED rejected the runtime tarball because
  vendor member names have a `./` prefix; exact matching now removes that prefix.
  GREEN `make explorer-install-test` (30.591 s): pinned public files verify,
  runtime notices are present, repeat installation performs no HTTP, and the
  installed runtime analyzes a synthetic image under OS network denial.
- Ronin reported a clipped first-use page in a 520 x 360 window. RED reproduced
  that size; GREEN grows setup to 720 x 660 and retains action buttons in a
  520 x 400 window. Explanatory content scrolls when space is reduced.
- Ronin found the inset dialog's dark frame distracting. RED caught its title
  inset at normal/larger sizes; GREEN uses a modal page that fills the canvas
  through resize, keeping the reading area scrollable and actions fixed.
  The initial canonical race run was stopped because this later change
  superseded its compiled UI. A fresh final `make verify` covers the full page.
- Final recovery review: RED a previously prepared session repeated worker
  errors after its model was removed; GREEN failure invalidates asset readiness,
  and the real-worker retry returns to Download. The full verification is rerun
  after this final code change.
- Ronin requested actual transparent artwork after seeing the white fallback.
  Ronin explicitly authorized local background removal. The final 1448 x 1086
  PNG has verified zero/partial/full alpha, preserves all photo stacks/prints,
  and was inspected on dark backgrounds and both setup-page captures.
  The original opaque PNG failed the corner-alpha check; the final asset passes.
- Final `make verify`: PASS, exit 0, including all Linux race partitions.
  Artifacts: `.scratch/race-runs/20260910T190212Z-EDTNgK`. The final bitmap
  edit was separately checked with the setup-window regression (PASS, 1.053 s),
  visual captures, and native build/packaging. No application code changed after
  the gate began.
- Release preparation is complete; no version bump, tag or release publication
  was performed. Ronin committed the setup implementation as `c258386`; the
  transparent artwork and closeout documents remain reviewable changes.
