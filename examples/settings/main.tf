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

resource "arcane_settings" "env" {
  environment_id          = var.environment_id
  base_server_url         = "http://localhost:3552"
  polling_enabled         = "true"
  polling_interval        = "0 */15 * * * *"  # Cron expression: every 15 minutes
  sidebar_hover_expansion = "true"

  # GitOps pre-deploy lifecycle hooks (disabled by default): allow syncs to
  # configure pre-deploy scripts, with a default runner image and a cap on
  # the per-sync timeout.
  lifecycle_enabled              = "true"
  lifecycle_default_runner_image = "ghcr.io/getsops/sops:v3.11.0"
  lifecycle_max_timeout_sec      = "300"
}

