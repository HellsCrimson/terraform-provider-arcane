# Arcane Provider

The Arcane provider allows managing Arcane via its HTTP API using an API key.

## Example Usage

```hcl
provider "arcane" {
  api_key  = var.arcane_api_key
  endpoint = "http://localhost:3552/api"
  insecure = true
}
```

## Argument Reference

- `api_key` (String, Sensitive) — API key; alternatively set `ARCANE_API_KEY`.
- `endpoint` (String) — Base API URL. Defaults to `http://localhost:3552/api`.
- `insecure` (Boolean) — Disable TLS certificate verification for API requests. Defaults to `false`.

## Write-only arguments

Secrets have a write-only variant, named after the original argument with a
`_wo` suffix. Terraform passes it to the provider, which sends it to Arcane,
but never stores it in the state or in saved plan files. This requires
Terraform or OpenTofu 1.11 or later. The value can come from an ephemeral
resource:

```hcl
ephemeral "vault_kv_secret_v2" "github" {
  mount = "secret"
  name  = "github"
}

resource "arcane_git_repository" "app" {
  name             = "app"
  url              = "https://github.com/example/app.git"
  auth_type        = "token"
  token_wo         = ephemeral.vault_kv_secret_v2.github.data.token
  token_wo_version = 1
}
```

Terraform cannot diff a value it does not keep, so every `_wo` argument has a
required `_wo_version` companion. Editing the secret alone plans no change;
change the version in the same edit to send the new value.

| Resource | Write-only arguments |
|----------|----------------------|
| `arcane_user` | `password_wo` |
| `arcane_git_repository` | `token_wo`, `ssh_key_wo` |
| `arcane_container_registry` | `token_wo`, `aws_access_key_id_wo`, `aws_secret_access_key_wo` |
| `arcane_environment` | `access_token_wo` |
| `arcane_settings` | `oidc_client_secret_wo`, `depot_token_wo`, `trivy_server_token_wo` |
| `arcane_gitops_sync` | `pre_deploy_env_wo` |
| `arcane_swarm_secret` | `data_wo` (changing `data_wo_version` replaces the secret) |

The original arguments still work but are deprecated: they keep the secret in
the state, and will be removed in the next major release. To migrate, replace
the argument with its `_wo` variant and set `_wo_version`. The next apply
updates the resource in place (a swarm secret is not replaced) and drops the
secret from the state; it sends the `_wo` value, so it can also rotate it.

## Authentication

Uses header `X-API-Key` per the OpenAPI spec.
