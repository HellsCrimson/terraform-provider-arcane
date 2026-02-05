# arcane_gitops_sync

Reads an Arcane GitOps sync configuration.

## Example Usage

```hcl
data "arcane_gitops_sync" "app_sync" {
  environment_id = "env-123456"
  id             = "sync-789"
}

output "last_sync" {
  value = data.arcane_gitops_sync.app_sync.last_sync_at
}

output "project_id" {
  value = data.arcane_gitops_sync.app_sync.project_id
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — GitOps sync ID.

## Attributes Reference

- `name` (String) — sync configuration name.
- `repository_id` (String) — git repository ID.
- `branch` (String) — git branch.
- `compose_path` (String) — path to docker-compose file.
- `project_name` (String) — project name.
- `auto_sync` (Bool) — auto sync enabled.
- `sync_interval` (Number) — sync interval in seconds.
- `enabled` (Bool) — whether sync is enabled.
- `environment_variables` (Map of String) — environment variables from the associated project.
- `project_id` (String) — associated project ID.
- `last_sync_at` (String) — last sync timestamp.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
