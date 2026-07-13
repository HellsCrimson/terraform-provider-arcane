# arcane_container

Creates a single container in an environment.

Most config changes force replacement.

## Example Usage

```hcl
resource "arcane_container" "alpine" {
  environment_id = var.environment_id
  name           = "hello"
  image          = "alpine:latest"
  command        = ["sh", "-c", "sleep 3600"]
  ports          = { "8081/tcp" = "8081" }
  force_delete   = true
  remove_volumes = true
}
```

## Argument Reference

- `environment_id` (String, Required)
- `name` (String, Required, ForceNew)
- `image` (String, Required, ForceNew)
- Optional: `command`, `entrypoint`, `environment`, `networks`, `volumes` (List(String), ForceNew)
- Optional: `ports` (Map(String), ForceNew) — map container port to host port, numeric strings only (e.g., `{ "8081" = "8081" }`). Protocol defaults to TCP.
- Optional: `auto_remove`, `privileged` (Bool, ForceNew)
- Optional: `restart_policy`, `user`, `working_dir` (String, ForceNew)
- Optional: `cpus` (Float64, ForceNew), `memory` (Int64, ForceNew)
- Optional: `hostname`, `domain_name` (String, ForceNew) — container hostname and domain name
- Optional: `labels` (Map(String), ForceNew) — container labels for metadata
- Optional: `tty` (Bool, ForceNew) — allocate a pseudo-TTY
- Optional: `open_stdin` (Bool, ForceNew) — keep stdin open
- Optional: `stdin_once` (Bool, ForceNew) — close stdin after the client disconnects
- Optional: `attach_stdin`, `attach_stdout`, `attach_stderr` (Bool, ForceNew) — attach the respective stream
- Optional: `network_disabled` (Bool, ForceNew) — disable networking
- Optional: `fail_if_name_exists` (Bool) — If true, fail during the plan phase when a container with the same name already exists in the environment, instead of letting Arcane reject the create at apply time. Defaults to `false`.
- Delete behavior: `force_delete`, `remove_volumes`

## Attributes Reference

- `id`, `created`, `status`, `redeploy_disabled`
