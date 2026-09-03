# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

- Reduced the measured median reusable CI gate from 22:17 to 7:52 while
  retaining Linux race coverage, native Windows tests, validation, and release
  failure gating. The final topology can use six concurrent required runners
  instead of two and raises median runner time from 23:26 to 30:43 (+31.1%).
  Its exact manifest proves UI coverage and non-overlap, but no one process now
  exercises the whole UI suite's ordering; see the
  [completed measurement record](finished_refactorings/2026-09-03-measured-ci-test-sharding.md).

## TODO

### Publish PicFetch in Microsoft Store

Build the Partner Center-reserved PicFetch product as one x64/ARM64 MSIX
bundle, make the Store build defer updates to Microsoft Store, retain the
portable GitHub/WinGet channel, validate on Windows with WACK, prepare the
English/German listing and privacy policy, then submit it for certification.
Implementation plan: `plans/2026-09-03-microsoft-store-msix.md`.

### Functional test coverage

Audit baseline, 2026-09-01: package-local statement coverage came from a
passing run of:

```sh
go test -count=1 -skip '^TestE2E' \
  -coverprofile=/tmp/picfetch-cover-clean.out ./...
```

"Effective" coverage below additionally instruments the named package while
running its existing higher-level tests. Percentages locate candidates;
completion means pinning useful PicFetch behavior through an established
interface, not reaching a blanket coverage target.

1. **P0 - `internal/update` (77.2% local / 78.8% effective).**
   - Through `update.Apply`, force a plist-backup failure after binary
     replacement begins (a conflicting `Info.plist.old` is the deterministic
     fixture). Verify the installed binary and plist remain the original
     versions and partial replacement files are removed.
   - Through `Client.Download` with an HTTP test server and fake external
     verifier, reject ZIP symlinks and TAR symlink/hardlink entries. No usable
     stage or file outside the staging directory may be created.
   - Do not exercise the real Sigstore implementation or unreachable platform
     stubs merely to cover their lines.
2. **P1 - `internal/filesort` (73.7% local / 89.9% effective).**
   - Pin `FromPref` / `PrefValue` round-trips for every `Modes` entry and the
     unknown-value fallback to `ByName`.
   - Verify missing files use the documented zero-key fallback for capture
     date, modification time, and size sorts, while `Order` leaves its input
     slice unchanged.
3. **P1 - `internal/ui/copyselection` (85.4% local / 88.1% effective).**
   - Through `Feature` and canvas gestures, verify all eight image-region
     selection handles resize the correct edges, including corner and crossed
     movement.
   - Move a committed image-region selection beyond each image edge and verify
     it clamps to the image while preserving its dimensions.
   - Do not test `resizeRect`, renderer no-ops, or Fyne event methods directly.
4. **P2 - `internal/favthumbs` (86.9%).**
   - Verify `Read` falls through from a corrupt JPEG preview to a valid PNG
     sibling.
   - Verify `Sweep` preserves directories and other non-regular files whose
     names resemble preview files.
5. **P3 - `scripts/releasenotes` (86.8%).**
   - Exercise `Build` and `ClearDone` with CRLF input and with `## Done` as the
     final section, preserving surrounding Markdown and changelog output.

Future tests stay at the exported seams named above. Each must be seen failing
against a deliberate behavior break before it is accepted, then pass its
focused package test and `make verify` at final handoff.

Do not chase the low numbers in `main`, `internal/uitest`, `scripts/synctuf`,
Fyne widget adapters, or the accepted OS-integration seams. `internal/session`
and `internal/favstore` already cover their useful persistence behavior; their
remaining gaps are chiefly framework/filesystem failure plumbing.
`internal/ui/autoupdate` and `internal/appearance` reach 91.4% and 96.4%
respectively when their higher-level tests are counted. Re-audit
`internal/ui/compare` only after the current command-isolation work lands; its
present uncovered input-shield methods are deliberate no-ops.

## LATER

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS
runner, in August 2027. Before then, decide whether PicFetch will stop
shipping an Intel macOS archive or retain it through another build path. If
Intel support remains, replace the `macos-15-intel` release job with a tested
alternative; otherwise remove the x86_64 artifact and update the release and
installation documentation. The native Apple-silicon build is not affected
by Rosetta's retirement.

## not deemed worth implementing (edge cases)

- Windows releases are not Authenticode-signed. Controlled Folder Access and
  SmartScreen both judge by signature and reputation as well as by which
  program is writing, so an unsigned `picfetch.exe` can still be blocked
  even with the in-process swap (see Done → Bugfix above, where the block
  would now name `picfetch.exe` instead of `cmd.exe`). The real remaining
  fix is signing the Windows release build — Azure Trusted Signing or a
  purchased certificate — in `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the
IDE's 71. The 8 fragments CI is missing are 7 in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries
exactly 3 `#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in
db" warnings, naming exactly those two files and no others, emitted
immediately after the line `The Project analysis stage completed in 41s` —
so Qodana's own log shows detection succeeded and serialisation into the
report/SARIF failed afterwards. This is an upstream defect, not a picfetch
config problem: nothing here suppresses or excludes those two files, and the
drop happens before any project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes
this defect invisible going forward in this repository, because every
dropped fragment happens to live in a test file that the exclusion now
removes from the inspection entirely — recorded here so the defect is not
lost along with the rule that used to surface it. Of the 12-fragment
CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the
other 4 are the source-suppressed production fragments in the orientation
pixel loops recorded above, so nothing about that gap is left open — only
the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte
offsets and anchoring detail, and `plans/2026-08-29-qodana-serialisation-bug-report.md`,
Task 8's draft of the upstream report text — as of this writing not yet
submitted to JetBrains; check that file for whether it has been sent since.
