# arcane_ignored_vulnerabilities

Lists ignored vulnerabilities for an environment.

## Example Usage

```hcl
data "arcane_ignored_vulnerabilities" "ignored" {
  environment_id = "env-123456"
}

output "ignored_count" {
  value = data.arcane_ignored_vulnerabilities.ignored.total_count
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `total_count` (Number) - number of ignored vulnerability records.
- `data_json` (String) - full ignored vulnerability payload as JSON.
