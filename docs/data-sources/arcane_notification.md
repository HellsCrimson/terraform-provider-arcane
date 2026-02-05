# arcane_notification

Reads an Arcane notification provider configuration.

## Example Usage

```hcl
data "arcane_notification" "slack" {
  environment_id = "env-123456"
  provider_name  = "slack"
}

output "notification_enabled" {
  value = data.arcane_notification.slack.enabled
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `provider_name` (String, Required) — notification provider name.

## Attributes Reference

- `id` (String) — notification ID (environment_id:provider_name).
- `enabled` (Bool) — whether the notification is enabled.
- `config` (Map of String) — provider-specific configuration.
