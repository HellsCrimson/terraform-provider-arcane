# arcane_deployment_snippets

Reads deployment snippets for an environment.

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

- `docker_run` (String) - docker run command snippet.
- `docker_compose` (String) - docker compose YAML snippet.
- `mtls_docker_run` (String) - mTLS docker run command snippet, when available.
- `mtls_docker_compose` (String) - mTLS docker compose YAML snippet, when available.
- `mtls_host_dir_hint` (String) - Suggested mTLS asset host directory, when available.
- `mtls_files_json` (String, Sensitive) - Generated mTLS file metadata as JSON.
