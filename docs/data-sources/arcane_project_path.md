# arcane_project_path

Reads an Arcane project created from filesystem paths.

## Example Usage

```hcl
data "arcane_project_path" "webapp" {
  environment_id = "env-123456"
  id             = "project-789"
}

output "project_status" {
  value = data.arcane_project_path.webapp.status
}

output "compose_content" {
  value     = data.arcane_project_path.webapp.compose_content
  sensitive = true
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — project ID.

## Attributes Reference

- `name` (String) — project name.
- `compose_content` (String, Sensitive) — Docker Compose content.
- `env_content` (String, Sensitive) — environment variables content.
- `path` (String) — project path on the environment.
- `status` (String) — project status.
- `service_count` (Number) — number of services.
- `running_count` (Number) — number of running services.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
