# arcane_public_settings

Reads public settings for an environment.

## Example Usage

```hcl
data "arcane_public_settings" "public" {
  environment_id = "env-123456"
}

output "public_settings" {
  value = data.arcane_public_settings.public.settings
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `settings` (Map of String) - public settings as a key/value map.
