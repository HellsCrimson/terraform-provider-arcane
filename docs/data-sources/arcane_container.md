# arcane_container

Reads an Arcane container configuration.

## Example Usage

```hcl
data "arcane_container" "nginx" {
  environment_id = "env-123456"
  id             = "container-789"
}

output "container_status" {
  value = data.arcane_container.nginx.status
}

output "container_image" {
  value = data.arcane_container.nginx.image
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — container ID.

## Attributes Reference

- `name` (String) — container name.
- `image` (String) — container image.
- `created` (String) — creation timestamp.
- `status` (String) — container status.
