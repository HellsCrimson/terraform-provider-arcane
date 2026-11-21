# arcane_job_schedules

Manages cron schedules for automated background jobs in an environment.

## Example Usage

```hcl
resource "arcane_job_schedules" "env" {
  environment_id = var.environment_id

  # Check for updates daily at 2 AM
  auto_update_interval = "0 2 * * *"

  # Environment health check every minute
  environment_health_interval = "*/1 * * * *"

  # Automatic image patching daily at 4 AM
  image_auto_patch_interval = "0 4 * * *"

  # Prune Docker resources weekly on Sunday at 1 AM
  scheduled_prune_interval = "0 1 * * 0"
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID.

### Optional

All interval attributes use standard 5-field cron format (minute hour day-of-month month day-of-week):

- `auto_heal_interval` (String) - Cron expression for auto-heal checks.
- `auto_update_interval` (String) - Cron expression for auto-update checks (e.g., '0 2 * * *' for daily at 2 AM).
- `docker_client_refresh_interval` (String) - Cron expression for Docker client refresh.
- `environment_health_interval` (String) - Cron expression for environment health checks (e.g., '*/1 * * * *' for every minute).
- `event_cleanup_interval` (String) - Cron expression for event log cleanup (e.g., '0 3 * * *' for daily at 3 AM).
- `expired_sessions_cleanup_interval` (String) - Cron expression for expired session cleanup.
- `image_auto_patch_interval` (String) - Cron expression for automatic image patching.
- `polling_interval` (String) - Cron expression for general polling operations (e.g., '*/5 * * * *' for every 5 minutes).
- `scheduled_prune_interval` (String) - Cron expression for scheduled pruning of Docker resources (e.g., '0 1 * * 0' for weekly on Sunday at 1 AM).
- `vulnerability_scan_interval` (String) - Cron expression for vulnerability scans.

## Attributes Reference

- `id` (String) - Resource ID (same as environment_id).

## Import

Import using the environment ID:

```
terraform import arcane_job_schedules.env <environment_id>
```

## Cron Format

The cron expressions use the standard 5-field format:

```
┌──────────── minute (0-59)
│ ┌────────── hour (0-23)
│ │ ┌──────── day of month (1-31)
│ │ │ ┌────── month (1-12)
│ │ │ │ ┌──── day of week (0-6, 0=Sunday)
│ │ │ │ │
* * * * *
```

Examples:
- `*/5 * * * *` - Every 5 minutes
- `0 * * * *` - Every hour
- `0 2 * * *` - Daily at 2:00 AM
- `0 0 * * 0` - Weekly on Sunday at midnight
- `0 0 1 * *` - Monthly on the 1st at midnight
