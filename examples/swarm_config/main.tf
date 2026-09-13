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

resource "arcane_swarm_config" "app_config" {
  environment_id = var.environment_id
  name           = "app_config"
  data           = "greeting: hello\nlevel: info"

  labels = {
    "app" = "demo"
    "env" = "prod"
  }
}

data "arcane_swarm_config" "app_config" {
  environment_id = var.environment_id
  id             = arcane_swarm_config.app_config.id
}

output "swarm_config_id" {
  value = arcane_swarm_config.app_config.id
}

output "swarm_config_version" {
  value = data.arcane_swarm_config.app_config.version_index
}
