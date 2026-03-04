# arcane_volume_backup_has_path

Checks whether a specific path exists inside a volume backup.

## Example Usage

```hcl
data "arcane_volume_backup_has_path" "check" {
  environment_id = "env-123456"
  backup_id      = "backup-789"
  path           = "/var/lib/postgresql/data/PG_VERSION"
}

output "exists" {
  value = data.arcane_volume_backup_has_path.check.exists
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.
- `backup_id` (String, Required) - backup ID.
- `path` (String, Required) - path to check in the backup.

## Attributes Reference

- `exists` (Bool) - whether the path exists.
