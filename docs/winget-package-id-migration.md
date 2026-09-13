# Deferred WinGet package ID migration

**Status: deferred by Ronin on September 13, 2026.** Retain
`io.github.frathe.picfetch` as the WinGet package ID for now. The possible future
target is `frathe.picfetch`. The process and the effect on existing Windows
installations need evidence before choosing to proceed. Ronin is handling the
earlier rename edits separately.

This document is the requested preparation for later work. All tickets below
are pending; writing the plan does not start a migration or authorize upstream
submissions, pushes, website publication, or releases. Keep this as one document
until the work is resumed. Evidence can then be attached to each ticket or linked
from it, with dates, exact revisions, commands, output, and an owner.

## Scope and known constraints

Only the WinGet catalog identifier and references to that identifier would change.
The application identity stays `io.github.frathe.picfetch` in
[main.go](../main.go), [FyneApp.toml](../FyneApp.toml), and
[Makefile](../Makefile). That identity locates preferences/cache and identifies
the packaged app; changing it is outside this migration. Keep binary contents,
release URLs, checksums, signing, and Microsoft Store identity unchanged.

Three separate results are required:

| Result | What proves it |
|---|---|
| Automation can find a base manifest | At least one already released version exists upstream under the new ID. |
| Catalog migration is complete | Every agreed version has moved, paired PRs have completed, and the public index reflects the result. |
| Existing users have a supported transition | Windows tests demonstrate the chosen upgrade or reinstall procedure, including cleanup and preservation of user data. |

