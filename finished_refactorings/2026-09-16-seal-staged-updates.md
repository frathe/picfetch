# Seal staged updates before privileged installation

## Problem

The Windows updater applies cache-controlled `stage.json` metadata inside the
trusted PicFetch process. A same-user process can currently replace both the
staged executable and its claimed digest, causing PicFetch to cross Controlled
Folder Access on the attacker's behalf.

## Decisions

- Authenticate the complete persisted stage document with a process-ephemeral
  HMAC key. Stages do not survive a PicFetch restart; the next check downloads
  and verifies the release again. This deliberately trades cached-download
  reuse for a secret that is never persisted beside attacker-writable data.
- On Windows, open and hash the staged executable once, then copy from that
  validated handle and compare the installed file with the authenticated
  digest. Never reopen the mutable staged path during replacement.
- Keep release download and Sigstore verification unchanged.

## Acceptance criteria

1. A forged stage document containing self-consistent attacker bytes and
   digests is rejected by `LoadStage`.
   Verify: `go test -tags no_emoji,nodynamic ./internal/update -run 'TestLoadStage_RejectsForgedSeal|TestDownload'`
2. Windows replacement copies from the already validated file handle and
   verifies the destination against the authenticated digest.
   Verify: `GOOS=windows GOARCH=amd64 go test -tags no_emoji,nodynamic -c -o /tmp/picfetch-update.test.exe ./internal/update`
3. Existing updater behavior in the creating process remains intact.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/autoupdate ./internal/ui -run 'Test(Updater|ApplyStagedUpdate|PerformUpdate)'`

## Non-goals and honest limit

This does not preserve a completed update across application restarts. That is
intentional: durable unattended reuse requires retaining and re-verifying the
attested archive, which is broader than the minimal remediation. The normal
same-process automatic and manual update flows remain supported.

## Tasks and budget

| Task | Owner | Files | Test | Spawns | Full suite |
|---|---|---|---|---:|---|
| Seal metadata | T0 inline | `internal/update/download.go`, tests | forged JSON rejected | 0 | no |
| Bind apply handle | T0 inline | Windows apply/swap, tests | source replacement cannot change copied bytes | 0 | no |
| Final gate | T0 inline | plan/evidence | repository verification | 0 | yes |

No delegation: this is security-sensitive Windows behavior, which the working
agreement reserves for the lead.

## Evidence and cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Result |
|---|---:|---:|---|---|
| Seal metadata | 0 / 0 | 1 | no | Forged-stage guard passed and was negatively verified. |
| Bind apply handle | 0 / 0 | 1 | no | Handle/path replacement regression passed; Windows test binary cross-compiled. |
| Final gate | 0 / 0 | 1 | yes | See handoff test report. |

## PR #32 follow-up

The initial native Windows run exposed three failing tests. Review also moved
archive extraction and payload hashing onto the verified memory buffer, added
Windows write-denying source handles, and consolidated installation onto the
rollback path exercised by the existing tests. Current completion evidence is tracked in
[the PR review record](2026-09-16-pr32-review.md).
