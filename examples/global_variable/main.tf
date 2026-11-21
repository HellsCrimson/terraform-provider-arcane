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

variable "smtp_password" {
  type      = string
  sensitive = true
}

# Plain variable, available in every environment
resource "arcane_global_variable" "domain" {
  key   = "BASE_DOMAIN"
  value = "example.com"
}

# Secret variable for the local environment, injected into its containers.
# The value is write-only: bump value_wo_version to rotate it.
resource "arcane_global_variable" "smtp_password" {
  key                    = "SMTP_PASSWORD"
  value_wo               = var.smtp_password
  value_wo_version       = 1
  environment_ids        = ["0"]
  inject_into_containers = true
}
