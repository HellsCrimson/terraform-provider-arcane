# arcane_notification

Manages notification settings for a provider. Notification settings are app-wide (not per environment).

## Example Usage

```hcl
resource "arcane_notification" "discord" {
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

- `provider_name` (String, Required)
- `enabled` (Bool, Required)
- `config` (Dynamic, Optional) — provider-specific configuration object. Nested objects are supported.

## Attributes Reference

- `id` (String) — the provider name.

## Import

```shell
terraform import arcane_notification.discord discord
```
