# arcane_environment

Manages Arcane environments (agent connections).

## Example Usage

```
resource "arcane_environment" "agent" {
  name     = "Production"
  api_url  = "http://agent-host:8080"
  enabled  = true
  use_api_key = true
}
```

## Argument Reference

- `api_url` (String, Required) — agent API URL.
- `name` (String, Optional)
- `access_token` (String, Optional, Sensitive)
- `bootstrap_token` (String, Optional, Sensitive)
- `use_api_key` (Bool, Optional) — request Arcane to generate an API key for pairing.
- `enabled` (Bool, Optional)

## Attributes Reference

- `id` (String)
- `status` (String)
- `api_key` (String, Sensitive) — only returned on create when `use_api_key = true`.

