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

resource "arcane_container" "alpine" {
  environment_id = var.environment_id
  name           = "hello-alpine"
  image          = "alpine:latest"
  command        = ["sh", "-c", "sleep 3600"]
  # Ports map: container_port => host_port (numeric strings)
  ports          = { "8081" = "8081" }
  force_delete   = true
  remove_volumes = true
}

