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

data "arcane_role_permissions" "all" {}

resource "arcane_role" "deployer" {
  name        = "compose-deployer"
  description = "Can start, stop and view containers"
  permissions = [
    "containers:start",
    "containers:stop",
    "containers:view",
  ]
}

data "arcane_role" "admin" {
  name = "Admin"
}

output "deployer_role_id" {
  value = arcane_role.deployer.id
}

output "admin_role_id" {
  value = data.arcane_role.admin.id
}

output "all_permissions" {
  value = data.arcane_role_permissions.all.all_permissions
}
