# arcane_role_permissions

Returns the permission manifest: every permission the server recognizes, grouped by resource, plus preset bundles. Useful for building `arcane_role` / `arcane_api_key` permission lists.

## Example Usage

```hcl
data "arcane_role_permissions" "all" {}

# Grant every container-related permission to a role.
resource "arcane_role" "container_admin" {
  name        = "container-admin"
  permissions = one([
    for r in data.arcane_role_permissions.all.resources : r.permissions
    if r.key == "containers"
  ])
}

output "all_permissions" {
  value = data.arcane_role_permissions.all.all_permissions
}
```

## Argument Reference

This data source takes no arguments.

## Attributes Reference

- `all_permissions` (Set of String) — flattened, sorted set of every permission string the server recognizes.
- `resources` (List of Object) — permission groups, in display order. Each object has:
  - `key` (String) — stable resource key (e.g. `containers`).
  - `label` (String) — human-readable label.
  - `scope` (String) — `global` or `env`.
  - `permissions` (Set of String) — permission strings belonging to this resource.
- `presets` (List of Object) — optional preset permission bundles for bulk selection. Each object has:
  - `key` (String) — stable preset key.
  - `label` (String) — human-readable preset label.
  - `permissions` (Set of String) — permissions included in the preset.
