# arcane_project_files

Reads the workspace (directory tree) of a project.

## Example Usage

```hcl
data "arcane_project_files" "project" {
  environment_id = var.environment_id
  project_id     = var.project_id
}
```

## Argument Reference

- `environment_id` (String, Required) - Environment ID.
- `project_id` (String, Required) - Project ID.

## Attributes Reference

- `id` (String) - Data source ID.
- `details_json` (String) - Raw workspace response data as JSON.
