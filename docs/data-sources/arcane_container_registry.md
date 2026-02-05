# arcane_container_registry

Reads an Arcane container registry configuration.

## Example Usage

```hcl
data "arcane_container_registry" "docker_hub" {
  id = "registry-123456"
}

output "registry_url" {
  value = data.arcane_container_registry.docker_hub.url
}
```

## Argument Reference

- `id` (String, Required) — registry ID.

## Attributes Reference

- `url` (String) — registry URL.
- `username` (String) — registry username.
- `description` (String) — registry description.
- `insecure` (Bool) — whether the registry uses insecure connections.
- `enabled` (Bool) — whether the registry is enabled.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** The registry token/password is never exposed in data sources for security reasons.
