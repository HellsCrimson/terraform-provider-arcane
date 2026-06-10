# arcane_swarm_stack

Reads an Arcane Docker Swarm stack, including its source compose content.

## Example Usage

```hcl
data "arcane_swarm_stack" "demo" {
  environment_id = var.environment_id
  id             = "demo-swarm-stack"
}

output "stack_namespace" {
  value = data.arcane_swarm_stack.demo.namespace
}

output "stack_compose_content" {
  value = data.arcane_swarm_stack.demo.compose_content
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — stack name.

## Attributes Reference

- `name` (String) — stack name.
- `compose_content` (String) — Docker Compose content for the stack.
- `env_content` (String) — `.env` content for the stack.
- `namespace` (String) — Docker namespace for the stack.
- `services` (Number) — number of services in the stack.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.
