# PR 61: completing the FOSSA licensing review

Checked against FOSSA's official documentation on 2026-09-26. This document
separates repository compliance work from the FOSSA decisions needed to clear
the check. It does not record a completed FOSSA review or a passing remote scan.

## Evidence and proposed dispositions

The first CSV contained eight active **Flagged** issues. The second export,
`CSV_Licensing_ISSUES_2026-09-26_161730994Z.csv`, contains those same eight rows
unchanged plus five **Denied** issues. This reconciles the original 13-versus-eight
discrepancy. Both exports still describe revision
`cf24b842e471a3f5dc32e130b38ed8798b76f2b0`, analyzed at 14:25 UTC on
2026-09-26, before the notice fix. Export time is not analysis time.

The live check on notice-fix commit
`f0ed64a0ea319d16bd49013ab13c580d3b6355b3` reports **15 issues**, updated at
15:05:37 UTC. Its complete current issue list is not in either supplied CSV.
Open the latest PR check's Details link before exporting or resolving issues.
The CSV's `originPaths` identify dependency discovery, not licensed source file
matches. The following reasons are proposals for the project owner to validate
and record, not approvals already made in FOSSA or a legal opinion.

| CSV issue | License and component | Proposed project/version disposition |
| --- | --- | --- |
| 21232391 | BitstreamVera; Fyne v2.8.0 | **Accept documented use.** PicFetch ships Fyne's unchanged DejaVuSansMono-Powerline font, not a standalone font product. The full supplied Bitstream/Arev terms and attribution are now in the shipped notice document. Do not remove this real font-license detection. |
| 21232390 | LGPL-2.1-or-later; GLFW `v0.1.0-pre.1.0.20260707082822-2a407d02d01a` | **Accept only the reviewed header use.** Windows GLFW includes the Wine-derived `dinput.h` and `xinput.h` interface headers. These provide declarations, layouts, constants and short invocation macros; no Wine runtime is bundled. The LGPL 2.1 section 5 header-use provision is the relevant rationale, not an assertion that all static LGPL linking is harmless. Full header attributions, LGPL text and exact upstream source are now supplied. Reassess if File Matches includes other implementation code. |
| 21232392 | MPL-2.0; json-canonicalization `v0.0.0-20241213102144-19d51d7fe467` | **Not in the shipped build, if matches agree.** The MPL notices are in the Java/C# number-conversion implementations. PicFetch selects the Apache-2.0 Go package `go/src/webpki.org/jsoncanonicalizer`; its Apache terms are already shipped. Keep the upstream mixed-license classification intact. |
| 21232394 | CC-BY-SA-4.0 (**Denied**); go-digest v1.0.0 | **Not distributed, if matches agree.** The upstream README expressly assigns this license to `README.md` and `CONTRIBUTING.md`, via `LICENSE.docs`; the Go implementation is Apache-2.0. PicFetch selects five Apache-licensed Go files, with no embedded files, and does not package those upstream documents. Its Apache terms are already shipped. Record a project/version exception for the documentation-only match, not a global CC-BY-SA allowance. |
| 21232398 | CC-BY-SA-1.0 (**Denied**); x/text v0.42.0 | **Not in the shipped build, if matches agree.** The identified Russian/Hebrew excerpts are test samples. All six production target dependency lists exclude the sample package and `_test.go` files. Record the test-only scope; retain the module's BSD notice. |
| 21232395 | CC-BY-SA-2.0 (**Denied**); x/text v0.42.0 | **Not in the shipped build, if matches agree.** The identified Japanese/Korean excerpts are test samples excluded from all six production targets. Use the same scoped test-only disposition, with the version-specific source lines below. |
| 21232397 | CC-BY-SA-2.5 (**Denied**); x/text v0.42.0 | **Not in the shipped build, if matches agree.** The identified Chinese excerpts are test samples excluded from all six production targets. Do not change the license of the upstream module as a whole. |
| 21232396 | CC-BY-SA-3.0 (**Denied**); x/text v0.42.0 | **Not in the shipped build, if matches agree.** The identified Vietnamese/Greek/Arabic/Thai excerpts are test samples excluded from all six production targets. Resolve only the validated sample matches in this project/version. |
| 21232393 | openssl-ssleay; x/crypto v0.57.0 | **Provisional: inspect File Matches first.** Source search found OpenSSL/CRYPTOGAMS ancestry in `chacha20/chacha_ppc64x.s`, but no literal SSLeay/Eric Young notice. That assembly is constrained to ppc64/ppc64le and excluded from all six amd64/arm64 releases. If this is the complete match, record “not compiled into supported release targets”; otherwise review the actual matched files before resolving. |
| 21232402 | Apache-2.0 WITH LLVM exception; PicFetch root | **Accept documented use, if matches agree.** The AVIF WASM closure includes compiler/runtime components under these terms. The complete reviewed upstream aggregates, source revisions and payload hashes are already retained by `scripts/avifnotices` and shipped in the notices. This does not change PicFetch's own MIT license. |
| 21232399 | LGPL-3.0-or-later; PicFetch root | **Accept the scoped binding, if matches agree.** `internal/heic/libheif_abi.h` openly identifies its adapted libheif v1.17.6 ABI declarations. Source attribution and full GPL/LGPL texts are retained; Linux loads the system libheif dynamically, without bundling it. Review against LGPL 3 section 3 for these layout declarations. This rationale does not approve bundling a future libheif binary. |
| 21232401 | GPL-3.0-only; PicFetch root | **Provisional: inspect File Matches first.** If all matches are the complete GPL text supplied with the LGPL notices, record that this is required third-party notice delivery, not GPL-only licensing of PicFetch implementation. Do not delete the text to silence detection. If implementation files match, investigate them separately. |
| 21232400 | LGPL-3.0-only; PicFetch root | **Provisional: inspect File Matches first.** If all matches are copies of the LGPL version 3 license text, record their notice-only role; the identified libheif-derived declarations explicitly use LGPL-3.0-or-later. Keep that real finding and its obligations separate. Additional implementation matches require review. |

