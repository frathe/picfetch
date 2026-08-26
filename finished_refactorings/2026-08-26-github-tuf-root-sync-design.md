# GitHub TUF root embed: check, sync, release, monthly PR

## Problem

`internal/update/embed/tuf-repo.github.com/root.json` is the bootstrap trust
anchor for GitHub’s TUF mirror (`https://tuf-repo.github.com`). It is GitHub
TUF **root metadata** (currently `version` 9), not an X.509 cert. It expires
`2027-01-28T20:18:53Z`. Unversioned `root.json` on that mirror 404s; clients
follow `N.root.json`.

Runtime `NewSigstoreVerifier` already refreshes TUF into the user cache while
the embed is still valid. After the **embedded** root expires, a machine with
an empty cache cannot bootstrap. There is no in-app rotation. This spec
automates keeping the **repo** embed current, with a fail-closed expiry gate.

## Non-goals

- Changing attestation policy (SAN, `WithSignedTimestamps`, in-toto checks).
- Fetching or rewriting the embed inside `.github/workflows/release.yml`
  (tag-triggered packaging). Artifacts must match the tagged source.
- Hitting live GitHub or the TUF mirror from `go test` / PR CI except the
  optional monthly workflow.
- Auto-merging the monthly PR.
- Requiring a PAT so that GITHUB_TOKEN-created PRs run CI (optional later).

## Trust rule

Never overwrite the embed from an unsigned download. A newer root is written
only after TUF verification against the keys in the current dest file
(threshold signatures on `N+1.root.json`, repeating until there is no newer
verified root). Refuse to write a root whose `signed.expires` is already in
the past.

## Components

### `internal/update` helper (`tufroot.go`)

Two functions, kept out of `attest.go`:

- `CheckEmbeddedRoot(root []byte, now time.Time) error`  
  Parse `signed.expires` (RFC3339). Fail if missing/unparseable, if
  `now >= expires`, or if remaining time is less than **60 days**. No
  network. 60 days matches GitHub TUF-on-CI’s `x-tuf-on-ci-signing-period`
  on the current root and prevents tagging a release that dies before the
  next one.

- `SyncGitHubRoot(ctx context.Context, destPath string, hc Doer) (changed bool, err error)`  
  Read dest (the embed path in production). Bootstrap a sigstore-go TUF
  client with that file as `opts.Root`, `RepositoryBaseURL = GitHubTUFMirror`,
  `CachePath` a temp dir, and `hc` as the HTTP fetcher (httptest in tests).
  After a successful refresh, take the verified latest root bytes. If those
  bytes are already expired, return an error and do not write. If they equal
  dest, return `changed=false`. Otherwise write dest (same bytes GitHub
  served / the client stored; do not reformat JSON) and return
  `changed=true`.

`NewSigstoreVerifier` is unchanged (still caches under `…/picfetch/tuf`).

### CLI (`scripts/synctuf/main.go`)

Package `main`, imports `internal/update`. `go build ./...` compiling it is
intended.

- `--check`: `CheckEmbeddedRoot` on the repo embed path, `time.Now()`, exit
  non-zero on error. Offline.
- `--write`: `SyncGitHubRoot` on the repo embed path. Needs network (or the
  client we inject; the CLI uses a real HTTP client).

Default embed path: `internal/update/embed/tuf-repo.github.com/root.json`
relative to the current working directory. Makefile targets run from the
repository root. If the file is missing, the CLI errors with “run from the
repository root” and does not walk parent directories.

### Tests (`tufroot_test.go`)

httptest only. Cover:

- expired fixture → `CheckEmbeddedRoot` error
- expires in 59 days → error
- expires in 61 days → nil
- unsigned / wrong-key `N+1` → `SyncGitHubRoot` error, dest unchanged
- verified newer root → dest replaced, `changed=true`
- dest already latest → `changed=false`, no rewrite required

Do not call the live mirror from tests.

### Make / CI

- `make check-tuf-root`: `go run ./scripts/synctuf --check`
- `make sync-tuf-root`: `go run ./scripts/synctuf --write`
- `make verify` runs `check-tuf-root` immediately after `fmt-check` and
  before `vet` (fail cheap). Offline.
- `.github/workflows/ci.yml` runs `make check-tuf-root` in the existing
  job (after format, no extra GUI packages). Same gate as `make verify`.

CI does **not** fail a PR solely because a newer root exists on the mirror.
Runtime refresh still follows the chain while the embed is valid. The monthly
job and `make release` are what pick up a rotation.

### `make release` (two commits)

After dirty-tree / branch / tag checks, **before** `make verify`:

1. `make sync-tuf-root` (network to `tuf-repo.github.com`).
2. Existing `y/N` prompt: include the tag; if the embed is dirty, also say
   there will be a separate TUF-root commit before `Release vX`.
3. `make verify` (offline expiry check on whatever is now in the tree).
4. If the embed changed: `git add` that path, commit
   `Update GitHub TUF root` (no FyneApp.toml in this commit).
5. Existing version bump, `git add FyneApp.toml`, commit `Release vX`,
   annotated tag, push both commits then the tag.

Abort at the prompt or a red verify: no git commits. Sync may leave a dirty
embed; the operator discards or commits it by hand.

`.github/workflows/release.yml` does not run sync.

### Monthly PR

`.github/workflows/tuf-root.yml`:

- `schedule`: `0 9 1 * *` (09:00 UTC on the 1st of each month)
- `workflow_dispatch`
- `permissions`: `contents: write`, `pull-requests: write`
- Job: checkout default branch, setup-go from `go.mod`,
  `go run ./scripts/synctuf --write`. If the embed changed, push a branch
  and open a PR with `gh` (title/body state that this is a verified TUF
  root bump). No auto-merge.

GITHUB_TOKEN-created PRs often do not run CI. Accepted for v1. Human PRs
and `make verify` still fail if the embed is expired or inside 60 days. A
PAT secret is out of scope until we need CI on the bot PR.

## Docs

Update `ARCHITECTURE.md` (`internal/update` table: `tufroot.go`, embed path)
in the same change that adds the helper. Mention `make check-tuf-root` /
`make sync-tuf-root` in Makefile `help` only; no end-user manual section.

## Error handling

Failures are process errors (non-zero CLI, `make` stop, CI red). The
in-app updater still logs `update verifier unavailable` if a **shipped**
binary’s embed is somehow expired; this work is to keep that from shipping.
