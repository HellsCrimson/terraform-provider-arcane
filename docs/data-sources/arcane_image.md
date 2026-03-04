# arcane_image

Reads details for a specific Docker image.

## Example Usage

```hcl
data "arcane_image" "nginx" {
  environment_id = "env-123456"
  id             = "sha256:..."
}

output "image_size" {
  value = data.arcane_image.nginx.size
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.
- `id` (String, Required) - image ID.

## Attributes Reference

- `created` (String) - creation timestamp.
- `size` (Number) - image size in bytes.
- `author` (String) - image author.
- `architecture` (String) - target architecture.
- `os` (String) - target operating system.
- `data_json` (String) - full image details payload as JSON.
