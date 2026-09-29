# arcane_user

Manages an Arcane user.

## Example Usage

```hcl
resource "arcane_user" "example" {
  username            = "johndoe"
  password_wo         = var.user_password
  password_wo_version = 1
  display_name        = "John Doe"
  email               = "john@example.com"

  # A global role, plus one scoped to a single environment.
  role_assignments = [
    { role_id = arcane_role.viewer.id },
    { role_id = arcane_role.deployer.id, environment_id = var.environment_id },
  ]
}
```

## Argument Reference

- `username` (String, Required, ForceNew)
- `password` (String, Optional, Sensitive) — at least 8 characters. Exactly one of `password` or `password_wo` is required. **Deprecated**: stored in state; use `password_wo` instead, `password` will be removed in the next major release.
- `password_wo` (String, Optional, Sensitive, Write-only) — at least 8 characters. Never stored in state or plan files; only sent when `password_wo_version` changes. Conflicts with `password`. See [write-only arguments](../index.md#write-only-arguments).
- `password_wo_version` (Number, Optional) — change it to send a new `password_wo`. Required with `password_wo`.
- `display_name` (String, Optional)
- `email` (String, Optional)
- `locale` (String, Optional) — locale preference (e.g. `en-US`).
- `role_assignments` (Set of Object, Optional) — manual role assignments for the user. Manages only manual assignments; assignments created via OIDC are left untouched. Each object supports:
  - `role_id` (String, Required) — ID of the role to grant.
  - `environment_id` (String, Optional) — environment ID to scope the assignment to; omit for a global assignment.

## Attributes Reference

- `id` (String)
- `created_at` (String)
- `updated_at` (String)
