# arcane_environment

Manages Arcane environments (agent connections).

Environments are agent-based: Terraform creates the environment, Arcane mints a single-use
**enrollment token**, and an Arcane agent started with that token (`ARCANE_AGENT_TOKEN`) joins the
environment. The environment stays `pending` until an agent enrolls. Arcane derives the environment
URL itself (`agent://<name>`).

## Example Usage

```hcl
resource "arcane_environment" "agent" {
  name    = "Production"
  enabled = true
}

# Start the agent with the token, e.g. through the docker provider:
# env = ["ARCANE_AGENT=true", "ARCANE_AGENT_TOKEN=${arcane_environment.agent.enrollment_token}",
#        "ARCANE_MANAGER_API_URL=https://arcane.example.com"]
```

## Argument Reference

- `name` (String, Optional) — display name.
- `enabled` (Bool, Optional)

## Attributes Reference

- `id` (String)
- `status` (String) — `pending` until an agent enrolls.
- `api_url` (String) — URL Arcane derived for the environment.
- `enrollment_token` (String, Sensitive) — single-use enrollment token minted on create. Null for imported environments; use the [`arcane_deployment_snippets`](../data-sources/arcane_deployment_snippets.md) data source to mint a new one.
- `edge_agent_instance`, `edge_security_mode`, `edge_session_id` (String) — agent details.
- `edge_capabilities` (List of String)
- `edge_mtls_certificate_json` (String) — agent mTLS certificate metadata as JSON.

## Import

```shell
terraform import arcane_environment.agent <environment-id>
```
