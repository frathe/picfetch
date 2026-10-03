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
  existing normalization step handles this; run the complete suite again against
  the final signed, normalized candidate. No installer pass is claimed yet.
- GoLand IDE fallback inspected package.py, test_package.py and FyneApp.toml
  with errorsOnly=false (weak warnings included): zero reported findings. Local
  Qodana itself was not run; this profile is not an exact qodana.starter substitute.

Fresh hosted review gates and final signed-candidate qualification remain pending.
Final exact-head hosted results and signed candidate path will be recorded on
PR #75 to avoid changing the reviewed source after the final round.
