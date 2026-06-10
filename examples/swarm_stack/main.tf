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

resource "arcane_swarm_stack" "demo" {
  environment_id      = var.environment_id
  name                = "demo-swarm-stack"
  compose_content     = file("${path.module}/stack-compose.yml")
  env_content         = <<EOF
APP_ENV=production
NGINX_PORT=8080
EOF
  prune               = true
  resolve_image       = "changed"
  with_registry_auth  = false
}

# Read back the deployed stack and source content.
data "arcane_swarm_stack" "demo" {
  environment_id = var.environment_id
  id             = arcane_swarm_stack.demo.id
}

output "stack_namespace" {
  value = arcane_swarm_stack.demo.namespace
}

output "stack_services" {
  value = arcane_swarm_stack.demo.services
}

output "stack_compose_content" {
  value = data.arcane_swarm_stack.demo.compose_content
}
