# arcane_notification

Reads an Arcane notification provider configuration (notification settings are app-wide).

## Example Usage

```hcl
data "arcane_notification" "slack" {
  provider_name = "slack"
}

output "notification_enabled" {
  value = data.arcane_notification.slack.enabled
}
```

## Argument Reference

- `provider_name` (String, Required) — notification provider name.

## Attributes Reference

- `id` (String) — notification ID (the provider name).
- `enabled` (Bool) — whether the notification is enabled.
- `config` (Dynamic) — provider-specific configuration object. Nested objects are preserved.
