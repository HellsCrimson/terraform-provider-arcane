# arcane_user

Reads an Arcane user configuration.

## Example Usage

```hcl
data "arcane_user" "admin" {
  id = "user-123456"
}

output "user_email" {
  value = data.arcane_user.admin.email
}
```

## Argument Reference

- `id` (String, Required) — user ID.

## Attributes Reference

- `username` (String) — username.
- `display_name` (String) — display name.
- `email` (String) — email address.
- `locale` (String) — locale preference.
- `role_assignments` (Set of Object) — role assignments held by the user. Each object contains:
  - `role_id` (String) — ID of the granted role.
  - `environment_id` (String) — environment ID the assignment is scoped to; empty for a global assignment.
  - `source` (String) — how the assignment was created (`manual` or `oidc`).
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** The user's password is never exposed in data sources for security reasons.
