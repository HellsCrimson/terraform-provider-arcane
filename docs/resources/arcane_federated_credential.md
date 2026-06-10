# arcane_federated_credential

Manages a workload identity federation trust rule. Allows an external OIDC issuer to exchange tokens for an Arcane service token bound to a role.

## Example Usage

```hcl
resource "arcane_federated_credential" "github_actions" {
  name          = "github-actions-deploy"
  enabled       = true
  issuer_url    = "https://token.actions.githubusercontent.com"
  audiences     = ["https://arcane.example.com"]
  subject_match = "repo:my-org/my-repo:ref:refs/heads/main"
  role_id       = arcane_role.deployer.id

  match_type        = "exact"
  subject_claim     = "sub"
  token_ttl_seconds = 900
}
```

## Argument Reference

### Required

- `name` (String) - Display name.
- `enabled` (Boolean) - Whether token exchanges are allowed.
- `issuer_url` (String) - Trusted external OIDC issuer URL.
- `audiences` (Set of String) - Allowed external token audiences (at least one).
- `subject_match` (String) - Exact subject or anchored glob pattern to match.
- `role_id` (String) - Mapped role ID granted to exchanged tokens.

### Optional

- `description` (String) - Optional description.
- `environment_id` (String) - Optional environment scope for the role assignment.
- `expires_at` (String) - Optional credential expiration (RFC3339).
- `match_type` (String) - Subject match strategy: `exact` or `glob`. Computed when not set.
- `subject_claim` (String) - Claim path to match against; defaults to `sub`. Computed when not set.
- `token_ttl_seconds` (Number) - Issued token lifetime in seconds (60-3600). Computed when not set.

## Attributes Reference

- `id` (String) - Unique identifier of the federated credential.
- `role_name` (String) - Mapped role name.
- `environment_name` (String) - Mapped environment name when scoped.
- `identity_user_id` (String) - Dedicated service user ID backing issued tokens.
- `service_username` (String) - Dedicated service account username.
- `last_used_at` (String) - Last successful token exchange.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the federated credential ID:

```
terraform import arcane_federated_credential.github_actions <credential_id>
```
