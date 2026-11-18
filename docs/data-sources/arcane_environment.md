# arcane_environment

Reads an Arcane environment configuration.

## Example Usage

```hcl
data "arcane_environment" "prod" {
  id = "env-123456"
}

output "environment_status" {
  value = data.arcane_environment.prod.status
}
```

## Argument Reference

- `id` (String, Required) — environment ID.

## Attributes Reference

- `name` (String) — environment display name.
- `api_url` (String) — environment URL (`agent://<name>` for agent environments).
- `status` (String) — environment status.
- `enabled` (Bool) — whether the environment is enabled.
- `edge_agent_instance`, `edge_security_mode`, `edge_session_id` (String) — agent details.
- `edge_capabilities` (List of String)
- `edge_mtls_certificate_json` (String) — agent mTLS certificate metadata as JSON.
