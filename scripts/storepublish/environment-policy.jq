# Inputs: environment, deployment branch policies, custom protection rules.
length == 3 and
(.[0] |
  .name == "microsoft-store" and
  .can_admins_bypass == false and
  .deployment_branch_policy.custom_branch_policies == true and
  .deployment_branch_policy.protected_branches == false and
  (.protection_rules | length) == 2 and
  ([.protection_rules[] | select(.type == "branch_policy")] | length) == 1 and
  ([.protection_rules[] | select(.type == "required_reviewers")] |
    length == 1 and
    .[0].prevent_self_review == false and
    (.[0].reviewers | length) == 1 and
    .[0].reviewers[0].type == "User" and
    .[0].reviewers[0].reviewer.login == "frathe")) and
(.[1] | .total_count == 1 and (.branch_policies | length) == 1 and
  .branch_policies[0].name == "main" and .branch_policies[0].type == "branch") and
(.[2] | .total_count == 0 and (.custom_deployment_protection_rules | length) == 0)
