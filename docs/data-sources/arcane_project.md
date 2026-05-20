# arcane_project

Reads an Arcane project configuration.

## Example Usage

```hcl
data "arcane_project" "webapp" {
  environment_id = "env-123456"
  id             = "project-789"
}

output "project_status" {
  value = data.arcane_project.webapp.status
}

output "running_services" {
  value = data.arcane_project.webapp.running_count
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — project ID.

## Attributes Reference

- `name` (String) — project name.
- `compose_content` (String) — Docker Compose content.
- `env_content` (String) — environment variables content.
- `path` (String) — project path on the environment.
- `status` (String) — project status.
- `service_count` (Number) — number of services.
- `running_count` (Number) — number of running services.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
- `archived` (Bool) — whether the project is archived.
- `archived_at` (String) — archive timestamp.
- `is_discovered` (Bool) — whether the project was discovered.
- `redeploy_disabled` (Bool) — whether redeploy/update actions are disabled for this project.
