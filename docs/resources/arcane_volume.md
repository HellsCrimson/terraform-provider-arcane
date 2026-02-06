# arcane_volume

Manages a Docker volume for persistent storage.

## Example Usage

```hcl
resource "arcane_volume" "data" {
  environment_id = var.environment_id
  name           = "app-data"
  driver         = "local"
  labels = {
    "app"         = "myapp"
    "environment" = "production"
  }
}

# Volume with NFS driver
resource "arcane_volume" "nfs_share" {
  environment_id = var.environment_id
  name           = "nfs-data"
  driver         = "local"
  driver_opts = {
    type   = "nfs"
    o      = "addr=192.168.1.100,rw"
    device = ":/exports/data"
  }
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `name` (String) - Name of the volume. Changing this forces a new resource.

### Optional

- `driver` (String) - Volume driver (e.g., local, nfs). Defaults to 'local'. Changing this forces a new resource.
- `driver_opts` (Map of String) - Driver-specific options.
- `labels` (Map of String) - User-defined labels for metadata.

## Attributes Reference

- `id` (String) - Unique identifier of the volume.
- `mountpoint` (String) - Mount point of the volume on the host.
- `scope` (String) - Scope of the volume (local or global).
- `created_at` (String) - Creation timestamp.
- `in_use` (Boolean) - Whether the volume is currently in use.
- `size` (Number) - Size of the volume in bytes.
- `containers` (List of String) - List of containers using this volume.

## Import

Import using the format `environment_id/volume_name`:

```
terraform import arcane_volume.data <environment_id>/<volume_name>
```

## Notes

Volumes cannot be updated in place. Any changes to the volume configuration will force the creation of a new volume.
