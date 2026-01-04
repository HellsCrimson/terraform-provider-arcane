# arcane_project

Manages a compose project with inline content.

## Example Usage

```
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
- `running` (Bool, Optional) — when true, ensures the project is running (compose up); when false, brings it down. If unset, lifecycle is not managed.

## Attributes Reference

- `id`, `path`, `status`, `service_count`, `running_count`, `created_at`, `updated_at`
