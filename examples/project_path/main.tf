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

resource "arcane_project_path" "demo_from_path" {
  environment_id = var.environment_id
  name           = "demo-from-path"
  compose_path   = "${path.module}/demo-compose.yml"
  # env_path        = "${path.module}/demo.env"
  content_hash_mode = true
  running           = true
}

