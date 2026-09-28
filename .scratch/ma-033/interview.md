# MA-033: launch side-effect policy interview

Status: resolved; all ten decisions accepted and shared understanding confirmed.
Date: 2026-09-28
Source: `/grill-with-docs ma-033`
Baseline: `5070d6d`; working tree clean before this interview record.

## Session scope

Sharpen the proposal in `needs_refactoring.md:570` through grilling and
domain-modeling. This is a design interview, not implementation authorization.
All three rounds are accepted. The tracked design is `docs/launch-policy.md`;
`docs/adr/0007-captured-launch-policy.md` records the ownership choice and
`CONTEXT.md` defines the domain-specific isolated-trial term.

Process: Standard design discovery. Lead owns decisions and documentation.
One read-only scout traces the cross-package launch/update paths; no delegated
design, review or edits. The search established locations but could not settle
the behavior matrix without tracing callers. No implementation plan yet.
The same scout supplied two bounded follow-ups on evidence ownership/launcher
expectations and existing verification coverage. Codebase-design informed the
lead's interface choices. Final assessment and document review stayed with the lead.

## Established facts

- Pre-app predecessor cleanup combines the distribution build constant with
  raw trial flags (`main.go:115`). Identity validation follows that guard and
  precedes `app.NewWithID` (`main.go:158`).
- `Options.ApplicationID` rejects simultaneous trial modes, derives separate
  application identities, and verifies OS network denial for Explorer trials
  only (`internal/launch/launch.go:66`). Location Map trial isolation does not
  imply that network-denial requirement.
- Root update admission reads `storeManaged` and live Explorer/Location Map
  trial objects (`internal/ui/autoupdate.go:28`, `:64`, `:103`). Startup backup
  cleanup and shutdown apply repeat related checks (`internal/ui/run.go:97`,
  `:233`).
- The updater methods have no distribution/trial admission of their own
  (`internal/ui/autoupdate/updater.go:413`, `:426`, `:605`, `:632`). The viewer
  currently supplies that protection; this alone does not establish a bypass
  in an existing production route.
- Viewer construction supplies the ordinary staging path (`internal/ui/build.go:101`)
  before trial options replace it (`internal/ui/launchoptions.go:112`). The
  updater constructor only stores its configuration; it does not read stages
  or start workers (`internal/ui/autoupdate/updater.go:102`). Do not describe
  this temporary configuration as proven normal-install disk access.
- Manual checks bypass the automatic preference and daily due gate. Turning
  automatic checks off cancels the request but retains a completed stage for
  shutdown apply (`internal/ui/autoupdate.go:24`, `:97`). Launch permissions
  and the user's automatic-check preference are separate concerns.
- Trial options redirect Favorites, presets, general analysis and updater
  storage (`internal/ui/launchoptions.go:102`). Fyne preferences/session storage
  is separately selected by the application identity.
- Evidence-directory newness is checked after Fyne app creation: Explorer at
  the beginning of `Run`, Location Map after viewer construction
  (`internal/ui/run.go:47`, `:61`, `internal/ui/locationtrial.go:23`). This is a
  startup-order observation, not evidence of ordinary-storage corruption.

## Round 1: accepted decisions

User response: "go with defaults". Q1-Q3 are settled as recommended below.
That response accepted the first round only; later rounds are recorded below.

### Q1: scope of launch policy

Should MA-033 cover GitHub self-update effects and existing launch storage
isolation, or become a general authority over networking and filesystem effects?

Accepted: the bounded scope. Include predecessor cleanup, startup backup
sweep/update records, automatic/manual check and download, staging/apply/relaunch,
application identity and existing trial storage routing. Map tiles, model-asset
setup, source-file editing and OS network enforcement retain their own contracts.
Preserve Explorer's existing verified-offline launch prerequisite.

### Q2: enforcement boundary

Is consuming the captured policy in root/UI guards sufficient, or must the
updater also refuse prohibited effects when called directly?

Accepted: both root composition and the updater consume the same captured
decision. Root retains presentation and pre-app cleanup; updater admission rejects
prohibited work before verifier preparation, stage access, network or apply.
Do not repeat trial-name predicates or introduce a global permission service.

### Q3: invalid-launch failure boundary

Must an invalid or already-used trial evidence directory be rejected before
Fyne preferences/session access and feature construction, or may the present
late rejection remain?

Accepted: reject before Fyne storage access and feature construction.
Necessary validation/probing/reservation belongs to explicit launch preparation;
the captured policy value itself performs no effects. Validation or isolation-setup
failure aborts the launch, never falls back to ordinary storage or permissions.
Preserve successful-launch IDs/layouts, normal automatic/manual update semantics,
Store restrictions and the trials' different network requirements. Reservation,
error cleanup and recorder startup are resolved in Q6 and Q8 below.

