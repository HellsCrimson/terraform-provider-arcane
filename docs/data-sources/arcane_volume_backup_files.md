# arcane_volume_backup_files

Lists files contained in a volume backup.

## Example Usage

```hcl
data "arcane_volume_backup_files" "backup_files" {
  environment_id = "env-123456"
  backup_id      = "backup-789"
}

output "files" {
  value = data.arcane_volume_backup_files.backup_files.files
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.
- `backup_id` (String, Required) - backup ID.

## Attributes Reference

- `files` (List of String) - file paths found in the backup.
