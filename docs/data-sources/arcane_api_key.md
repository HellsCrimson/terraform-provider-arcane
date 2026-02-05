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
- `user_id` (String) — owner user ID.
- `last_used_at` (String) — last usage timestamp.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** The full API key secret is never retrievable after creation and is not exposed in data sources.
