# 01: Configure Store access and prove read-only connectivity

**What to build:** A maintainer can verify from CI that the automation identity can read PicFetch's existing Store product, with credentials available for unattended updates. Complete account setup independently of publisher implementation; this ticket makes no Store submission.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

**Parent:** [Automatic Microsoft Store updates](../spec.md).

**Coverage:** User stories US19, US20, US23; AC7 (live access prerequisite).

## Acceptance criteria

- [ ] Inspect existing Partner Center/Entra configuration before provisioning anything. A supported authenticated read returns product 9P0DM0KTH01K and its current published/pending submission identifiers. Record the observed state, which may have advanced beyond the reported 1.0.2 baseline.
  Verification: After configuring the official CLI without exposing credentials: `msstore apps get 9P0DM0KTH01K`.

- [ ] Configure a dedicated `microsoft-store` GitHub environment and suitable automation identity, or reuse compatible existing configuration. Record authentication method, exact credential variable/secret names, permitted repository/ref access, and expiry/renewal information. Store credential values only in protected storage.
  Verification: `gh secret list --repo frathe/picfetch --env microsoft-store` plus the successful authenticated product read above; retain names and results, never secret values.

- [ ] Inspect actual environment protection rules and deployment-ref policy. Routine eligible updates must not wait for required reviewers or another approval rule. Preserve restrictions against untrusted refs; GitHub Release publication must not be required.
  Verification: `gh api repos/frathe/picfetch/environments/microsoft-store --jq '{name, protection_rules, deployment_branch_policy}'`; assess and record the returned rules, including any custom protection behavior.

- [ ] Run the product read from a GitHub Actions diagnostic context using the intended environment. Retain a redacted run summary proving CI access. Document only genuinely unavailable account-administration steps; an unresolved permission/configuration step keeps this ticket open while code work continues.
  Verification: `gh run view <diagnostic-run-id> --repo frathe/picfetch --json conclusion,jobs,url` plus the run's redacted product-read result. A local read alone does not satisfy CI connectivity.

## Implementation notes

This is an operational slice, not a prerequisite for fake-service development. Use an existing supported read-only client while the publisher command is being built. Discovery must establish the necessary Microsoft permissions; do not assume OIDC support. Enabling automatic publishing belongs to ticket 06. Record account facts and renewal instructions in the Store operations documentation. No human release-approval gate is added.

The spec's proposed defaults remain defaults, not newly confirmed user decisions.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands referring to publisher tests or commands describe future
implementation requirements; they have not been executed as part of ticket creation.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.
