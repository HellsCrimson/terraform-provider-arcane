# arcane_project_path

Manages a compose project sourced from local files.

## Example Usage

```
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

## Attributes Reference

- `compose_content`, `env_content` (Sensitive, Computed) — when hash mode disabled
- `compose_content_hash`, `env_content_hash` (Sensitive, Computed) — when hash mode enabled
- `id`, `path`, `status`, `service_count`, `running_count`, `created_at`, `updated_at`
