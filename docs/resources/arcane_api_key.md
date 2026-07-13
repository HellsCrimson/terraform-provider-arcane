# arcane_api_key

Manages an API key for programmatic access to Arcane.

## Example Usage

```hcl
resource "arcane_api_key" "ci" {
  name        = "ci-pipeline"
  description = "API key for CI/CD pipeline"
  expires_at  = "2026-12-31T23:59:59Z"

  # A global grant (no environment_id) plus one scoped to a single environment.
  permissions = [
    { permission = "containers:list" },
    { permission = "containers:manage", environment_id = var.environment_id },
  ]
}

output "api_key" {
  value     = arcane_api_key.ci.key
  sensitive = true
}
```

## Argument Reference

### Required

- `name` (String) - Name of the API key (1-255 characters).
- `permissions` (Set of Object) - Permission grants for the key (at least one). Cannot exceed the creating user's own permissions. Each object supports:
  - `permission` (String, Required) - Permission string, e.g. `containers:list`.
  - `environment_id` (String, Optional) - Environment ID to scope the grant to; omit for a global grant.

### Optional

- `description` (String) - Optional description of the API key (max 1000 characters).
- `expires_at` (String) - Optional expiration date for the API key (RFC3339 format, e.g., '2025-12-31T23:59:59Z').

## Attributes Reference

- `id` (String) - Unique identifier of the API key.
- `key` (String, Sensitive) - The full API key secret. Only available on creation - cannot be retrieved later.
- `key_prefix` (String) - Prefix of the API key for identification.
- `is_bootstrap` (Bool) - Whether the API key is an auto-generated environment bootstrap key.
- `is_static` (Bool) - Whether the API key is environment-managed and protected from deletion.
- `user_id` (String) - ID of the user who owns the API key.
- `last_used_at` (String) - Last time the API key was used.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the API key ID:

```
terraform import arcane_api_key.ci <api_key_id>
```

Note: The `key` attribute cannot be retrieved after import since it is only returned on creation.
