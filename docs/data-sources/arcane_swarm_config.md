# arcane_swarm_config

Reads an Arcane Docker Swarm config. Only metadata is exposed; use the
`arcane_swarm_config` resource to manage config content.

## Example Usage

```hcl
data "arcane_swarm_config" "app_config" {
  environment_id = var.environment_id
  id             = "config-123"
}

output "swarm_config_version" {
  value = data.arcane_swarm_config.app_config.version_index
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — swarm config ID.

## Attributes Reference

- `name` (String) — config name.
- `labels` (Map of String) — config labels.
- `version_index` (Number) — Swarm object version index.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
