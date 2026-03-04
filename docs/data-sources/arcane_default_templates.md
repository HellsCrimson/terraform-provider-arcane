# arcane_default_templates

Reads global default templates.

## Example Usage

```hcl
data "arcane_default_templates" "defaults" {}

output "compose_template" {
  value = data.arcane_default_templates.defaults.compose_template
}
```

## Argument Reference

This data source has no arguments.

## Attributes Reference

- `compose_template` (String) - default compose template.
- `env_template` (String) - default env template.