The [pinned WinGet Releaser action](https://github.com/vedantmgoyal9/winget-releaser/blob/b3a5dae0047c6180023acba3f548c55fdf6b7193/action.yml#L43-L49)
fails when the configured ID's upstream directory is absent. It subsequently
uses [`komac update`](https://github.com/vedantmgoyal9/winget-releaser/blob/b3a5dae0047c6180023acba3f548c55fdf6b7193/action.yml#L87-L97),
so changing the workflow cannot create the initial listing. An existing release
can seed it; a fresh PicFetch release is unnecessary for that step.

[Release](../.github/workflows/release.yml) publishes GitHub assets before the
independent [WinGet workflow](../.github/workflows/winget.yml) runs. A successful
GitHub release therefore does not establish successful WinGet publication.

## Dated observations to recheck

The investigation on **September 13, 2026, around 11:30 UTC** found:

- The [old upstream directory](https://github.com/microsoft/winget-pkgs/tree/master/manifests/i/io/github/frathe/picfetch)
  contained ten versions: `0.2.11`, `0.2.14`, `0.2.15`, `0.2.16`, `0.2.17`,
  `1.0.0`, `1.0.1`, `1.0.2`, `1.0.3`, and `1.1.0`.
- The [new directory API](https://api.github.com/repos/microsoft/winget-pkgs/contents/manifests/f/frathe/picfetch?ref=master)
  returned 404. No PicFetch migration PR was found.
- [Trenly suggested the shorter ID and offered help](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559).
  [Ronin's September 12 reply](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5647199327)
  was still unanswered. An individual reply is not a prerequisite if the
  documented process and technical questions can otherwise be resolved.
- The [1.1.0 installer manifest](https://github.com/microsoft/winget-pkgs/blob/master/manifests/i/io/github/frathe/picfetch/1.1.0/io.github.frathe.picfetch.installer.yaml#L4-L20)
  uses ZIP archives with nested portable `picfetch.exe`, command alias
  `picfetch`, and separate x64/ARM64 assets. It has no `UpgradeBehavior`,
  `ProductCode`, or `AppsAndFeaturesEntries` fields.
- The workflow pinned WinGet Releaser to
  `b3a5dae0047c6180023acba3f548c55fdf6b7193`. The local `website` branch had an
  unpushed shorter-ID commit, `6cc1efc`, ahead of remote `website` at `f413e7`.
  These are historical pointers, not instructions to push or discard that work.

## Portable installation uncertainty

WinGet [derives a portable product code from the manifest ID and source ID](https://github.com/microsoft/winget-cli/blob/master/src/AppInstallerCLICore/Workflows/PortableFlow.cpp#L25-L40)
and [uses it in the default installation directory](https://github.com/microsoft/winget-cli/blob/master/src/AppInstallerCLICore/Workflows/PortableFlow.cpp#L154-L180).
Keeping the same executable does not preserve this generated identity. Catalog
correlation and successful cleanup are separate things to test; MSI/Inno
installer assumptions do not establish portable behavior.

The CLI has an [`uninstallPrevious` path for portable installers](https://github.com/microsoft/winget-cli/blob/master/src/AppInstallerCLICore/Workflows/InstallFlow.cpp#L399-L463),
including execution through an archive's nested installer. It is only a candidate:
the update must first be recognized, and no PicFetch migration using this
mechanism has been proved. Adding metadata alone is not an accepted solution.

Old exact-ID commands and saved WinGet exports also need a transition plan.
The [manifest redirect issue](https://github.com/microsoft/winget-cli/issues/1899)
is a proposal, not evidence that an old-ID redirect works for PicFetch.

## Ticket 01 — Refresh inventory and maintainer procedure

**Purpose:** Replace historical assumptions with a current, bounded work list.
**Depends on:** Ronin resuming this deferred effort.

1. Record `git status --short`, the local branch/HEAD, live remote main and website
   revisions, latest release, and current WinGet workflow ID/action pin. Preserve
   concurrent edits. Locate every main/website reference to either package ID.
2. Query both upstream manifest directories, enumerate every version/file, and
   record the upstream commit. Read all later comments/reviews on PR 433339 and
   search current PicFetch PRs before preparing duplicate work.
3. Recheck [Trenly's per-version procedure](https://github.com/microsoft/winget-pkgs/discussions/316247#discussioncomment-15053420)
   and the [accepted all-version/correlation guidance](https://github.com/microsoft/winget-pkgs/discussions/236595#discussioncomment-12460004).
   The documented process uses one removal PR and one addition PR per version,
   submitted around the same time, with no combined add/remove PR. Ten versions
   would mean twenty PRs; recompute from the refreshed inventory. The earlier
   answer's simple-move advice assumes unchanged Registry/ARP metadata; it does
   not establish PicFetch's portable transition. It also warns that similar
   metadata across two IDs can confuse installed-package correlation.
4. Record any unresolved maintainer questions, particularly portable correlation
   and the ordering of merges. Use current documented answers when sufficient;
   seek clarification for unresolved behavior rather than waiting solely for a
   personal reply from Trenly.

**Acceptance / verification:** Retain `gh api` directory outputs, current PR and
comment URLs, `git status`/revision output, and a table of version, old path, new
path, asset URLs/hashes, and expected paired PRs. Every process question has an
answer or a named dependency on Ticket 03/04.

**Output:** Dated inventory and agreed procedure; no external submissions.

## Ticket 02 — Preserve realistic Windows baselines

**Purpose:** Make existing installations available for repeatable experiments
and later public-source verification.
**Depends on:** Ticket 01.

1. Prepare disposable Windows environments for x64 and ARM64 where available.
   Record OS, architecture, WinGet version, source identifiers, and install scope.
   Record unavailable coverage explicitly; x64 evidence does not qualify ARM64.
2. Install PicFetch using the old ID from the actual `winget` source. Cover the
   current version and an available older version, so both a same-version ID
   transition and an upgrade to a newer version can be tested. Record exact
   `winget install --id io.github.frathe.picfetch --exact --source winget`
   commands, including the selected `--version` and architecture.
3. Launch PicFetch, set recognizable preferences, save a favorite/session using
   disposable images, and identify the actual preferences/cache locations. Save
   backups and a WinGet export. Record registration, install directory, executable
   hash/version, and the `picfetch` command target.
4. Save VM snapshots with the old-ID installations intact. Retain them through
   Ticket 07; do not depend on reinstalling from a catalog entry after its removal.

**Acceptance / verification:** Keep `winget --info`, `winget source list`,
`winget list --id io.github.frathe.picfetch --exact`,
`winget export --output before.json --include-versions`, and
`Get-Command picfetch -All` output, plus registration/folder inventories, backups,
snapshot identifiers, and observed launch/preferences evidence.

**Output:** Reusable baseline snapshots and a matrix of versions, scopes,
architectures, and source identities actually covered.

## Ticket 03 — Prove a portable transition in the lab

**Purpose:** Establish whether an automatic upgrade or a documented one-time
reinstall is supportable before any production rollout decision.
**Depends on:** Tickets 01 and 02.

1. Inspect the WinGet version/source relevant to the baselines. Trace package
   correlation, generated portable identity, install location, alias ownership,
   and uninstall cleanup. Pin source references to the inspected revision.
2. Prepare private trial manifests/catalogs from existing release assets. Change
   only candidate migration fields, preserving binary URLs/hashes. Test a plain
   ID change first, then individually justified candidates such as
   `uninstallPrevious`. Do not publish these experiments to the community source.
3. Restore snapshots between trials. Observe which package `winget list` and
   `winget upgrade` associate with the old installation, then run the candidate
   transition and a subsequent upgrade under the new ID. Include fresh installs,
   same-version transitions, older-to-newer transitions, temporary catalog
   overlap with both IDs visible, and the final state after old-ID removal.
4. Check installed version and launch, old/new registration and folders, alias
   target/conflicts, and retained preferences, favorites/session, and cache.
   Check old exact-ID commands and export/import behavior in disposable accounts.
5. If automatic migration fails, test an explicit one-time uninstall/reinstall
   procedure with backups, exact commands, and restoration checks. Do not remove
   user-data directories blindly. Test interruption/failure recovery as well.
6. Record original and candidate install methods and source IDs for every trial.
   A community-to-local test changes both identity components. Local manifests,
   private sources, and manual registry edits cannot prove the ordinary public
   upgrade experience. Carry the same-community-source check into Ticket 07.

**Acceptance / verification:** Each matrix row has actual commands, exit codes,
WinGet logs, before/after inventories, `winget list`/`Get-Command` output, and
launch/data-preservation results. The chosen procedure leaves one intended active
installation and a working alias, accounts for old registration/folder cleanup,
and demonstrates the next upgrade. Failed candidates and untested platforms are
explicit. If no safe procedure is demonstrated, the result is continued deferral.

**Output:** Tested procedure or a failure report, coverage limits, public-source
acceptance steps, and draft recovery/user instructions.

## Ticket 04 — Decide whether the benefit justifies migration

**Purpose:** Make the proceed/defer decision against concrete evidence.
**Depends on:** Tickets 01–03.

1. Present the shorter-ID benefit, refreshed PR workload, tested transition,
   export/script impact, architecture coverage, and remaining source-specific risk.
2. Have Ronin choose continued deferral or the tested rollout. Record any accepted
   coverage limitation, the public verification gate, and the supported user path.
3. If proceeding, agree a release pause during the upstream move so automation
   does not create another old-ID version. Choose owners for paired PR tracking,
   Windows acceptance, main/website changes, and user support.
4. Set a failure response for partial publication. Once entries have been
   published or users have moved, reverting one identifier is not a demonstrated
   rollback. Use the tested recovery procedure and coordinate catalog repair.

**Acceptance / verification:** A dated decision links the Ticket 03 evidence,
names the selected procedure and unresolved limits, and records the rollout
scope/owners. A defer decision stops here and retains the old ID.

**Output:** Recorded proceed/defer decision and, only if proceeding, rollout scope.

## Ticket 05 — Prepare and validate the final manifests

**Purpose:** Produce reviewable per-version changes without submitting them.
**Depends on:** A proceed decision in Ticket 04.

1. In an isolated fork checkout, refresh the inventory against upstream again.
   Add any intervening versions or return to Ticket 04 if scope has changed.
2. Prepare each version under `manifests/f/frathe/picfetch/<version>/`, renaming
   all manifest filenames and their `PackageIdentifier` fields consistently.
   Preserve locale completeness, version, portable fields, architecture,
   release URLs, and checksums. Apply only migration metadata proved in Ticket 03.
3. Prepare separate removal changes for the corresponding old version. Keep each
   addition/removal pair independently reviewable. Link the pair and explain the
   approved transition in draft PR text.
4. Run `winget validate --manifest <version-directory>` on every addition using
   the selected Windows tooling. Compare parsed manifests with the originals:
   URLs/hashes and app identity must be unchanged; document every other difference.
5. Inspect [Tools/YamlCreate.ps1 move mode](https://github.com/microsoft/winget-pkgs/blob/master/Tools/YamlCreate.ps1#L2600-L2669)
   before choosing automation. After confirmation it commits and pushes two
   branches per version. PR creation depends on `AutoSubmitPRs` and prompts;
   disabling PR submission does not prevent pushes. Reserve this external-write
   operation for Ticket 06, not a preparation dry run.

**Acceptance / verification:** Retain validation output for every version,
`git diff --check`, per-pair diffs, and a comparison report showing preserved
URLs/hashes and complete version/file parity. No upstream submission has occurred.

**Output:** Validated changes, draft paired PR descriptions, and submission ledger.

## Ticket 06 — Submit and track the paired upstream PRs

**Purpose:** Execute the agreed catalog move as an explicitly external action.
**Depends on:** Tickets 04 and 05; rollout authorization and release pause in place.

1. Recheck maintainer guidance and any newly opened migration PRs. Confirm who
   will submit each pair, avoiding duplicate submissions with a maintainer.
2. Submit one removal and one addition PR per version around the same time,
   cross-linking the pair and migration explanation. Use the reviewed tool or
   manual submission path. Follow the merge sequencing agreed with maintainers;
   submission order alone does not guarantee merge or index order.
3. Track checks, review requests, merge revisions, and publication results for
   every pair. Fix validated findings without silently changing the tested
   portable strategy; material changes return to Ticket 03/04.
4. Track the first new-ID version separately as the automation bootstrap point.
   Keep the full migration open until all agreed versions and removals complete.
   If a pair fails or only partly publishes, follow the Ticket 04 response.

**Acceptance / verification:** The ledger contains two PR URLs per inventoried
version, successful validation/publication evidence, and final merge revisions.
No combined add/remove PR or unexplained missing version remains.

**Output:** Completed PR ledger and public catalog publication evidence.

## Ticket 07 — Verify the public index and existing-user path

**Purpose:** Check the real source before advertising or enabling the new ID.
**Depends on:** Ticket 06; preserved Ticket 02 snapshots.

1. Refresh the public source with `winget source update`. Use
   `winget show --id frathe.picfetch --exact --source winget --versions` and
   upstream directory queries to verify the complete expected version set.
   Record the old ID's actual behavior too; do not presume a redirect.
2. Restore snapshots originally installed from the community source, refresh
   that same source, and execute the exact selected transition. Record source
   identifiers before/after. Repeat Ticket 03 checks for covered architectures
   and scopes, including a subsequent ordinary upgrade where versions permit.
   A local-manifest workaround or manual registry repair is not proof of the
   supported public-source upgrade path.
3. Test a clean new-ID installation and a saved old-ID export. Confirm the user
   instructions for changed exact-ID commands and exports, including any manual
   action. Record the observed result rather than inferring it from PR merges.
4. If public behavior differs from the lab, pause the automation/docs switch and
   use the agreed support/catalog-repair process. Preserve logs and remaining
   old-ID snapshots. Do not improvise ID reversions or delete user data.

**Acceptance / verification:** Keep source-update/show/list/upgrade logs and
before/after registration, folder, alias, executable/version, launch, and
preferences/cache evidence. Public results meet the Ticket 04 decision; remaining
coverage limits are explicitly accepted or keep rollout paused.

**Output:** Public-source acceptance report and final tested transition instructions.

## Ticket 08 — Switch main's WinGet automation

**Purpose:** Make future version submissions use the now-qualified catalog ID.
**Depends on:** Ticket 07.

1. Inspect current main and concurrent edits. Confirm the live action's bootstrap
   requirement, then change only the WinGet identifier and its explanatory
   references in `.github/workflows/winget.yml` to `frathe.picfetch`.
2. Preserve the Windows ZIP asset filter, tag/release gates, token configuration,
   and action pin unless a separate reviewed fix is required. Keep the internal
   app ID in `main.go`, `FyneApp.toml`, and `Makefile` unchanged.
3. Run `go test ./scripts/wingettag -count=1` and `git diff --check`; inspect the
   workflow diff and retained identity fields. Land the reviewed change under
   the rollout authorization before resuming release submissions.

**Acceptance / verification:** Focused checks pass; the reviewed diff contains
no app-identity or unrelated change; live main uses the new WinGet ID and the
upstream base manifest is present. These checks do not yet prove a new submission.

**Output:** Workflow change/revision and focused validation evidence.

## Ticket 09 — Reconcile README, website, and user instructions

**Purpose:** Publish commands only after they work, with an honest transition path.
**Depends on:** Ticket 07; coordinate landing with Ticket 08.

1. Recheck references on main and the separate website branch. Inspect the fate
   of local commit `6cc1efc` before reusing anything; preserve unrelated changes.
   Keep website work on its own branch and use its current generation/check flow.
2. Update README installation commands and website source/generated variants to
   the new ID. Add the tested existing-user procedure, any reinstall requirement,
   and instructions for updating saved exports and scripts using the old ID.
3. Preserve historical evidence references and the internal app ID. Do not perform
   an indiscriminate replacement of every occurrence of the old identifier.
4. Validate links, website generation checks, and documented commands in Windows.
   Publish the reviewed website change separately after the new ID resolves.

**Acceptance / verification:** Targeted `rg` results account for old/new ID
references; main and website diffs have the intended scope; website checks pass;
the live README/site show working commands and the tested user procedure.

**Output:** Main/website revisions, command evidence, and published support guidance.

## Ticket 10 — Validate the next release and close the migration

**Purpose:** Prove ordinary operation and retain a useful recovery record.
**Depends on:** Tickets 08 and 09; normal release gates independently satisfied.

1. For the next separately authorized release, record the GitHub Release result,
   WinGet workflow result, generated new-version PR, validation/merge, and public
   availability. A successful action that opens a PR is not final publication.
2. Verify installation and upgrade to that version on the qualified Windows
   environments, including version, launch, alias, cleanup, and saved state.
3. If WinGet submission fails after GitHub assets publish, inspect the failure
   and fix the relevant workflow/catalog issue. The workflow's
   `workflow_dispatch` input can retry that published tag using updated main.
   Check existing PRs/manifests first: do not submit an already seeded identical
   version again, create a duplicate PR, or replace immutable release assets.
4. Record tested recovery and escalation steps for partial catalog publication,
   failed transitions, and obsolete old-ID commands/exports. Recovery must match
   the actual published state; do not assume changing the ID back is harmless.
5. Close the tickets only when the inventory, public-source transition, main,
   website, and next-release evidence are complete. Link the record from the
   active work tracker when this effort is resumed. Retain unresolved limitations
   explicitly rather than marking them tested.

**Acceptance / verification:** Retain workflow/PR/publication links, final
`winget show` and installation/upgrade logs, the completed ledger, and a dated
closure decision. General release work, such as dependency-notice delivery in
[todos.md](../todos.md), remains a separate gate and is not implemented here.

**Output:** Completed migration record and tested release/recovery instructions.
