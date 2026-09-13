# arcane_swarm_config

Manages a Docker Swarm config in an Arcane environment.

Swarm configs are immutable; changing `name`, `data`, or `labels` forces replacement.

## Example Usage

```hcl
resource "arcane_swarm_config" "app_config" {
  environment_id = var.environment_id
  name           = "app_config"
  data           = file("${path.module}/config.yml")

  labels = {
    "app" = "demo"
    "env" = "prod"
  }
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `name` (String) - Config name. Changing this forces a new resource.
- `data` (String) - Config content (plaintext). The provider encodes this to base64 for the API. Changing this forces a new resource.

### Optional

- `labels` (Map of String) - Config labels. Changing this forces a new resource.

## Attributes Reference

- `id` (String) - Swarm config ID.
- `version_index` (Number) - Swarm object version index.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the format `environment_id:config_id`:

```
terraform import arcane_swarm_config.app_config <environment_id>:<config_id>
```

Unlike swarm secrets, the config API returns the stored content, so `data` and
`labels` are recovered during the import refresh: an import of a config whose
configuration already matches plans clean instead of forcing a replacement. A
config holding non-UTF-8 content cannot be represented in a Terraform string;
the import then warns and leaves `data` unset, which does force a replacement on
the next apply.
