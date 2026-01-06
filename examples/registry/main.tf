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
  type = string
  sensitive = true
}
variable "arcane_endpoint" {
  type = string
  default = "http://localhost:3552/api"
}
variable "ghcr_token" {
  type = string
  sensitive = true
}

resource "arcane_container_registry" "ghcr" {
  url         = "https://ghcr.io"
  username    = "bot"
  token       = var.ghcr_token
  description = "GitHub Container Registry"
  insecure    = false
  enabled     = true
}

