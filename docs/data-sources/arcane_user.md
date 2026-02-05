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
- `roles` (Set of String) — assigned roles.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** The user's password is never exposed in data sources for security reasons.
