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

resource "arcane_role" "deployer" {
  name        = "compose-deployer"
  permissions = ["containers:start", "containers:stop", "containers:view"]
}

resource "arcane_oidc_role_mapping" "platform_admins" {
  claim_value = "platform-admins"
  role_id     = arcane_role.deployer.id
}

resource "arcane_oidc_role_mapping" "staging_viewers" {
  claim_value    = "staging-viewers"
  role_id        = arcane_role.deployer.id
  environment_id = var.environment_id
}

output "platform_admins_mapping_id" {
  value = arcane_oidc_role_mapping.platform_admins.id
}
