# arcane_settings

Updates Arcane environment settings using explicit fields mapped from SettingsUpdate.

## Example Usage

```
resource "arcane_settings" "env" {
  environment_id          = var.environment_id
  base_server_url         = "http://localhost:3552"
  polling_enabled         = "true"
  polling_interval        = "10s"
}
```

## Argument Reference

- `environment_id` (String, Required)
- Optional string fields: see SettingsUpdate in `api-1.json` (e.g., `base_server_url`, `polling_enabled`, `polling_interval`, etc.)

## Attributes Reference

- `id` (String) — same as `environment_id`
- `applied` (Map(String)) — server values after apply

