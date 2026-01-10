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

resource "arcane_environment" "agent" {
  name        = "Production"
  api_url     = "http://agent-host:8080"
  enabled     = true
  use_api_key = true
}