For root-level findings, **Selected version** refers to the scanned Git revision,
not a Go module version. A new commit can require reviewing these again; apply
the decision to the actual latest PR revision, not only the historical CSV SHA.

### Source/build evidence for those reasons

- **Fyne:** the pinned
  [font directory](https://github.com/fyne-io/fyne/tree/26a8e80cfa2b008c4eeb9be2f93772f69d582b66/theme/font)
  supplies the complete DejaVu/Arev, Noto OFL and Inter OFL terms. The
  `no_emoji` build tag does not remove the DejaVu/Noto/Inter fonts. The DejaVu
  font already contains a Bitstream notice internally; this change makes the
  full supplied terms explicit in the external, offline notice delivery too.
  Noto's copyright/trademark is also retained from the embedded font metadata.
- **GLFW:** pinned
  [Windows CGo flags](https://github.com/go-gl/glfw/blob/2a407d02d01a80ed49591d6d4e73fd1e385af304/v3.4/glfw/build.go)
  include `glfw/deps/mingw`;
  [win32_platform.h](https://github.com/go-gl/glfw/blob/2a407d02d01a80ed49591d6d4e73fd1e385af304/v3.4/glfw/glfw/src/win32_platform.h)
  includes both reviewed headers. The relevant permission is
  [LGPL 2.1 section 5](https://www.gnu.org/licenses/old-licenses/lgpl-2.1.en.html#SEC5)
  for numerical parameters, layouts/accessors and small macros/inline functions
  of ten lines or fewer. The separately embedded GLFW implementation's zlib
  license is now included alongside the already-present Go wrapper's BSD terms.
- **Canonical JSON:**
  [jsoncanonicalizer.go](https://github.com/cyberphone/json-canonicalization/blob/19d51d7fe467/go/src/webpki.org/jsoncanonicalizer/jsoncanonicalizer.go)
  and
  [es6numfmt.go](https://github.com/cyberphone/json-canonicalization/blob/19d51d7fe467/go/src/webpki.org/jsoncanonicalizer/es6numfmt.go)
  carry Apache-2.0 headers. The four MPL files are
  `java/deprecated/org/webpki/jcs/NumberDToA.java`,
  `java/deprecated/org/webpki/jcs/NumberFastDtoaBuilder.java`,
  `dotnet/es6numberserializer/NumberDToA.cs` and
  `dotnet/es6numberserializer/NumberFastDToABuilder.cs`.
  The existing updater inventory checks the selected Go package on all six
  supported OS/architecture combinations.
- **x/crypto:**
  [the PowerPC assembly header and build constraint](https://github.com/golang/crypto/blob/3f62bf119e84c6e35e8518a2958089ade622d1a3/chacha20/chacha_ppc64x.s)
  support the conditional exclusion rationale, but do not prove which source
  FOSSA matched. The module's BSD notice is already shipped.
- **go-digest:** the upstream
  [copyright/license section](https://github.com/opencontainers/go-digest/blob/v1.0.0/README.md#copyright-and-license)
  distinguishes Apache-licensed code from the two CC-licensed documentation
  files. Its wording omits “ShareAlike” in one sentence, but
  [LICENSE.docs](https://github.com/opencontainers/go-digest/blob/v1.0.0/LICENSE.docs)
  and the next sentence identify CC-BY-SA 4.0. Do not classify this as simply
  CC-BY 4.0. Production selection is `algorithm.go`, `digest.go`, `digester.go`,
  `doc.go` and `verifiers.go`, each with an Apache header; `EmbedFiles` is empty.
  The release workflow and MSIX staging copy the application and PicFetch's
  explicit notice/privacy files, not the upstream README/contribution docs.
  The existing [updater inventory](../scripts/updaternotices/manifest.json)
  already supplies this module's full Apache license.
- **x/text:** the exact
  [v0.42.0 module source](https://proxy.golang.org/golang.org/x/text/@v/v0.42.0.zip)
  contains the identified samples in `internal/testtext/text.go`,
  `cases/map_test.go` and `unicode/norm/normalize_test.go`. All 23 imports of
  `internal/testtext` within this module are from `_test.go` files.
  The source attribution lines for each detected version are:

  | Version | `internal/testtext/text.go` | `cases/map_test.go` | `unicode/norm/normalize_test.go` |
  | --- | --- | --- | --- |
  | 1.0 | 35, 60 | 869 | 1256, 1281 |
  | 2.0 | 79, 94 | None | 1290, 1308 |
  | 2.5 | 86 | 861 | 1315 |
  | 3.0 | 23, 43, 52, 69 | 851, 876 | 1244, 1264, 1273, 1298 |

  Production `go list -mod=readonly -tags=no_emoji,nodynamic -deps -json .`
  with `CGO_ENABLED=1`, for Darwin/Linux/Windows on amd64/arm64, completed
  without package errors at `f0ed64a`. Every target selects 29 x/text packages
  but not `internal/testtext` or `cases`; none selects or embeds these test
  sample files. This is build-selection evidence, not a cross-compilation or
  the missing FOSSA File Matches. If FOSSA names different files, stop and review
  them. No dependency replacement, extra CC license text or scanner exclusion
  is needed for the identified non-distributed samples.
- **PicFetch root:** see [the binding attribution](../internal/heic/libheif_abi.h),
  [its notice/source-delivery record](../internal/heic/notices/README.md), and
  [the AVIF notice manifest](../scripts/avifnotices/manifest.json).
  [LGPL 3 section 3](https://www.gnu.org/licenses/lgpl-3.0.html#section3)
  distinguishes header material from other combined works. Both GPL and LGPL
  texts accompany the binding; their presence does not by itself relicense
  unrelated project code. This is source-based reasoning awaiting the actual
  FOSSA matches where marked provisional above.

The repository change retains the MIT project license and every pre-existing
third-party license. It adds complete Fyne font, native GLFW and Wine-derived
header notices, exact module-source links, and an automated pinned-source test
in `make check-updater-notices`. Existing packaging and embedded-resource tests
cover delivery of the entire notice document. No dependencies, native binaries,
fonts or scan exclusions are changed.

## Why repository changes do not clear every finding

FOSSA's **Flagged** findings represent a policy decision requiring review, not
proof that notices are missing. The Standard Bundle Distribution policy flags
copyleft licenses for that review. Adding complete notices can satisfy a real
distribution obligation while leaving the policy flag active. Existing project
policy assignments cannot be changed by committing `project.policy` in
`.fossa.yml`; FOSSA explicitly requires project settings for that change.
[Licensing policies](https://docs.fossa.com/docs/policies/licensing-policies).

**Denied** means the assigned policy rejects a detected license. It does not
establish that the licensed upstream documentation/test material is part of
PicFetch's distribution. For the five CC findings, validate the file scope and
record the project/version disposition above. Do not mark CC-BY-SA globally
allowed, overwrite a module's real mixed-license classification, or remove
unrelated required notices.

There is no documented repository-only issue-resolution mechanism in the
configuration interfaces reviewed below. Clearing the remaining valid flags
requires an authenticated FOSSA account with permission to resolve licensing
issues for PicFetch. Retain the licensing check and record narrow, evidenced
decisions instead of disabling enforcement.

## Configuration capabilities and limits

| File or setting | Supported purpose | Limit relevant to this PR |
| --- | --- | --- |
| `.fossa.yml` or `.fossa.yaml` | CLI configuration with `version: 3`; both default filenames are recognized. | Neither is a list of issue approvals. The CLI parser has no per-dependency license-override field. |
| `targets.only`, `targets.exclude`, `paths.only`, `paths.exclude` | Choose local analysis targets and their discovery paths. | They do not rewrite the licenses detected in an upstream Go module. |
| `vendoredDependencies.licenseScanPathFilters` | `only` and `exclude` file-glob lists for CLI vendored license scans. | This is not a general exclusion list for registry dependencies or hosted issue findings. |
| `fossa-deps.yml`, `fossa-deps.yaml`, `fossa-deps.json` | Declare additional referenced, custom, remote or vendored dependencies. | These default filenames have no leading dot. `license` is allowed for custom dependencies, but is explicitly invalid for referenced dependencies. |

The configuration filename and schema checks come from the
[CLI parser](https://github.com/fossas/fossa-cli/blob/master/src/App/Fossa/Config/ConfigFile.hs).
Target settings are described in the
[configuration reference](https://docs.fossa.com/docs/cli/references/files/fossa-yml);
file filters in the
[vendored-scan reference](https://docs.fossa.com/docs/cli/features/vendored-dependencies);
dependency declarations in the
[fossa-deps reference](https://docs.fossa.com/docs/cli/references/files/fossa-deps).

These are documented CLI interfaces. FOSSA's hosted GitHub import is a separate
analysis path. Its public documentation does not establish that it honors the
above files identically; hosted support for those controls is unverified here.
The official comparison documents static hosted analysis and more extensive
CLI filtering. Do not promise a passing hosted check merely by adding one of
these files. A CLI migration would also require integration and credentials.
[CLI versus Quick Import](https://docs.fossa.com/docs/get-started/cli-vs-quick-import),
[GitHub Actions integration](https://docs.fossa.com/docs/integrations/github-actions).

## Actions in FOSSA

After the repository notice changes are available in the PR:

1. Open PR 61's **License Compliance -> Details** link. Verify that the FOSSA
   revision matches the latest PR commit, then open **Issues -> Licensing ->
   Active**.
2. Open each finding. Check **File Matches**, including all additional paths,
   and the dependency version against the evidence above. Do not resolve a
   provisional match if FOSSA names a different file.
3. Select the validated issue and choose **Actions -> Ignore**; some
   organizations label the same action **Resolve**. Choose **In this project**
   and **Selected version**. Include the evidence and disposition in the reason.
4. Keep unresolved findings active. Repeat for any findings present in the live
   revision but absent from the CSV export.

This scope applies to revisions of PicFetch using that exact package version;
new versions require review again. It does not create an ongoing auto-ignore
rule. Account permissions are required; confirm the available role controls if
the action is missing. [Issue review and scope controls](https://docs.fossa.com/docs/licenses/reviewing-licensing-issues).
The check's Details link identifies the affected revision.
[PR checks](https://docs.fossa.com/docs/project-setup/pr-checks).

Use this issue-level decision for a correctly detected license that is acceptable
in PicFetch's documented use. A **license correction** changes detection data:
FOSSA documents that dependency edits affect every project, revision and version
in the organization. Do not erase a real license from a mixed-license dependency
merely because PicFetch does not build the covered files.
[License corrections](https://docs.fossa.com/docs/licenses/license-corrections).

License Conclusion and its policy options are separate organization features.
They can change which detected licenses produce issues, but do not repair missing
attributions or establish that a license is inapplicable. They are unnecessary
for the narrow issue review above.
[License conclusions](https://docs.fossa.com/docs/licenses/license-conclusions).

## Confirming completion

Keep the repository's GitHub update hook enabled in **Settings -> Hooks**. A new
pushed revision triggers analysis, followed by an issue scan. After recording the
decisions, refresh/reanalyze the current PR revision through the project's
available controls and verify the actual GitHub License Compliance status. A
green local notice check is not evidence that the remote FOSSA gate passed.
[Automatic updates](https://docs.fossa.com/docs/project-setup/automatic-updates).

The second export resolves the original 13-versus-eight count discrepancy, but
still predates the notice fix. Export **all active licensing issue types** from
the latest PR revision, including both Flagged and Denied. The observed current
check has 15 issues; the additional two are not identified by the supplied data
and must not be guessed or pre-approved. Completion requires every live finding
to have a validated disposition and the check to pass on the latest PR commit.
