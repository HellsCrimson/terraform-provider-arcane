# arcane_role

Looks up an Arcane RBAC role by ID or name. Useful for resolving built-in role IDs (e.g. for `arcane_user` role assignments) without hardcoding them.

Provide exactly one of `id` or `name`.

## Example Usage

```hcl
data "arcane_role" "admin" {
  name = "Admin"
}

output "admin_role_id" {
  value = data.arcane_role.admin.id
}
```

## Argument Reference

- `id` (String, Optional) — role ID. Provide either `id` or `name`.
- `name` (String, Optional) — role name. Provide either `id` or `name`.

## Attributes Reference

- `description` (String) — optional human description.
- `permissions` (Set of String) — permission strings granted by this role.
- `built_in` (Boolean) — whether this is a built-in role.
- `assigned_user_count` (Number) — how many users currently hold an assignment to this role.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
