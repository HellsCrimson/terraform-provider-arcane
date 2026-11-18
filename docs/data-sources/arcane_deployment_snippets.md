# arcane_deployment_snippets

Reads agent deployment snippets for an environment.

~> **Note:** every read mints a new single-use enrollment token (embedded in the
snippets), so each plan or refresh produces a fresh token. For a stable token use the
`enrollment_token` of the [`arcane_environment`](../resources/arcane_environment.md) resource.

## Example Usage

```hcl
data "arcane_deployment_snippets" "snippets" {
  environment_id = "env-123456"
}

output "docker_run" {
  value = data.arcane_deployment_snippets.snippets.docker_run
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `docker_run` (String, Sensitive) - docker run command snippet (embeds the enrollment token).
- `docker_compose` (String, Sensitive) - docker compose YAML snippet (embeds the enrollment token).
- `enrollment_token` (String, Sensitive) - single-use enrollment token minted by this read.
- `mtls_docker_run` (String) - mTLS docker run command snippet, when available.
- `mtls_docker_compose` (String) - mTLS docker compose YAML snippet, when available.
- `mtls_host_dir_hint` (String) - Suggested mTLS asset host directory, when available.
- `mtls_files_json` (String, Sensitive) - Generated mTLS file metadata as JSON.
