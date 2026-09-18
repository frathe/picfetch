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

## PR #41 review follow-up (2026-09-18)

Route: Standard. Lead owns both standards/spec review and all fixes. Review
baseline: `5052c0e7cef9f32bfe709cb3f073d227bf7d7c1e`; initial PR head:
`4e35b2264558c020904bd2553784127d220abfae`. No dependency changes.

- Standards: GoLand reported one weak HTTP-link warning on an intentionally
  rejected downgrade fixture. A fixture-scoped `HttpUrlsUsage` suppression explains
  that the fixture uses a transport without network access. The TLS test helper
  now verifies its local server certificate and closes cloned idle connections.
- Spec: initial-URL rejection tests accepted any network error as success, so
  they did not prove rejection before transport. The single redirect fixture
  combined an HTTP downgrade with an untrusted host; it did not independently
  guard HTTPS host rejection, subsequent hops, or the final-response check.
  Production policy inspection found no confirmed runtime defect.
- Fix: transport spies return valid images if reached, assert zero requests for
  rejected initial URLs and no requests to rejected redirect targets, cover
  each rule at the first/subsequent hop, and check allowed cross-provider
  redirects, all five redirect statuses, the ten-request bound and final URLs.

Acceptance: `go test -tags no_emoji,nodynamic ./internal/ui/help -run
'TestLoadReleaseImageRejectsUntrusted|TestReleaseImageClient' -count=1` passes;
temporarily removing each relevant guard must fail its corresponding test.
Focused lifecycle/race verification uses the Help package and root release-note
integration tests. GitHub CI owns the complete suite for the review loop.

Delegation: one read-only CI/artifact scout, no implementation/review delegation.
G1 bounded evidence question; G2 downloaded SARIF and `gh` output; G3 zero repo
files; G4 separate remote-artifact context; G5 no duplicated Lead context. Budget:
1/1 scout; fixes inline; full local suite: no (review-loop override).

Verification evidence:

- All four negative checks failed for the intended reason when independently
  weakening initial admission, redirect admission, final-response admission and
  the redirect-count bound; the source was restored after each check.
- Focused release-note race tests pass: Help `1.912s`, root UI `2.006s`.
  Complete Help race tests pass after restoring the guards (`39.180s`).
- GoLand inspected all four Go files changed by the PR, including weak warnings;
  the final test file and restored production helper both report zero problems.
- `make fmt`, `make verify-build` and `git diff --check` pass. The build gate
  includes formatting, TUF freshness, generated assets/notices, exact Qodana
  exclusions, repository-wide vet and build. No root UI tests were added, so
  shard assignments are unchanged.
- Initial-head CI `35318650630` and CodeQL `35318650641` pass. Qodana
  `35318650657` has zero post-suppression SARIF results, independently read by
  the Lead. No existing review threads required disposition.
- Final commit-bound hosted code/security review, CI, Qodana SARIF and CodeQL
  evidence is recorded in the [PR #41 discussion](https://github.com/frathe/picfetch/pull/41).
  The complete race suite is delegated to native GitHub CI under the explicitly
  invoked review-loop workflow, rather than duplicated locally.