## Round 2: accepted decisions

User response: "go with defaults". Q4-Q7 are settled as recommended below.
That response accepted the second round only; the final round is recorded below.

### Q4: missing policy and lifetime

If a caller forgets to supply the captured policy, should that mean ordinary
portable behavior or refuse covered effects?

Accepted: explicit successful construction is required; missing/invalid
policy refuses covered effects and application composition returns an error.
An explicit construction from zero launch options remains a valid ordinary
launch. Capture read-only decisions once, with no setters or fallback to live
feature state. Closing a trial view cannot change the process's permissions.
Tests construct the policy through the same interface as production.

### Q5: distribution and trial combinations

Are ordinary portable, Store, Explorer trial and Location Map trial four
mutually exclusive modes, or do distribution and trial purpose remain separate?

Accepted: separate distribution from trial purpose. Store plus a trial
combines both restrictions and uses trial storage, subject to existing platform
and offline-verification prerequisites. Both trial flags together remain an
error. Do not introduce a new ban on trials merely because a build is Store
managed, or claim that Explorer trials are newly supported on Windows.

Evidence: launch validation currently does not reject Store/trial combinations;
Explorer's `VerifyOffline` still refuses Windows. Distribution is captured from
the build, not inferred from executable paths or trial feature objects.

### Q6: failed preparation and evidence ownership

If a fresh evidence directory has been reserved but later startup fails, should
the app remove it for retry or retain the failed attempt? Who closes any prepared
recorder before the UI lifecycle has taken ownership?

Accepted: retain the directory and any evidence, report the failure, and
require a fresh directory for the next attempt. Preparation owns closing files
and stopping/joining any started recorder on failure. Normal termination retains
the existing order: stop/join producers before finalizing their evidence. Keep
resource cleanup explicit and idempotent, outside the passive policy value;
never recursively delete or silently reuse an evidence directory.

Evidence: runners already keep failures and require absent app-owned child
directories (`scripts/explorereval/native_darwin.go:46`, `:96`, `:184`;
`scripts/locationmapqualify/runner.go:190`, `:232`; `manual.go:87`, `:119`).
Location Map treats a missing state file as pending (`runner.go:71`). Explorer
`Close` finalizes incomplete evidence even before any analysis; Location Map
requires `Stop` then `Wait` to join its recorder worker
(`internal/explorertrial/session.go:221`, `internal/locationtrial/recorder.go:71`).
Directory existence alone is not evidence of a completed trial.

### Q7: unavailable-update presentation

Should trial Settings continue displaying update controls whose actions are
refused, or render the captured restriction before the user tries them?

Accepted: keep the Updates tab and installed version, replacing unavailable
check/apply controls with an accurate explanation. Preserve existing Store copy;
trial sessions explain that updates are unavailable for the session. Root and
updater refusal remain authoritative for stale/direct calls. Presentation reads a
small supplied decision/reason, not trial objects or independently derived rules.
Combined-reason presentation is resolved in Q10 below.

Evidence: `settingswin.Show` currently receives only the Store bool
(`internal/ui/commandadmission.go:155`); its non-Store branch displays both update
controls even in trial launches (`internal/ui/settingswin/settingswin.go:433`).

## Round 3: accepted final decisions

User response: "go with defaults". Q8-Q10 are settled as recommended below.
The final round explicitly stated that accepting these defaults confirms the
consolidated design and completes the interview. No design questions remain open.

### Q8: ownership across application construction

Should preparation transfer trial resource ownership into the viewer, or retain
one owner across construction, the UI run and every ordinary error return?

Accepted: retain one preparation owner around the UI run. It provides the
immutable policy separately from the borrowed trial recorders. The UI stops and
joins trial-evidence producers before returning; the preparation owner then
closes evidence and stops/joins any recorder, including on startup failure. Cleanup must complete
before process exit, rather than relying on defers bypassed by `os.Exit`.
The policy carries no live features, callbacks or resource handles.
This concerns trial-evidence producers; existing unrelated feature shutdown
policies, including the documented uninterruptible-read exception, stay local.

Reuse existing identity validation and trial recorders; keep the entry point
thin. Supply selected storage configuration before feature construction. Ordinary
app-cache paths can resolve from the already correctly identified Fyne app;
this does not authorize trial fallback to normal paths. Retain private-worker
dispatch before desktop preparation and the macOS Open With installation before
Fyne/GLFW initialization. Covered cleanup follows validated policy; the ordinary
predecessor wait remains before Fyne preferences, and backup sweep remains before
failure-record consumption.

