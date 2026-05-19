# arcane_container_registry_pull_usage

Reads configured container registry pull usage and rate limit visibility.

## Example Usage

```hcl
data "arcane_container_registry_pull_usage" "current" {}
```

## Attributes Reference

- `id` (String) - Static data source ID.
- `total_count` (Number) - Number of registry usage entries.
- `registries_json` (String) - Registry pull usage entries as JSON.
