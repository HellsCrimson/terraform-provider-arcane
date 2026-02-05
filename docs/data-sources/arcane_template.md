# arcane_template

Reads an Arcane template configuration.

## Example Usage

```hcl
data "arcane_template" "wordpress" {
  id = "template-123456"
}

output "template_content" {
  value = data.arcane_template.wordpress.content
}
```

## Argument Reference

- `id` (String, Required) — template ID.

## Attributes Reference

- `name` (String) — template name.
- `description` (String) — template description.
- `content` (String) — Docker Compose YAML content.
- `env_content` (String) — environment variables template content.
- `is_custom` (Bool) — whether this is a custom template.
- `is_remote` (Bool) — whether this template is from a remote registry.
- `registry_id` (String) — registry ID if remote.
