# Arcane Provider Data Sources

This directory contains documentation for all data sources provided by the Arcane Terraform provider. Data sources allow you to query existing infrastructure configurations for use in your Terraform configurations.

## Available Data Sources

### Simple ID Data Sources
These data sources require only an `id` parameter:

- [arcane_environment](./arcane_environment.md) - Read environment configurations
- [arcane_user](./arcane_user.md) - Read user details
- [arcane_container_registry](./arcane_container_registry.md) - Read container registry configurations
- [arcane_git_repository](./arcane_git_repository.md) - Read git repository configurations
- [arcane_api_key](./arcane_api_key.md) - Read API key metadata
- [arcane_template](./arcane_template.md) - Read template configurations
- [arcane_template_registry](./arcane_template_registry.md) - Read template registry configurations
- [arcane_notification](./arcane_notification.md) - Read notification provider configurations (requires `environment_id` and `provider_name`)

### Composite ID Data Sources
These data sources require both `environment_id` and `id` parameters:

- [arcane_project](./arcane_project.md) - Read project details
- [arcane_container](./arcane_container.md) - Read container details
- [arcane_gitops_sync](./arcane_gitops_sync.md) - Read GitOps sync configurations
- [arcane_network](./arcane_network.md) - Read Docker network configurations
- [arcane_volume](./arcane_volume.md) - Read Docker volume configurations
- [arcane_project_path](./arcane_project_path.md) - Read project path configurations
- [arcane_job_schedules](./arcane_job_schedules.md) - Read job schedules for an environment

### Special Data Sources

- [arcane_settings](./arcane_settings.md) - Read all environment settings as a key-value map

## Usage Example

Data sources are typically used to reference existing infrastructure:

```hcl
# Query an existing environment
data "arcane_environment" "prod" {
  id = "env-123456"
}

# Query a project in that environment
data "arcane_project" "webapp" {
  environment_id = data.arcane_environment.prod.id
  id             = "project-789"
}

# Use the data in outputs or other resources
output "webapp_status" {
  value = data.arcane_project.webapp.status
}

output "running_services" {
  value = "${data.arcane_project.webapp.running_count}/${data.arcane_project.webapp.service_count}"
}
```

## Security Notes

For security reasons, certain sensitive fields are never exposed in data sources:

- User passwords
- Container registry tokens/passwords
- Git repository SSH keys and access tokens
- The full API key secret (only the key prefix is available)

This ensures that sensitive credentials cannot be accidentally exposed through Terraform state or outputs.