### Q9: meaning of storage isolation

Does this change guarantee correct startup selection of isolated storage, or
also prevent another process from redirecting directories during a run?

Accepted: preserve a routing guarantee: exclusive reservation of a fresh
trial directory, existing identity/path layout, no runtime fallback to ordinary
storage and no permissions derived from live feature state. It does not add a
filesystem sandbox or ownership handles for the complete trial directory tree,
nor guarantee protection from external replacement of its paths. Existing
Favorite ownership contracts still apply to their own operations. Source edits
and shared model/runtime assets remain outside this launch-policy scope.

Application identity and trial root derive from the same captured launch input.
Existing identity formats remain unchanged. Normal app-cache path resolution
stays with Fyne, and model assets do not become trial-owned merely because the
trial uses them. New adversarial filesystem guarantees require separate scope.

### Q10: combined refusal reasons

When both Store distribution and trial purpose prohibit self-update, should
Settings display only one reason or both?

Accepted: show both existing explanations with the installed version,
and no update action controls. Policy retains both facts so presentation does
not impose an arbitrary precedence or reconstruct launch mode. Ordinary Store
and portable-trial sessions each show their single applicable explanation.
All displayed strings retain the project's localization rules.

## Completion of the interview

All ten decisions are accepted. Shared understanding is confirmed and the
interview is complete. This authorizes recording the design, not implementation,
commits, pushes, PR changes or release actions. Exact code identifiers and
acceptance commands belong to the subsequent specification/implementation plan.

The verification obligations below follow from the accepted scope and the
repository's required gates; they are not optional questions.

## Required implementation evidence

- Exercise the production startup orchestration through injected per-call
  operations: help/bad flags and private worker dispatch must not enter desktop
  preparation; rejected trial preparation must precede Fyne storage/features.
  Preserve macOS bridge placement and predecessor wait before app preferences.
- Test the distribution-by-trial matrix, invalid/missing policy, explicit
  ordinary construction and dual-trial rejection. Real offline verification
  remains separate from a simulated policy case.
- Exercise automatic/manual checks, staging, verifier construction, apply and
  relaunch through real root and direct updater entry points. Denied paths
  produce no covered I/O. Use operation counters to observe reads and temporary
  sentinel artifacts to observe mutation; unchanged file contents alone cannot
  prove that no read occurred.
- Test both startup cleanup paths and the order protecting a failed-restore
  backup. Cover automatic preference changes while retaining a completed stage,
  stale/direct requests and trial feature closure without permission changes.
- Test selected identity/paths before construction, prepared-resource failures,
  retained evidence, fresh-directory reuse rejection and all worker completion
  observations. A policy-unit matrix alone does not prove composition.
- Check Settings' actually mounted content for each applicable refusal reason;
  keep translations, root UI shard assignments and exact Qodana exclusions current.
- Run focused TDD, changed-file GoLand inspections including weak warnings,
  `make verify` using native Linux/amd64 Docker, and relevant native Windows,
  Store and macOS qualification. Extend native guard selection to include the
  new startup guards where current selections omit them. Use existing fixtures
  and focused native startup/evidence checks; this does not reopen Location Map
  performance or model-quality qualification. Report unsupported or unrun
  combinations accurately.

Inventory finding: `main_test.go` tests helpers; worker test binaries use their
own `TestMain` dispatch. Neither proves production entry-point order. Store and
Windows native suites currently omit root main, and Store omits root UI as well
(`scripts/nativeguards/main.go:73`). Existing Cocoa graft and trial-runner tests
likewise do not prove the proposed earlier preparation order. New composition
coverage and native selection updates are required, not inferred passes.

## Verification

Code/document inspection and documentation checks only. No production code was
changed, and no runtime tests or native qualification have been performed for
this interview. Documentation verification covers `git diff --check`, local links
and the MA-033 anchor, unique Q1-Q10 headings, three accepted rounds, and
whitespace/final-newline hygiene, including the ignored interview record.

## Comments

2026-09-28: `/to-spec ma-033` published [the specification](spec.md) with
`ready-for-agent` status, 60 user stories and 22 acceptance criteria. The agreed
testing interfaces were carried forward without another interview. Read-only
reinspection confirmed the current `a5caf73` tree matches the interview baseline;
the updater persistence/record inventory sharpens coverage within Q1-Q2 without
reopening the accepted scope. No implementation or runtime verification occurred.
