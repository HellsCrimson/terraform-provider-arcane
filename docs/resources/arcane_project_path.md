# arcane_project_path

Manages a compose project sourced from local files.

## Example Usage

```hcl
resource "arcane_project_path" "demo" {
  environment_id    = var.environment_id
  name              = "demo"
  compose_path      = "${path.module}/demo-compose.yml"
  # env_path        = "${path.module}/demo.env"
  # Store only hashes in state to detect changes
  content_hash_mode = true
}
```

## Argument Reference

- `environment_id` (String, Required)
- `name` (String, Required)
- `compose_path` (String, Required)
- `env_path` (String, Optional)
- `content_hash_mode` (Bool, Optional) — keeps only SHA256 hashes in state.
- `running` (Bool, Optional) — when true, ensures the project is running (compose up); when false, brings it down. If unset, lifecycle is not managed.
- `pull_on_update` (Bool, Optional) — when true, pulls images before each redeploy (default false).
- `redeploy_trigger` (String, Optional) — when the project is redeployed: `never`, `default` (when the compose/env file content changed — the default), `update` (on any in-place update) or `always` (on every apply). A redeploy is skipped when `running = false`. `always` makes the resource report a change on every plan (`last_redeploy` becomes "known after apply"), which is what lets Terraform call the provider when nothing else changed. See [arcane_project](arcane_project.md#redeploy-trigger) for details.
- `stop_before_rename` (Bool, Optional) — when true, a rename stops the project, renames it and starts it again in the same apply (default false). Arcane only renames stopped projects; without this, the plan fails when a rename targets a running project. See [arcane_project](arcane_project.md#renaming-a-project) for details.
- `remove_files` (Bool, Optional) — remove files on destroy.
- `remove_volumes` (Bool, Optional) — remove volumes on destroy.

## Attributes Reference

- `compose_content`, `env_content` (Sensitive, Computed) — when hash mode disabled
- `compose_content_hash`, `env_content_hash` (Sensitive, Computed) — when hash mode enabled
- `id`, `path`, `status`, `service_count`, `running_count`, `created_at`, `updated_at`
- `last_redeploy` — RFC3339 timestamp of the last redeploy performed by the provider. Null until the provider has redeployed the project at least once.
