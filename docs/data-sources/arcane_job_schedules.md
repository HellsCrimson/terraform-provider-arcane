# arcane_job_schedules

Reads Arcane job schedules configuration for an environment.

## Example Usage

```hcl
data "arcane_job_schedules" "prod_schedules" {
  environment_id = "env-123456"
}

output "gitops_sync_interval" {
  value = data.arcane_job_schedules.prod_schedules.gitops_sync_interval
}

output "prune_interval" {
  value = data.arcane_job_schedules.prod_schedules.scheduled_prune_interval
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.

## Attributes Reference

- `id` (String) — resource ID (same as environment_id).
- `analytics_heartbeat_interval` (String) — analytics heartbeat cron expression.
- `auto_heal_interval` (String) — auto-heal cron expression.
- `auto_update_interval` (String) — auto-update check cron expression.
- `docker_client_refresh_interval` (String) — Docker client refresh cron expression.
- `environment_health_interval` (String) — environment health check cron expression.
- `event_cleanup_interval` (String) — event cleanup cron expression.
- `expired_sessions_cleanup_interval` (String) — expired sessions cleanup cron expression.
- `gitops_sync_interval` (String) — GitOps sync cron expression.
- `polling_interval` (String) — polling interval cron expression.
- `scheduled_prune_interval` (String) — scheduled prune cron expression.
- `vulnerability_scan_interval` (String) — vulnerability scan cron expression.

**Note:** All intervals use cron format (e.g., `0 */5 * * * *` for every 5 minutes).
