# Store runtime version correction — 2026-10-03

## Scope and acceptance

Standard SDD, lead-owned implementation and review; no delegation. Apple rejected
version 1.1.11 because the in-app version was 0.0.1. Preserve marketing version
1.1.11 and advance the candidate build from 483 to 484. Commit and push to PR #75,
complete its code/security/CI review loop, then build a fresh signed installer.
No upload, submission, merge or release is authorized by this task.

Acceptance commands:
- `python3 -m unittest discover -s scripts/macstorepackage`: a real headless Fyne
  executable reports the configured version/build outside its checkout, including
  a future version, identity, release state, migrations and icon. Source files
  remain unchanged. Existing packaging policy tests pass.
- `make apple-store-package-local`: both native architectures retain their
  deployment bounds and the verified runtime/entitlement/resource closure.
- Inspect the built native app's About window: Version 1.1.11 (Build 484).
- GoLand inspections of both changed Python files include weak warnings; record
  unavailable/incomplete findings rather than treating them as a clean gate.
- After push: fresh clean code and security-focused Codex reviews, required hosted
  CI, CodeQL, exact-head Qodana post-suppression SARIF assessment and FOSSA statuses.
- After clean review: `make apple-store-package`, using the existing local signing
  setup, validates the new signed app and installer payload. No Apple upload.

## Diagnosis and implementation

The universal packager used raw `go build`, then supplied version/build only to
`fyne package --executable`. Fyne skips compilation/metadata injection for an
existing executable; About and other installed-version consumers therefore read
Fyne's default 0.0.1/build 1. Development TOML discovery can mask this error.
A minimal executable using the same build seam reproduced `0.0.1 != 1.1.11`
and `0.0.1 != 2.4.6` before the fix.

Compile-time `app.SetMetadata` is supplied through a staging-only Go overlay,
using the same captured FyneApp.toml values as Info.plist. The overlay embeds the
existing icon and preserves migrations, identity and release state. Canonical
paths handle macOS /tmp symlinks. Neither the checkout nor FyneApp.toml is mutated
by packaging. The raw architecture-specific Go/cgo build path and deployment
flags are preserved; the release tag disables development metadata discovery.
No package moves, new dependencies, runtime/model assets or license changes.

## Verification and review evidence

Local qualification on the reviewed working patch based on 37d7aeb:
- Runtime regression failed with 0.0.1 before implementation; both configured
  versions now pass. Full portable policy suite: 38 tests, 10 native-only skips.
- `make verify-build` passed formatting/generated-assets/notices/TUF checks,
  vet and build. Broad race suite is left to native amd64 GitHub CI per the loop.
- `make apple-store-package-local` passed both architecture/deployment, exact
  entitlements, signature, native dependency and resource guards. Bundle:
  `bin/apple-store-version-2026-10-03-worktree/PicFetch.app`.
- Manual native About screenshot shows Version 1.1.11 (Build 484):
  `/private/tmp/picfetch-version-1.1.11-build484-about.jpg`.
- Native `test_package.py`: all 14 tests pass, including the eight artifact guards.
  A full-suite attempt on the ad-hoc input passed the other cases but its installer
  payload check rejected upstream dylib owner-only permissions. Distribution's
  existing normalization step handles this; run the installer guards against the signed, normalized candidate below.
- GoLand IDE fallback inspected package.py, test_package.py and FyneApp.toml
  with errorsOnly=false (weak warnings included): zero reported findings. Local
  Qodana itself was not run; this profile is not an exact qodana.starter substitute.

Reviewed implementation commit: b69171bca23965297041f2e76e2af8e2af109251.
- Fresh code review: https://github.com/frathe/picfetch/pull/75#issuecomment-5967878037
- Separate security-focused review: https://github.com/frathe/picfetch/pull/75#issuecomment-5967926585
  Both report no findings; all 41 older/current threads are resolved.
- Full CI: https://github.com/frathe/picfetch/actions/runs/37113634421 (success),
  including every Linux race shard, Windows and both macOS native guards.
- CodeQL: https://github.com/frathe/picfetch/actions/runs/37113634403 (success),
  zero open PR alerts. All three exact-head FOSSA statuses pass.
- Qodana: https://github.com/frathe/picfetch/actions/runs/37113634419 (success),
  artifact 11270562626. Root post-suppression SARIF identifies b69171b and has
  the identical 11 assessed unused-export false positives as 3da5f98; current
  production callers, platform dispatch and callback references were rechecked.
  Its findings summary is neutral, not a zero-result report.
- `make apple-store-package` passed with the existing local signing setup.
  The local manifest is clean and source-bound to b69171b, version 1.1.11/build
  484. Distribution verification includes both provisioning profiles, every
  nested signature/entitlement, non-owner permissions, trusted installer leaf
  and extracted-payload equality/signature. No upload or submission occurred.
- Native distribution suite: all 12 test_distribution.py tests passed using the
  signed candidate, including both installer guards. Native bundle mutation
  guards use the ad-hoc qualification app (14 test_package.py tests passed).
  These input roles differ: do not treat the distribution app as an ad-hoc input.
- Signed candidate: `bin/apple-store-candidate-1xg3fxka/signed/PicFetch.pkg`.
  SHA-256: af4bdac7d3cf4ee158b9f5d259793485ed320c0eeff6b86da2abffff702bbb79
- Direct desktop launch of the distribution app failed with macOS spawn error
  163. The matching clean qualification process ran, but the Computer Use tool
  could not locate its window; that process was stopped. No reason for the
  distribution-launch refusal is claimed. Native visual version evidence above
  is from the pre-commit qualification app with the same metadata implementation.
  Apple-installed execution/acceptance remains an external submission gate.
- Automatic approval review rejected inspection by shared bundle identifier,
  because it could select another PicFetch copy. Explicit candidate-path retries
  and scoped process checks were used; no unrelated app content was inspected.

Implementation and signed delivery are complete. The documentation-only closing
commit requires fresh latest-head review/CI; its final result is recorded on PR #75.
Unchanged-code local inspections retain b69171b as their analyzed revision.
Final exact-head hosted results and signed candidate path will be recorded on
PR #75 to avoid changing the reviewed source after the final round.
