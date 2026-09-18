# Release artwork redirect hardening

## Problem

Release-note artwork accepted HTTP URLs and used the default redirect behavior, so a trusted image response could redirect a GET to an arbitrary local or private service.

## Decisions

- Require HTTPS GitHub or GitHub-content hosts for both the original URL and every redirect.
- Preserve the existing bounded decoding, cancellation, and test-client seams.
- Do not add general-purpose private-network resolution; the narrow provider allowlist removes that attack surface.

## Acceptance criteria

1. Initial HTTP and non-provider URLs are rejected before transport.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/help -run TestLoadReleaseImageRejectsUntrustedDestinations`
2. A redirect to an HTTP/local destination is rejected.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/help -run TestReleaseImageClientRejectsUntrustedRedirect`
3. Existing release artwork formats, bounds, cancellation, and UI lifecycle behavior remain covered.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/help`

## Non-goals and honest limit

This does not make arbitrary remote images safe or supported. Release artwork is deliberately restricted to the GitHub providers used by the bundled notes; a future provider requires an explicit policy update.

## Tasks

### Task 1 — Pin destination policy
Owner: T0 inline
Files: `internal/ui/help/releaseimage.go`, `internal/ui/help/help.go`, tests
Contract: `releaseImageURL` validates initial/final destinations; `releaseImageRedirectPolicy` validates every followed redirect.
Test: Reject untrusted initial URLs and redirect targets.
Verify: `go test -tags no_emoji,nodynamic ./internal/ui/help`
Budget: 0 spawns · 1 review round · full suite: yes

### Task 2 — Keep architecture map accurate
Owner: T0 inline
Files: `ARCHITECTURE.md`
Depends: Task 1
Contract: Document the provider-restricted HTTPS artwork loader.
Test: Repository formatting and verification gates.
Verify: `make verify`
Budget: 0 spawns · 1 review round · full suite: yes
