# arcane_swarm_secret

Reads an Arcane Docker Swarm secret. The secret value is never returned by the API; only metadata is available.

## Example Usage

```hcl
data "arcane_swarm_secret" "db_password" {
  environment_id = var.environment_id
  id             = "secret-789"
}

output "swarm_secret_version" {
  value = data.arcane_swarm_secret.db_password.version_index
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — swarm secret ID.

## Attributes Reference

- `name` (String) — secret name.
- `labels` (Map of String) — secret labels.
- `version_index` (Number) — Swarm object version index.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
