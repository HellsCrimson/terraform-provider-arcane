# arcane_notification

Manages notification settings for a provider.

## Example Usage

```
resource "arcane_notification" "slack" {
  environment_id = var.environment_id
  provider_name  = "slack"
  enabled        = true
  config = {
    webhook = "https://hooks.slack.com/services/..."
    channel = "#deploys"
  }
}
```

## Argument Reference

- `environment_id` (String, Required)
- `provider_name` (String, Required)
- `enabled` (Bool, Required)
- `config` (Map(String), Optional)

## Attributes Reference

- `id` (String) — `{env_id}:{provider_name}`
