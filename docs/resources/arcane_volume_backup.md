# arcane_volume_backup

Creates and manages a backup snapshot for a Docker volume.

## Example Usage

```hcl
resource "arcane_volume_backup" "db_snapshot" {
  environment_id = var.environment_id
  volume_name    = "postgres-data"
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `volume_name` (String) - Volume name to back up. Changing this forces a new resource.

## Attributes Reference

- `id` (String) - Backup ID.
- `size` (Number) - Backup size in bytes.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp, when available.

## Import

Import using the format `environment_id/volume_name/backup_id`:

```
terraform import arcane_volume_backup.db_snapshot <environment_id>/<volume_name>/<backup_id>
```
