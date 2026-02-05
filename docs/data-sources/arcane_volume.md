# arcane_volume

Reads an Arcane Docker volume configuration.

## Example Usage

```hcl
data "arcane_volume" "postgres_data" {
  environment_id = "env-123456"
  id             = "volume-789"
}

output "volume_mountpoint" {
  value = data.arcane_volume.postgres_data.mountpoint
}

output "volume_driver" {
  value = data.arcane_volume.postgres_data.driver
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — volume ID.

## Attributes Reference

- `name` (String) — volume name.
- `driver` (String) — volume driver.
- `driver_opts` (Map of String) — driver-specific options.
- `labels` (Map of String) — volume labels.
- `mountpoint` (String) — mount point on host.
- `scope` (String) — volume scope.
- `created_at` (String) — creation timestamp.
