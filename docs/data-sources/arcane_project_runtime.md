# arcane_project_runtime

Reads runtime service state for a project.

## Example Usage

```hcl
data "arcane_project_runtime" "project" {
  environment_id = var.environment_id
  project_id     = var.project_id
}
```

## Argument Reference

- `environment_id` (String, Required) - Environment ID.
- `project_id` (String, Required) - Project ID.

## Attributes Reference

- `id` (String) - Data source ID.
- `details_json` (String) - Raw ProjectDetails response data as JSON.
