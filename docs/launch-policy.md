# MA-033: launch side-effect policy

Status: accepted design; all ten decisions implemented and qualified.
Date: 2026-09-28
Source: `/grill-with-docs ma-033`
Code inspected at: `5070d6d`; rechecked at `a5caf73` with an identical tracked tree.

This records the accepted decisions for
[MA-033](../needs_refactoring.md#ma-033). The
[local interview](../.scratch/ma-033/interview.md) retains questions and code
evidence. The user accepted all three rounds with "go with defaults", including
the final round's shared-understanding confirmation. The design interview is
complete. The [resolved specification](../.scratch/ma-033/spec.md) has
60 user stories and 22 qualified acceptance criteria covering the agreed
interfaces and verification. The subsequent `/implement MA-033` request authorized
implementation and the GitHub review loop. The
[archived record](../finished_refactorings/2026-09-28-ma-033-launch-policy.md)
contains code/native evidence; [PR 72](https://github.com/frathe/picfetch/pull/72)
tracks latest-head reviews and checks.
The [nine approved tickets](../.scratch/ma-033/issues/README.md) provide the
implementation sequence, dependency graph and acceptance-criterion ownership.

## Accepted scope

Capture launch side-effect policy before covered startup effects. The policy
governs GitHub self-update admission and existing ordinary/trial storage
selection: application identity, Favorites, presets, general analysis storage
and updater storage. Application identity selects Fyne preferences/session data.

Covered update effects include predecessor cleanup, startup backup cleanup and
update records, automatic/manual checks and downloads, staging, apply and
relaunch. Root composition and updater admission consume the same captured
decision; they do not independently reconstruct it from trial feature objects.
The updater refuses prohibited effects before verifier preparation, stage access,
network activity or installation. Root retains presentation and pre-app cleanup.

Map tiles, model-asset setup, source-file changes and OS network enforcement
retain their existing contracts. Explorer's verified-offline launch prerequisite
remains; Location Map isolation does not acquire that prerequisite.

## Invalid launch preparation

Invalid or already-used trial evidence directories must be rejected before Fyne
preferences/session access and feature construction. Necessary validation,
network-denial probing and evidence reservation belong to explicit launch
preparation. The captured policy value performs no effects itself.

A validation or isolation-setup failure aborts the launch without falling back
to ordinary storage or permissions. Preserve successful-launch identities,
storage layouts, Store restrictions and existing automatic/manual update
semantics. In particular, automatic opt-in remains separate from fixed launch
permissions: disabling automatic checks still leaves a completed ordinary
portable stage eligible for shutdown apply, and manual checks still bypass
the automatic preference/daily due gate.

## Validity and lifetime

Successful explicit construction is required. Missing/invalid policy refuses
covered effects and application composition returns an error. This differs from
valid construction using zero launch options, which requests an ordinary launch.
Tests construct policy through the same interface as production.

The captured decisions have no setters and remain fixed for the process.
Closing or replacing a trial feature cannot authorize updates or switch to
ordinary storage. Mutable automatic-check preferences remain separate.

## Distribution and trial purpose

Distribution and trial purpose are separate inputs. The following matrix
describes policy after existing launch prerequisites have succeeded; it does
not establish platform support for any trial.

| Distribution | Trial purpose | Storage selection | GitHub self-update effects |
| --- | --- | --- | --- |
| Portable | None | Existing ordinary storage | Permitted, subject to the existing operation's conditions |
| Store | None | Existing ordinary storage | Refused |
| Portable | Explorer | Existing isolated Explorer identity and trial paths | Refused |
| Store | Explorer | Existing isolated Explorer identity and trial paths | Refused |
| Portable | Location Map | Existing isolated Location Map identity and trial paths | Refused |
| Store | Location Map | Existing isolated Location Map identity and trial paths | Refused |

Both trial flags together remain invalid. Store distribution adds no new blanket
ban on trial launches. Explorer's verified-offline prerequisite still excludes
Windows; a policy matrix case is not native qualification of that combination.
Distribution comes from the build, not executable-path heuristics or live UI.

## Failed preparation and evidence

Retain a reserved trial directory and any partial evidence if subsequent startup
fails. Report the failure and require a new directory on retry. Do not recursively
delete or silently reuse an evidence directory. Directory existence does not
establish that evidence collection completed.

Preparation owns closing files and stopping/joining any recorder it started on
failure. Resource cleanup is explicit and idempotent, outside the passive policy
value. One preparation owner retains trial resource ownership across application
construction, the UI run and ordinary error returns. The UI borrows the recorders
and stops/joins its trial-evidence producers before returning; preparation then
finalizes evidence and stops/joins any recorder. Startup failures use the same cleanup
owner. Complete cleanup before process exit rather than relying on defers that
`os.Exit` bypasses. The policy contains no feature objects, callbacks or handles.

Reuse the existing identity validation and trial recorders. Resource ownership
stays separate from the read-only decisions consumed by root and updater.

## Startup composition

Preserve the existing native ordering while moving validation and selected
storage configuration ahead of their consumers:

1. Parse flags; help and malformed input stop before desktop effects. Dispatch
   private HEIC/similarity worker modes before desktop preparation.
2. Keep native Open With installation in its existing early desktop position,
   before Fyne/GLFW initialization.
3. Prepare and validate the launch, capture the immutable policy, and reserve
   fresh trial evidence before opening Fyne preferences/session storage.
4. Use the validated decision for ordinary predecessor cleanup/wait, still
   before Fyne application creation and preferences access.
5. Construct the app with the selected identity. Supply the selected storage
   configuration before feature construction; ordinary app-cache paths may
   resolve from that correctly identified Fyne app. Trial consumers never
   temporarily rely on ordinary storage and later retarget it.
6. Preserve explicit feature construction and runtime startup. For admitted
   ordinary update startup, sweep the backup before consuming the failure
   record, so a failed-restore backup remains protected.
7. After the UI stops and joins its trial-evidence producers, finalize prepared
   trial resources through their single owner. Return startup/run/finalization
   errors honestly.

Keep `main` thin. The implementation chooses concrete identifiers while retaining
this production sequence as an observable test surface. An ordinary options value
does not bypass explicit policy construction, and a separate test-only startup
sequence cannot establish that the production ordering is correct.

## Storage guarantee and limits

Isolation means correct startup selection and exclusive reservation of a fresh
trial directory, preservation of existing identity/path layouts, no runtime
fallback to ordinary storage, and permissions independent of feature lifetime.
Application identity and trial root derive from the same captured launch input.
Keep ordinary Fyne cache resolution with Fyne.

This does not introduce a filesystem sandbox or ownership handles for the whole
trial directory tree, and does not guarantee protection from another process
replacing trial paths during execution. Existing Favorite ownership protections
continue to govern their own operations. Source-image edits and shared model/
runtime assets retain their existing contracts; they do not become trial-owned.

## Update presentation

Keep the Updates tab and installed version. When updates are prohibited, replace
check/apply controls with an accurate explanation derived from the supplied policy
decision/reason. Preserve the Store explanation and provide the trial-session
explanation. A Store trial shows both applicable explanations, with the installed
version and no update action controls. Ordinary Store and portable-trial sessions
show their single applicable explanation. Retain both facts in the supplied
decision instead of imposing a priority or reconstructing the mode in Settings.
All displayed strings follow the existing localization and catalogue rules.
Root and updater refusal still protect stale/direct calls; presentation does not
derive independent permission rules from trial objects.

The [launch-policy ADR](adr/0007-captured-launch-policy.md) records why fixed
admission and prepared-resource ownership remain separate.

## Scope of the next work

No design decisions remain open. The specification turns these ten decisions
into behavioral contracts and executable acceptance criteria. Implementation
planning will name concrete interfaces and tasks.
There is no new dependency, storage-format migration, general permission service,
automatic feature registry or trial-mode expansion in the accepted scope.
Implementation, commits and publication were not requested by this interview.

## Required implementation verification

The policy matrix must be exercised through production startup composition and
root/updater effect interfaces. Existing helper tests do not prove that rejected
trial preparation precedes Fyne storage access. Use per-call injected operations
and temporary artifacts to observe sequencing, reads and writes; unchanged
sentinel contents alone cannot prove that a prohibited read was avoided.

Cover automatic/manual checks, verifier construction, stage cleanup/access,
apply/relaunch, both startup cleanup paths, and backup-sweep ordering before
failure-record consumption. Preserve automatic preference changes and manual
bypass behavior. Feature close must not change fixed launch permissions.
Test resource cleanup after partial preparation and UI errors, retained failed
evidence, ordinary/trial identity and paths, and Settings' mounted content.

The specification supplies the acceptance-criterion command map for those
behaviors and identifies focused existing regressions. Implementation must add
the required new guards. Required gates remain focused TDD,
changed-code GoLand inspection, `make verify` on native Linux/amd64 Docker and
relevant Windows/Store/macOS native qualification. Native guard selections must
actually include the changed entry-point paths: current Windows/Store suites
omit root main, and Store also omits root UI. Native trial prerequisites remain
real checks; a simulated Store/Explorer matrix case does not establish Windows
Explorer support. Unavailable gates remain unverified.

No Location Map performance or model-quality requalification is implied by this
startup refactoring. The [interview](../.scratch/ma-033/interview.md) records the
existing test inventory's limits and the required additional coverage. No tests
listed as required here have been added or run by this design interview.

## Evidence limits

Code and document inspection only. No production code was changed and no runtime
tests or native qualification have been run for this interview.
