terraform {
  required_version = ">= 1.11.0"
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
variable "ghcr_token" {
  type      = string
  sensitive = true
}

resource "arcane_container_registry" "ghcr" {
  url      = "https://ghcr.io"
  username = "bot"
  # Write-only: never stored in state. Change token_wo_version to send a new one.
  token_wo         = var.ghcr_token
  token_wo_version = 1
  description      = "GitHub Container Registry"
  insecure         = false
  enabled          = true

  repository_names = ["my-org", "my-org/platform"]
}

