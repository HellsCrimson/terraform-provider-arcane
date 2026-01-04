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
  provider_name  = "slack"
  enabled        = true
  config = {
    webhook = "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    channel = "#deploys"
  }
}

