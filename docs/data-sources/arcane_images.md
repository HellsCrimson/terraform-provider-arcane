# arcane_images

Lists Docker images in an environment.

## Example Usage

```hcl
data "arcane_images" "all" {
  environment_id = "env-123456"
}

output "image_count" {
  value = data.arcane_images.all.count
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `count` (Number) - number of images returned.
- `data_json` (String) - full image list payload as JSON.
