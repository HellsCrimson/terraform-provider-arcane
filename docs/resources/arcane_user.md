# arcane_user

Manages an Arcane user.

## Example Usage

```hcl
resource "arcane_user" "example" {
  username     = "johndoe"
  password     = "SuperSecret123!"
  display_name = "John Doe"
  email        = "john@example.com"

  # A global role, plus one scoped to a single environment.
  role_assignments = [
    { role_id = arcane_role.viewer.id },
    { role_id = arcane_role.deployer.id, environment_id = var.environment_id },
  ]
}
```

## Argument Reference

- `username` (String, Required, ForceNew)
- `password` (String, Required, Sensitive) — at least 8 characters.
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
