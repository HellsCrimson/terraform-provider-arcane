# arcane_environment

Manages Arcane environments (agent connections).

## Example Usage

```hcl
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
- `access_token` (String, Optional, Sensitive) — access token for agent pairing. **Deprecated**: stored in state; use `access_token_wo` instead, `access_token` will be removed in the next major release.
- `access_token_wo` (String, Optional, Sensitive, Write-only) — access token for agent pairing. Never stored in state or plan files; only sent when `access_token_wo_version` changes. Conflicts with `access_token`. See [write-only arguments](../index.md#write-only-arguments).
- `access_token_wo_version` (Number, Optional) — change it to send a new `access_token_wo`. Required with `access_token_wo`.
- `use_api_key` (Bool, Optional) — request Arcane to generate an API key for pairing.
- `is_edge` (Bool, Optional) — whether the environment uses edge transport.
- `regenerate_api_key` (Bool, Optional) — regenerate the pairing API key on update.
- `enabled` (Bool, Optional)

## Attributes Reference

- `id` (String)
- `status` (String)
- `api_key` (String, Sensitive) — only returned on create when `use_api_key = true`.
- `edge_agent_instance`, `edge_security_mode`, `edge_session_id` (String)
- `edge_capabilities` (List of String)
- `edge_mtls_certificate_json` (String) — edge mTLS certificate metadata as JSON.
