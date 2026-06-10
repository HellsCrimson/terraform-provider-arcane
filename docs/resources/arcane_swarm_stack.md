# arcane_swarm_stack

Manages a Docker Swarm stack in an Arcane environment.

## Example Usage

```hcl
resource "arcane_swarm_stack" "demo" {
  environment_id  = var.environment_id
  name            = "demo-swarm-stack"
  compose_content = file("${path.module}/stack-compose.yml")
  env_content     = <<EOF
APP_ENV=production
NGINX_PORT=8080
EOF

  prune              = true
  resolve_image      = "changed"
  with_registry_auth = false
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `name` (String) - Swarm stack name. Changing this forces a new resource.
- `compose_content` (String) - Docker Compose content for stack deployment.

### Optional

- `env_content` (String) - `.env` content for stack deployment.
- `prune` (Boolean) - Prune services that are no longer referenced in the compose file during deploy. Changing this forces a new resource.
- `resolve_image` (String) - Image resolution mode for deploy (for example `always`, `changed`, or `never`). Changing this forces a new resource.
- `with_registry_auth` (Boolean) - Forward registry authentication to swarm agents during deploy. Changing this forces a new resource.
- `working_dir` (String) - Working directory used by Arcane for compose include resolution. Changing this forces a new resource.

## Attributes Reference

- `id` (String) - Stack name (resource ID).
- `namespace` (String) - Docker namespace for the stack.
- `services` (Number) - Number of services in the stack.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the format `environment_id:stack_name`:

```
terraform import arcane_swarm_stack.demo <environment_id>:<stack_name>
```
