# arcane_container_registry

Manages a container registry in Arcane.

## Example Usage

```
resource "arcane_container_registry" "example" {
  url         = "https://ghcr.io"
  username    = "bot"
  token       = var.ghcr_token
  description = "GitHub Container Registry"
  insecure    = false
  enabled     = true
}
```

## Argument Reference

- `url` (String, Required)
- `username` (String, Required)
- `token` (String, Required, Sensitive)
- `description` (String, Optional)
- `insecure` (Bool, Optional)
- `enabled` (Bool, Optional)

## Attributes Reference

- `id` (String)
- `created_at` (String)
- `updated_at` (String)

