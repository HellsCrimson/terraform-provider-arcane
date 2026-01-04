Arcane Terraform Provider

Manage Arcane using Terraform or OpenTofu. This provider talks to the Arcane HTTP API using an API key and implements common workflows: users, environment settings, compose projects (inline or from files), project state (up/down), notifications, and single containers.

Overview

- Auth via `X-API-Key` header.
- Provider address used in this repository: `registry.terraform.io/hellscrimson/arcane`.

Requirements

- Terraform or OpenTofu 1.4+.
- Go 1.21+ (to build from source).

Installation

- From Registry (recommended):
```
terraform {
  required_providers {
    arcane = {
      source  = "hellscrimson/arcane"
      version = "~> 0.0.1"
    }
  }
}
```

- Local development override:
  1) Build the binary:

```
go build ./cmd/terraform-provider-arcane
```

  2) Add a dev override in your CLI config (e.g. `~/.terraformrc`):

```
dependency_lock_file_path = "./.terraform.lock.hcl"
provider_installation {
  dev_overrides {
    "hellscrimson/arcane" = "/abs/path/to/your/build/folder"
  }
  direct {}
}
```

  3) In your configuration, set:

```
terraform {
  required_providers {
    arcane = { source = "hellscrimson/arcane" }
  }
}
```

Authentication

- API key: provider attribute `api_key` or environment `ARCANE_API_KEY`.
- Endpoint: provider attribute `endpoint` (defaults to `http://localhost:3552/api`).

Quick Start

See `examples/basic/main.tf` for a working setup that demonstrates projects, file-based projects (with content hashing), notifications and containers. Example provider block:

```
provider "arcane" {
  api_key  = var.arcane_api_key
  endpoint = "http://localhost:3552/api"
}

variable "environment_id" {
  type = string
}
```

Resources

- arcane_user
  - Create/read/update/delete Arcane users.
  - Attributes: username (required, replace), password (required, sensitive), display_name, email, locale, roles.
  - Note: Older runtimes do not support write-only attributes; password is stored sensitive in state for apply consistency.

- arcane_settings
  - Update environment settings using explicit attributes mapped from SettingsUpdate.
  - Typical fields: base_server_url, polling_enabled, polling_interval, and many others defined in `api-1.json`.
  - Computed `applied` map exposes the server’s current settings after apply.

- arcane_project
  - Manage a compose project with inline content.
  - Attributes: environment_id, name, compose_content (required), env_content (optional).
  - Computed: id, path, status, service_count, running_count, created_at, updated_at.

- arcane_project_path
  - Manage a compose project from local files.
  - Attributes: environment_id, name, compose_path (required), env_path (optional).
  - content_hash_mode (bool): when true, state stores only SHA256 hashes (compose_content_hash/env_content_hash) instead of full content; still detects file changes and updates.
  - When false (default), state stores the last read file contents (sensitive) to detect changes.

- arcane_project_state
  - Manage the running state of a compose project (up/down).
  - Attributes: environment_id, project_id, running (bool), status (computed).
  - Example: set `running = true` to ensure the project is deployed (compose up).

- arcane_notification
  - Manage notification settings for a provider.
  - Attributes: environment_id, provider_name, enabled, config (map(string)).

- arcane_container
  - Create/delete a single container.
  - Attributes: environment_id, name, image (required), and advanced options (command, ports, volumes, etc.). Most changes force replacement.
  - Ports map format: container port -> host port, numeric strings only (e.g., `{ "8081" = "8081" }`). The provider normalizes values if a protocol suffix is present.

Imports

- arcane_user: `id`
- arcane_settings: `environment_id`
- arcane_project: `environment_id:project_id`
- arcane_project_path: `environment_id:project_id`
- arcane_project_state: `environment_id:project_id`
- arcane_notification: `environment_id:provider_name`
- arcane_container: `environment_id:container_id`

Examples

- Full example: `examples/basic/main.tf`

Building from source

- Ensure Go 1.21+ then build:

```
go build ./cmd/terraform-provider-arcane
```

Releases & Publishing

- This repo includes a minimal GoReleaser configuration (`.goreleaser.yaml`) to build multi-platform archives, generate checksums and GPG-sign the checksum file.

API Coverage & Notes

- This provider adheres to the OpenAPI available in an arcane instance.
- Implemented endpoints:
  - Users: `POST /users`, `GET/PUT/DELETE /users/{userId}`
  - Settings: `GET/PUT /environments/{id}/settings`
  - Projects: `POST /environments/{id}/projects`, `GET/PUT /environments/{id}/projects/{projectId}`, `DELETE /environments/{id}/projects/{projectId}/destroy`, `POST /environments/{id}/projects/{projectId}/up|down`
  - Notifications: `POST /environments/{id}/notifications/settings`, `GET/DELETE /environments/{id}/notifications/settings/{provider}`
  - Containers: `POST /environments/{id}/containers`, `GET/DELETE /environments/{id}/containers/{containerId}` (supports `force` and `volumes` on delete)

Limitations / Roadmap

- No data sources yet.
- Additional endpoints (images, registries, templates, etc.) not covered.
- When using Terraform/OpenTofu < 1.11, write-only attributes are not available; sensitive-only storage is used for passwords.

Contributing

- PRs welcome. Please keep changes minimal and aligned with the OpenAPI.
- Before opening a PR, run:

```
go mod tidy
go build ./cmd/terraform-provider-arcane
```

