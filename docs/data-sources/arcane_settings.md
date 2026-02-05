# arcane_settings

Reads all Arcane environment settings as a key-value map.

## Example Usage

```hcl
data "arcane_settings" "prod_settings" {
  environment_id = "env-123456"
}

output "all_settings" {
  value = data.arcane_settings.prod_settings.settings
}

output "docker_host" {
  value = lookup(data.arcane_settings.prod_settings.settings, "dockerHost", "")
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.

## Attributes Reference

- `settings` (Map of String) — all environment settings as a key-value map. This includes all configuration keys like `dockerHost`, `projectsDirectory`, `defaultShell`, etc.

**Note:** This data source returns all settings as a flat map of strings, making it easy to reference individual settings using the `lookup()` function.
