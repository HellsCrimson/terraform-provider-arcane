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
- `pre_deploy_script_path` (String) — path inside the synced repository to the pre-deploy hook script.
- `pre_deploy_runner_image` (String) — container image used to run the pre-deploy script.
- `pre_deploy_env` (String, Sensitive) — environment variables exposed to the pre-deploy script (`.env` file format).
- `pre_deploy_extra_mounts` (String, Sensitive) — extra bind mounts for the pre-deploy runner container, one per line.
- `pre_deploy_timeout_sec` (Number) — timeout in seconds for the pre-deploy script.
- `pre_deploy_network_mode` (String) — Docker network mode for the pre-deploy runner container.
- `environment_variables` (Map of String) — environment variables from the associated project.
- `project_id` (String) — associated project ID.
- `last_sync_at` (String) — last sync timestamp.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
