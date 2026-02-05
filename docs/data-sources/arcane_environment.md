# arcane_environment

Reads an Arcane environment configuration.

## Example Usage

```hcl
data "arcane_environment" "prod" {
  id = "env-123456"
}

output "environment_status" {
  value = data.arcane_environment.prod.status
}
```

## Argument Reference

- `id` (String, Required) — environment ID.

## Attributes Reference

- `name` (String) — environment display name.
- `api_url` (String) — agent API URL.
- `status` (String) — environment status.
- `enabled` (Bool) — whether the environment is enabled.
- `api_key` (String, Sensitive) — environment API key (if available).
