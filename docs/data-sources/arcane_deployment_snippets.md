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
