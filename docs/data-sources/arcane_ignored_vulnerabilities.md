# arcane_ignored_vulnerabilities

Lists ignored vulnerabilities for an environment.

## Example Usage

```hcl
data "arcane_ignored_vulnerabilities" "ignored" {
  environment_id = "env-123456"
}

output "ignored_count" {
  value = data.arcane_ignored_vulnerabilities.ignored.count
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `count` (Number) - number of ignored vulnerability records.
- `data_json` (String) - full ignored vulnerability payload as JSON.
