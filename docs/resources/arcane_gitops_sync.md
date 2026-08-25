# arcane_gitops_sync

Manages GitOps sync configurations that automatically deploy docker-compose projects from Git repositories.

## Example Usage

```hcl
resource "arcane_environment" "prod" {
  name     = "Production"
  api_url  = "http://agent:8080"
  enabled  = true
  use_api_key = true
}

resource "arcane_git_repository" "app_repo" {
  name      = "App Repository"
  url       = "https://github.com/user/app-repo.git"
  auth_type = "token"
  username  = "github-user"
  token     = var.github_token
  enabled   = true
}

resource "arcane_gitops_sync" "app_sync" {
  environment_id = arcane_environment.prod.id
  name           = "App Sync"
  repository_id  = arcane_git_repository.app_repo.id
  branch         = "main"
  compose_path   = "docker-compose.yml"
  project_name   = "my-app"

  auto_sync     = true
  sync_interval = 300  # 5 minutes
  enabled       = true
}
```

## Example with Custom Compose Path

```hcl
resource "arcane_gitops_sync" "staging_sync" {
  environment_id = arcane_environment.staging.id
  name           = "Staging Deployment"
  repository_id  = arcane_git_repository.app_repo.id
  branch         = "develop"
  compose_path   = "deploy/staging/docker-compose.yml"
  project_name   = "app-staging"

  auto_sync     = true
  sync_interval = 600  # 10 minutes
  enabled       = true
}
```

## Argument Reference

- `environment_id` (String, Required) — Environment ID (changing forces new resource)
- `name` (String, Required) — Sync configuration name
- `repository_id` (String, Required) — Git repository ID
- `branch` (String, Required) — Git branch to sync from
- `compose_path` (String, Required) — Path to docker-compose file in the repository
- `project_name` (String, Optional) — Project name for the compose stack. Changing it renames the project the sync is bound to; see [Renaming the project](#renaming-the-project).
- `auto_sync` (Bool, Optional) — Enable automatic sync on interval
- `sync_interval` (Int, Optional) — Sync interval in seconds
- `sync_directory` (Bool, Optional) — Whether to sync the full directory instead of only the compose file
- `target_type` (String, Optional) — GitOps sync target type
- `max_sync_binary_size` (Int, Optional) — Maximum binary file size to sync, in bytes
- `max_sync_files` (Int, Optional) — Maximum number of files to sync
- `max_sync_total_size` (Int, Optional) — Maximum total sync size, in bytes
- `environment_variables` (Map of String, Optional) — Environment variables for the synced project
- `start_project` (Bool, Optional) — Whether to start the project after creation (default: `true`). Controls lifecycle behavior only; not sent to the API.
- `fail_if_name_exists` (Bool, Optional) — If true, fail during the plan phase when a GitOps sync with the same name already exists in the environment, instead of creating a duplicate. Defaults to `false`.
- `stop_before_rename` (Bool, Optional) — when true, a `project_name` change stops the project, renames it and starts it again in the same apply (default `false`). Arcane only renames stopped projects; without this, the plan fails when the rename targets a running project. See [Renaming the project](#renaming-the-project).
- `enabled` (Bool, Optional) — Whether the sync is enabled

## Renaming the project

Arcane stores `project_name` on the sync record only: `PUT /gitops-syncs/{id}`
never touches the project, and a sync run looks its project up by ID. So the
provider renames the project itself when `project_name` changes, which puts the
change under the rule Arcane enforces on every project rename — only a stopped
project is renamed:

```
project must be stopped before renaming (current status: running)
```

The provider checks the project's status during the **plan**, so a rename that
would be rejected fails before anything is applied:

```
Error: rename requires a stopped project

  with arcane_gitops_sync.app_sync,
  on main.tf line 7, in resource "arcane_gitops_sync" "app_sync":
   7:   project_name = "my-app-renamed"

Renaming "my-app" to "my-app-renamed" would fail during apply: Arcane only
renames a project that is stopped, and this one is running.
```

Set `stop_before_rename = true` to have the provider stop the project, rename it
and start it again within the same apply (the project is down for the duration
of the rename), or stop the project before applying the change. Nothing happens
to a sync that has no project yet: the first sync creates it under the new name.

Only a `project_name` change renames. A project whose name drifted from the sync
record — for instance one renamed outside Terraform — is left alone until the
`project_name` in the configuration changes.

## Attributes Reference

- `id` (String) — GitOps sync ID
- `project_id` (String) — Associated project ID (created after first sync)
- `last_sync_at` (String) — Last sync timestamp
- `last_sync_commit` (String) — Last synced commit hash
- `last_sync_status` (String) — Last sync status
- `last_sync_error` (String) — Last sync error message (if any)
- `created_at` (String) — Creation timestamp
- `updated_at` (String) — Last update timestamp
