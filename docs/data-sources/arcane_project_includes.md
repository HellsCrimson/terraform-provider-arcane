# arcane_project_includes

Reads include files from a project.

## Example Usage

```hcl
data "arcane_project_includes" "project_files" {
  environment_id = "env-123456"
  project_id     = "project-789"
}

output "include_count" {
  value = data.arcane_project_includes.project_files.count
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.
- `project_id` (String, Required) - project ID.

## Attributes Reference

- `count` (Number) - number of include files.
- `includes_json` (String) - full include file payload as JSON.
