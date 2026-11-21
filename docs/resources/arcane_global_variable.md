# arcane_global_variable

Manages an Arcane global variable. Global variables are available to the projects and templates of every environment they are scoped to, and can also be injected into the containers of those environments.

## Example Usage

```hcl
# Plain variable, available everywhere.
resource "arcane_global_variable" "domain" {
  key   = "BASE_DOMAIN"
  value = "example.com"
}

# Secret variable for one environment, injected into its containers.
resource "arcane_global_variable" "smtp_password" {
  key                    = "SMTP_PASSWORD"
  value_wo               = var.smtp_password
  value_wo_version       = 1
  environment_ids        = [arcane_environment.prod.id]
  inject_into_containers = true
}
```

## Argument Reference

### Required

- `key` (String) - Variable name.

### Optional

- `value` (String) - Value of a plain variable, stored in state. Conflicts with `value_wo`.
- `value_wo` (String, Sensitive, Write-only) - Value of a secret variable. Never stored in state or plan files; only sent when `value_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `value_wo_version` (Number) - Change it to send a new `value_wo`. Required with `value_wo`.
- `is_secret` (Boolean) - Hide the value in Arcane; Arcane never returns a secret's value. Defaults to `true` with `value_wo`, `false` otherwise. A secret must use `value_wo`.
- `all_environments` (Boolean) - Make the variable available in every environment. Defaults to `true` unless `environment_ids` is set.
- `environment_ids` (Set of String) - Environments the variable is available in, when `all_environments` is `false`.
- `inject_into_containers` (Boolean) - Also inject the variable into the containers of those environments. Defaults to `false`.

## Attributes Reference

- `id` (String) - Variable ID.
- `created_at` (String) - Creation timestamp.

## Import

```shell
terraform import arcane_global_variable.domain <variable-id>
```

An imported secret has no value in state; set `value_wo` and `value_wo_version` to manage it.
