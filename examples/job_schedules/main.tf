terraform {
  required_version = ">= 1.4.0"
  required_providers {
    arcane = {
      source  = "hellscrimson/arcane"
      version = ">= 0.0.1"
    }
  }
}

provider "arcane" {
  api_key  = var.arcane_api_key
  endpoint = var.arcane_endpoint
}

variable "arcane_api_key" {
  type      = string
  sensitive = true
}

variable "arcane_endpoint" {
  type    = string
  default = "http://localhost:3552/api"
}

variable "environment_id" {
  type = string
}

# Configure job schedules for automated maintenance
resource "arcane_job_schedules" "production" {
  environment_id = var.environment_id

  # Health checks every minute
  environment_health_interval = "0 */1 * * * *"

  # GitOps sync every 5 minutes
  gitops_sync_interval = "0 */5 * * * *"

  # Polling every 10 minutes
  polling_interval = "0 */10 * * * *"

  # Auto-update check daily at 2 AM
  auto_update_interval = "0 0 2 * * *"

  # Event cleanup daily at 3 AM
  event_cleanup_interval = "0 0 3 * * *"

  # Analytics heartbeat every 15 minutes
  analytics_heartbeat_interval = "0 */15 * * * *"

  # Scheduled prune weekly on Sunday at 1 AM
  scheduled_prune_interval = "0 0 1 * * 0"
}

# Example: Development environment with less frequent checks
resource "arcane_job_schedules" "development" {
  environment_id = var.environment_id

  # Health checks every 5 minutes (less frequent than prod)
  environment_health_interval = "0 */5 * * * *"

  # GitOps sync every 15 minutes
  gitops_sync_interval = "0 */15 * * * *"

  # No auto-updates in dev
  # auto_update_interval not set

  # Event cleanup weekly
  event_cleanup_interval = "0 0 3 * * 0"
}

# Cron format reference:
# ┌───────────── second (0-59)
# │ ┌───────────── minute (0-59)
# │ │ ┌───────────── hour (0-23)
# │ │ │ ┌───────────── day of month (1-31)
# │ │ │ │ ┌───────────── month (1-12)
# │ │ │ │ │ ┌───────────── day of week (0-6) (Sunday=0)
# │ │ │ │ │ │
# * * * * * *
#
# Examples:
# "0 */15 * * * *" - Every 15 minutes
# "0 0 * * * *"    - Every hour
# "0 0 2 * * *"    - Daily at 2:00 AM
# "0 0 0 * * 0"    - Weekly on Sunday at midnight
# "0 0 0 1 * *"    - First day of each month
