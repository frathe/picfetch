# 01: Configure Store access and prove read-only connectivity

**What to build:** A maintainer can verify from CI that the automation identity can read PicFetch's existing Store product, with credentials available only after required environment approval. Complete account setup independently of publisher implementation; this ticket makes no Store submission.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human; environment setup verified, approved CI read check pending

**Parent:** [Approved Microsoft Store updates](../spec.md).

**Coverage:** User stories US19, US20, US23; AC7 (live access prerequisite).

## Acceptance criteria

- [ ] Inspect existing Partner Center/Entra configuration before provisioning anything. A supported authenticated read returns product 9P0DM0KTH01K and its current published/pending submission identifiers. Record the observed state, which may have advanced beyond the reported 1.0.2 baseline.
  Verification: Using the implemented read-only command without exposing credentials: `go run ./scripts/storepublish check`.

- [x] Configure a dedicated `microsoft-store` GitHub environment and suitable automation identity, or reuse compatible existing configuration. Record authentication method, exact credential variable/secret names, permitted repository/ref access, and expiry/renewal information. Store credential values only in protected storage.
  Verification: `gh secret list --repo frathe/picfetch --env microsoft-store` plus the successful authenticated product read above; retain names and results, never secret values.

- [x] Inspect actual environment protection rules and deployment-ref policy. Each rollout must wait for the configured REDACTED_REVIEWER reviewer; self-review is allowed, and bypass, timer and custom rules are disabled. Preserve restrictions against untrusted refs; GitHub Release publication must not be required.
  Verification: `gh api repos/frathe/picfetch/environments/microsoft-store --jq '{name, protection_rules, deployment_branch_policy}'`; assess and record the returned rules, including any custom protection behavior.

- [ ] Run the product read from a GitHub Actions diagnostic context using the intended environment. Retain a redacted run summary proving CI access. Document only genuinely unavailable account-administration steps; an unresolved permission/configuration step keeps this ticket open while code work continues.
  Verification: `gh run view <diagnostic-run-id> --repo frathe/picfetch --json conclusion,jobs,url` plus the run's redacted product-read result. A local read alone does not satisfy CI connectivity.

## Implementation notes

This is an operational slice, not a prerequisite for fake-service development. Use an existing supported read-only client while the publisher command is being built. Discovery must establish the necessary Microsoft permissions; do not assume OIDC support. Enabling automatic publishing belongs to ticket 06. Record account facts and renewal instructions in the Store operations documentation. Required reviewers must gate every rollout.

The final user amendment requires REDACTED_REVIEWER approval of each specific validated release.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands define the existing command/fake-service acceptance seam.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.

2026-09-08: GitHub secret-name and environment reads confirmed no Store
credentials/environment exist. The exact environment configuration and main-only
branch rule are prepared in `../environment.json` and `../main-branch.json`.
Automatic approval review rejected the remote environment creation because it
changes deployment security controls without approval of that exact scope; no
remote environment was created. Entra/Partner Center account access is also
unavailable. `../configure-access.sh` (bash syntax checked) guides the user-only
account steps, checks access, and installs environment secrets after the approved
environment exists. No live product read or CI connectivity evidence exists yet.

2026-09-08 approval amendment (supersedes earlier unattended/setup comments):
The final user requirement is approval of every specific release via GitHub Required
reviewers. GitHub metadata now verifies main-only, REDACTED_REVIEWER reviewer, self-review
allowed, no bypass/timer/custom rules, and all three secret names. The user confirmed
the linked Developer application and replacement key. Credentials remain exclusively
in GitHub Secrets. The publisher is not yet on main, so the approved CI read check,
actual submission permission and first ordinary approved publication remain open.
Preparation freezes artifact/notes/base/receipt before the protected job; submit
uses only that selection and reconcile never admits a newer release. Cron is removed;
status/recovery require manual approved dispatch. Local approval and workflow tests
cover the amendment; see the plan for verification. No Store update was submitted.
