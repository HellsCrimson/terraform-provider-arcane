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

resource "arcane_project" "demo" {
  environment_id  = var.environment_id
  name            = "demo"
  compose_content = <<YAML
version: "3.9"
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
YAML
  running         = true
}

