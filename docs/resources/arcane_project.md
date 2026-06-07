# arcane_project

Manages a compose project with inline content.

## Example Usage

```hcl
resource "arcane_project" "demo" {
  environment_id  = var.environment_id
  name            = "demo"
  compose_content = file("${path.module}/docker-compose.yml")
  # env_content  = file("${path.module}/.env")
}
```

## Argument Reference

- `environment_id` (String, Required)
- `name` (String, Required)
- `compose_content` (String, Required)
- `env_content` (String, Optional)
- `archived` (Bool, Optional) — when true, brings the project down and archives it; when false, unarchives it.
- `pull_on_update` (Bool, Optional) — when true, pulls images before redeploy when `compose_content`/`env_content` change (default false).
- `running` (Bool, Optional) — when true, ensures the project is running (compose up); when false, brings it down. If unset, lifecycle is not managed.
- `fail_if_name_exists` (Bool, Optional) — when true, the plan fails if a project with the same `name` already exists in the environment (including folders Arcane has discovered on disk), instead of letting Arcane auto-rename the new project with a numeric suffix (default false). The check runs during the plan phase, so the collision is reported before any change is applied.

## Attributes Reference

- `id`, `path`, `status`, `service_count`, `running_count`, `created_at`, `updated_at`
- `archived_at`, `is_discovered`, `redeploy_disabled`
