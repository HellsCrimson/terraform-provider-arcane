# arcane_notification

Manages notification settings for a provider.

## Example Usage

```hcl
resource "arcane_notification" "discord" {
  environment_id = var.environment_id
  provider_name  = "discord"
  enabled        = true
  config = {
    avatarUrl = ""
    events = {
      container_update    = true
      image_update        = false
      prune_report        = true
      vulnerability_found = false
    }
    token     = "some token"
    username  = "User"
    webhookId = "id"
  }
}
```

## Argument Reference

- `environment_id` (String, Required)
- `provider_name` (String, Required)
- `enabled` (Bool, Required)
- `config` (Dynamic, Optional) — provider-specific configuration object. Nested objects are supported.

## Attributes Reference

- `id` (String) — `{env_id}:{provider_name}`
