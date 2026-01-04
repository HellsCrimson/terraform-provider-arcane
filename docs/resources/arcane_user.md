# arcane_user

Manages an Arcane user.

## Example Usage

```
resource "arcane_user" "example" {
  username     = "johndoe"
  password     = "SuperSecret123!"
  display_name = "John Doe"
  email        = "john@example.com"
  roles        = ["user"]
}
```

## Argument Reference

- `username` (String, Required, ForceNew)
- `password` (String, Required, Sensitive)
- `display_name` (String, Optional)
- `email` (String, Optional)
- `locale` (String, Optional)
- `roles` (Set(String), Optional)

## Attributes Reference

- `id` (String)
- `created_at` (String)
- `updated_at` (String)

