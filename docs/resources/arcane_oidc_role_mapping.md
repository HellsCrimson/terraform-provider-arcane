# arcane_oidc_role_mapping

Maps an OIDC group/claim value to a role. On each OIDC login, a user's group claim is matched against `claim_value` and matching rows become role assignments.

Manages only manual mappings; mappings declared via the `OIDC_ROLE_MAPPINGS` env var are read-only.

## Example Usage

```hcl
resource "arcane_oidc_role_mapping" "platform_admins" {
  claim_value = "platform-admins"
  role_id     = arcane_role.deployer.id
}

# Optionally scope the assignment to a single environment.
resource "arcane_oidc_role_mapping" "staging_viewers" {
  claim_value    = "staging-viewers"
  role_id        = arcane_role.viewer.id
  environment_id = var.environment_id
}
```

## Argument Reference

### Required

- `claim_value` (String) - OIDC claim value to match (e.g. a group name).
- `role_id` (String) - ID of the role to assign when the claim matches.

### Optional

- `environment_id` (String) - Environment ID to scope the assignment to; omit for a global assignment.

## Attributes Reference

- `id` (String) - Unique identifier of the mapping.
- `source` (String) - How the mapping was created (`manual` or `env`).
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the mapping ID:

```
terraform import arcane_oidc_role_mapping.platform_admins <mapping_id>
```
