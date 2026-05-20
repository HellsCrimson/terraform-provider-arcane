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

resource "arcane_notification" "example" {
  environment_id = var.environment_id
  provider_name  = "discord"
  enabled        = true
  config = {
    avatarUrl = ""
    events = {
      container_update    = true
      image_update        = false
      prune_report        = true
      vulnerability_found = false
    }
    token     = "some token"
    username  = "User"
    webhookId = "id"
  }
}
