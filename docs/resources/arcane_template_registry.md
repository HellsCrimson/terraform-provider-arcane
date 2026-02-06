# arcane_template_registry

Manages an external template registry for accessing remote templates.

## Example Usage

```hcl
resource "arcane_template_registry" "community" {
  name        = "community-templates"
  url         = "https://templates.example.com/registry.json"
  description = "Community maintained templates"
  enabled     = true
}
```

## Argument Reference

### Required

- `name` (String) - Name of the template registry.
- `url` (String) - URL of the template registry.
- `description` (String) - Description of the template registry.
- `enabled` (Boolean) - Whether the registry is enabled.

## Attributes Reference

- `id` (String) - Unique identifier of the template registry.

## Import

Import using the template registry ID:

```
terraform import arcane_template_registry.community <registry_id>
```
