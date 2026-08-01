# arcane_project

Manages a compose project with inline content.

## Example Usage

```hcl
resource "arcane_project" "demo" {
  environment_id  = var.environment_id
  name            = "demo"
  compose_content = file("${path.module}/docker-compose.yml")
  # env_content  = file("${path.module}/.env")
}
```

## Argument Reference

- `environment_id` (String, Required)
- `name` (String, Required)
- `compose_content` (String, Required)
- `env_content` (String, Optional)
- `archived` (Bool, Optional) — when true, brings the project down and archives it; when false, unarchives it.
- `redeploy_trigger` (String, Optional) — when the project is redeployed (default `default`). See [Redeploy trigger](#redeploy-trigger).
- `redeploy_on_update` (Bool, **Deprecated**) — use `redeploy_trigger` instead: `true` is equivalent to `redeploy_trigger = "default"`, `false` to `redeploy_trigger = "never"`. Setting both attributes is an error.
- `pull_on_update` (Bool, Optional) — when true, pulls images before each redeploy (default false).
- `remove_orphans` (Bool, Optional) — when deploying (compose up), remove containers for services not defined in the compose file.
- `running` (Bool, Optional) — when true, ensures the project is running (compose up); when false, brings it down. If unset, lifecycle is not managed.
- `fail_if_name_exists` (Bool, Optional) — when true, the plan fails if a project with the same `name` already exists in the environment (including folders Arcane has discovered on disk), instead of letting Arcane auto-rename the new project with a numeric suffix (default false). The check runs during the plan phase, so the collision is reported before any change is applied.
- `remove_files` (Bool, Optional) — remove files on destroy.
- `remove_volumes` (Bool, Optional) — remove volumes on destroy.

## Redeploy trigger

`redeploy_trigger` controls when the provider calls Arcane's redeploy endpoint:

| Value | Redeploys |
| --- | --- |
| `never` | Never. |
| `default` | When `compose_content` or `env_content` changed. This is the default, and matches the old `redeploy_on_update = true`. |
| `update` | On any in-place update of the resource, including changes that leave the compose/env content untouched. |
| `always` | On every apply, even when nothing changed. |

A redeploy is skipped regardless of the trigger when the project is archived or
when `running = false`, since redeploying brings a project up.

`always` is meant for projects whose inputs live outside Terraform's view — bind
mounted secrets or config files pushed to the host by another provider, for
instance:

```hcl
resource "arcane_project" "demo" {
  environment_id   = var.environment_id
  name             = "demo"
  compose_content  = file("${path.module}/docker-compose.yml")
  running          = true
  redeploy_trigger = "always"
}
```

Terraform only calls a provider when the plan is not empty, so `always` works by
marking `last_redeploy` as unknown during plan. **The resource therefore reports
a change on every plan**, even a plan that would otherwise be empty:

```
  ~ resource "arcane_project" "demo" {
      ~ last_redeploy = "2026-08-01T09:12:03Z" -> (known after apply)
    }
```

That is inherent to "redeploy unconditionally"; the other trigger values keep
producing empty plans when nothing changed.

## Attributes Reference

- `id`, `path`, `status`, `service_count`, `running_count`, `created_at`, `updated_at`
- `archived_at`, `is_discovered`, `redeploy_disabled`
- `last_redeploy` — RFC3339 timestamp of the last redeploy performed by the provider. Null until the provider has redeployed the project at least once (creating a project deploys it through `up`, not `redeploy`).
