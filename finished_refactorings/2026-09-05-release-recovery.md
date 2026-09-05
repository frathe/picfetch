# Recover v1.0.1 without moving its tag

Route: Standard. Owner: T0 inline. No delegation. The scope grew from a Thin
cache-permission fix to recovery of the existing release.

## Evidence and decisions

- Run `33941450358`, job `101241780590`, fails writing Go's module cache.
  Root warm-up creates `/go/pkg/mod/cache/download` with mode 0755; fyne-cross
  subsequently runs as the host UID. A disposable Docker volume reproduced
  that exact permission failure, and warming as the host UID passed.
- The warm-up entered in `b80ee92` to prevent the Go toolchain download message
  from contaminating Fyne's JSON metadata input. It predates the Finis Easter
  egg (`4280c24`), which changed no build recipes or dependencies.
- v1.0.1 points to `4cbb9f6bf2e3d9225baa1138cb459fc6cf6db78c`. No GitHub release
  exists. Both macOS artifacts and all CI results from the failed run survive.
- Preserve the tag and existing artifacts. Do not replace or delete releases.
- Recovery runs the revised workflow, but every application checkout and CI
  checkout uses the resolved tag commit. Only cross-packaging takes its
  Makefile from the workflow revision. Signing retains the protected
  `release-signing` environment and existing verification steps.
- Manual recovery refuses an existing release and resolves the tag once to
  an immutable SHA. Publication uses the explicit resolved tag.
- No application packages, dependencies, translations, or version change.

## Tasks and acceptance

### 1. Preserve cache ownership

Files: `Makefile`, existing `scripts/msixstage/msixstage_test.go`.
Contract: the shared Windows warm-up uses the same UID and writable HOME as
fyne-cross. Release, debug and Store targets keep using that warm-up.
Verify: `go test ./scripts/msixstage -run TestWindowsToolchainWarmupUsesPackagingUser -count=1`;
`make package-windows FYNE_CROSS_CACHE=/tmp/picfetch-release-validation-cache`.
The regression was observed red before the recipe fix, then green.
Budget: 0 spawns, 1 review round, no full suite here.

### 2. Add tag-preserving recovery

Depends: 1. Files: `.github/workflows/release.yml`, `.github/workflows/ci.yml`,
existing packaging tests, `docs/release-signing.md`.
Contract: manual `release-tag` selects a tag, all source consumers use its
resolved SHA, the cross-build alone uses the workflow's corrected Makefile,
and publication remains dependent on CI, every build, and Windows signing.
Verify: packaging workflow regression tests and actionlint; inspect every
checkout and the unchanged signing job. Exercise the source resolver with
valid/missing/invalid tags, an existing release, and API failure.
Budget: 0 spawns, 1 review round, no full suite here.

### 3. Final gate and handoff

Depends: 1, 2. Files: `todos.md`, this plan.
Verify: `make verify`; `GOOS=windows GOARCH=amd64 go vet ./internal/...`;
Windows/Linux packaging; `git diff --check`.
Budget: 0 spawns, 1 final review, 1 full race suite.

## Activation after the reviewed change is committed

The user explicitly overrode the repository's no-commit instruction and
authorized committing, landing the reviewed repair on `main`, and recovering
the current release. The tag must stay unchanged. The existing signing
environment permits `main` and requires the repository owner's reviewer
approval; that gate remains in force.

After landing the reviewed change, run:

```sh
gh workflow run release.yml --repo frathe/picfetch --ref main -f release-tag=v1.0.1
```

Watch the resulting Release run through CI, builds, the existing signing gate,
and publication. Confirm all six platform archives exist and the tag still
resolves to the SHA above. Manual release recovery does not trigger the
push-only WinGet handoff; its existing manual workflow can be dispatched
with `release-tag=v1.0.1` after the GitHub release succeeds.

## Honest limits and ledger

Local cross-builds validate compilation and packaging, not Windows runtime
behavior. GitHub signing and publication can only be validated after landing
and dispatching the workflow. A previously root-owned persistent cache may
still need its ownership repaired; fresh GitHub runners do not have one.

Verified on 2026-09-05:

- The cache reproduction failed as UID 1001 after root warm-up, then passed
  with the same UID for both phases. The regression test likewise failed
  before the Makefile change and passed afterward.
- `make package-windows FYNE_CROSS_CACHE=/tmp/picfetch-release-validation-cache`
  and `make package-linux` passed for both amd64 and arm64. `file` confirmed
  the expected Windows PE and Linux ELF architectures.
- Exported `git archive v1.0.1` into a separate temporary directory, copied
  only the revised Makefile to `.release-tooling/Makefile`, and successfully
  ran both packaging targets with `make -f .release-tooling/Makefile`. This
  exercises the actual recovery layout using the original tag source.
- Executed the workflow's source resolver against GitHub: selected the
  unchanged tag and SHA recorded above. Regression cases cover missing or
  malformed tags, existing releases, API errors, and ordinary prerelease pushes.
- `go test -race ./scripts/msixstage ./scripts/wingettag -count=1`,
  `go vet ./scripts/msixstage`, and actionlint v1.7.12 passed.
- `make verify` passed: formatting, TUF root, Qodana exclusions, native vet
  and build, and all four Linux/amd64 race partitions. The new resolver cases
  were also rerun with the race detector after their final edits.
- `GOOS=windows GOARCH=amd64 go vet ./internal/...` passed.
- The entire Windows signing job is byte-for-byte unchanged. Local packaging's
  automatic Fyne build-number increments were restored; app metadata and tag
  source are unchanged in the reviewed diff.
- Final formatting/exclusion checks and `git diff --check` passed. No new
  package or test file was introduced. The earlier Finis verification TODO
  is now closed by the successful full suite.

| Task | Spawns budget/actual | Review rounds | Full suite |
|------|----------------------|---------------|------------|
| 1 | 0 / 0 | 1 | no |
| 2 | 0 / 0 | 1 | no |
| 3 | 0 / 0 | 1 | passed |
