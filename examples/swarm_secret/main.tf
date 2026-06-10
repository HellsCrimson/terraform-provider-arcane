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

resource "arcane_swarm_secret" "db_password" {
  environment_id = var.environment_id
  name           = "db_password"
  data           = "super-secret-password"

  labels = {
	"app" = "demo"
	"env" = "prod"
  }
}

data "arcane_swarm_secret" "db_password" {
  environment_id = var.environment_id
  id             = arcane_swarm_secret.db_password.id
}

output "swarm_secret_id" {
  value = arcane_swarm_secret.db_password.id
}

output "swarm_secret_version" {
  value = data.arcane_swarm_secret.db_password.version_index
}
