# arcane_template_registry

Reads an Arcane template registry configuration.

## Example Usage

```hcl
data "arcane_template_registry" "official" {
  id = "registry-123456"
}

output "registry_url" {
  value = data.arcane_template_registry.official.url
}
```

## Argument Reference

- `id` (String, Required) — template registry ID.

## Attributes Reference

- `name` (String) — registry name.
- `url` (String) — registry URL.
- `description` (String) — registry description.
- `enabled` (Bool) — whether the registry is enabled.
