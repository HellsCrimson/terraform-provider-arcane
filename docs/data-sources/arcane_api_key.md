# arcane_api_key

Reads an Arcane API key metadata.

## Example Usage

```hcl
data "arcane_api_key" "terraform_key" {
  id = "key-123456"
}

output "key_prefix" {
  value = data.arcane_api_key.terraform_key.key_prefix
}
```

## Argument Reference

- `id` (String, Required) — API key ID.

## Attributes Reference

- `name` (String) — name of the API key.
- `description` (String) — description of the API key.
- `expires_at` (String) — expiration date.
- `key_prefix` (String) — key prefix for identification.
- `permissions` (Set of Object) — permission grants held by the key. Each object contains:
  - `permission` (String) — permission string, e.g. `containers:list`.
  - `environment_id` (String) — environment ID the grant is scoped to; empty for a global grant.
- `is_bootstrap` (Bool) — whether the API key is an auto-generated environment bootstrap key.
- `is_static` (Bool) — whether the API key is environment-managed and protected from deletion.
- `user_id` (String) — owner user ID.
- `last_used_at` (String) — last usage timestamp.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** The full API key secret is never retrievable after creation and is not exposed in data sources.
