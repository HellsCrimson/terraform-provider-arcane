# arcane_swarm_secret

Manages a Docker Swarm secret in an Arcane environment.

Swarm secrets are immutable; changing `name`, `data_wo_version`, or `labels` forces replacement.

## Example Usage

```hcl
resource "arcane_swarm_secret" "db_password" {
  environment_id  = var.environment_id
  name            = "db_password"
  data_wo         = var.db_password
  data_wo_version = 1

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
- `data_wo` (String, Sensitive, Write-only) - Secret value (plaintext), never stored in state or plan files. The provider encodes it to base64 for the API. See [write-only arguments](../index.md#write-only-arguments).
- `data_wo_version` (Number) - Version of `data_wo` (always set together with it). Change it to replace the secret with the configured `data_wo`.

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
