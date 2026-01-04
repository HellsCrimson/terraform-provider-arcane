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
  # Or set ARCANE_API_KEY env var
  api_key = var.arcane_api_key
  # Defaults to http://localhost:3552/api
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

resource "arcane_user" "example" {
  username     = "johndoe"
  password     = "SuperSecret123!"
  display_name = "John Doe"
  email        = "john@example.com"
  locale       = "en-US"
  roles        = ["user"]
}

