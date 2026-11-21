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

### Optional

- `destination` (String) - Where the backup is stored: `local` (server default) or `s3` (needs `s3_destination_id`). Changing this forces a new resource.
- `policy_id` (String) - Backup policy whose settings are used for this backup. Changing this forces a new resource.
- `s3_destination_id` (String) - Saved S3 destination for a remote backup. Changing this forces a new resource.

## Attributes Reference

- `id` (String) - Backup ID.
- `status` (String) - Backup run status (`running`, `succeeded`, `failed`). The backup runs in the background, so it is `running` right after create.
- `error` (String) - Error of a failed backup run.
- `format` (String) - Storage format (rustic snapshot or legacy tar.gz).
- `trigger` (String) - How the backup was started.
- `s3_destination_name` (String) - Name of the S3 destination used.
- `remote_available` (Boolean) - Whether the backup is available in the S3 destination.
- `size` (Number) - Backup size in bytes.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp, when available.

## Import

Import using the format `environment_id/volume_name/backup_id`:

```
terraform import arcane_volume_backup.db_snapshot <environment_id>/<volume_name>/<backup_id>
```
