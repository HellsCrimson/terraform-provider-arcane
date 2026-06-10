# arcane_swarm_secret

Manages a Docker Swarm secret in an Arcane environment.

Swarm secrets are immutable; changing `name`, `data`, or `labels` forces replacement.

## Example Usage

```hcl
resource "arcane_swarm_secret" "db_password" {
  environment_id = var.environment_id
  name           = "db_password"
  data           = "super-secret-password"

  labels = {
    "app" = "demo"
    "env" = "prod"
  }
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `name` (String) - Secret name. Changing this forces a new resource.
- `data` (String, Sensitive) - Secret value (plaintext). The provider encodes this to base64 for the API. Changing this forces a new resource.

### Optional

- `labels` (Map of String) - Secret labels. Changing this forces a new resource.

## Attributes Reference

- `id` (String) - Swarm secret ID.
- `version_index` (Number) - Swarm object version index.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the format `environment_id:secret_id`:

```
terraform import arcane_swarm_secret.db_password <environment_id>:<secret_id>
```
