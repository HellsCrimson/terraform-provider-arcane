# arcane_role

Manages a custom RBAC role. Built-in roles (Admin/Editor/Deployer/Viewer) are read-only and cannot be managed by this resource.

## Example Usage

```hcl
resource "arcane_role" "deployer" {
  name        = "compose-deployer"
  description = "Can start and stop containers"
  permissions = [
    "containers:start",
    "containers:stop",
    "containers:view",
  ]
}
```

Use the `arcane_role_permissions` data source to discover valid permission strings.

## Argument Reference

### Required

- `name` (String) - Display name of the role (1-100 characters).
- `permissions` (Set of String) - Permission strings granted by this role (at least one), e.g. `containers:start`.

### Optional

- `description` (String) - Optional human description (max 500 characters).

## Attributes Reference

- `id` (String) - Unique identifier of the role.
- `built_in` (Boolean) - Whether this is a built-in role (always `false` for managed roles).
- `assigned_user_count` (Number) - How many users currently hold an assignment to this role.
- `created_at` (String) - Creation timestamp.
- `updated_at` (String) - Last update timestamp.

## Import

Import using the role ID:

```
terraform import arcane_role.deployer <role_id>
```
