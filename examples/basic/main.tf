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
  # Alternatively set ARCANE_API_KEY env var
  api_key = ""
  # endpoint defaults to http://localhost:3552/api
  endpoint = "http://localhost:3552/api"
}

resource "arcane_user" "example" {
  username     = "johndoe"
  password     = "SuperSecret123!" # required on create
  display_name = "John Doe"
  email        = "john@example.com"
  locale       = "en-US"
  roles        = ["user"]
}

# Set your target environment ID here for the following resources
variable "environment_id" {
  type = string
}

resource "arcane_settings" "env" {
  environment_id                = var.environment_id
  base_server_url               = "http://localhost:3552"
  polling_enabled               = "true"
  polling_interval              = "10s"
  sidebar_hover_expansion       = "true"
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
}

# Ensure the project is started (compose up); remove to avoid lifecycle management
resource "arcane_project_state" "demo_running" {
  environment_id = var.environment_id
  project_id     = arcane_project.demo.id
  running        = true
}

# Same as arcane_project but reads compose/env files from the filesystem
resource "arcane_project_path" "demo_from_path" {
  environment_id = var.environment_id
  name           = "demo-from-path"
  compose_path   = "${path.module}/demo-compose.yml"
  # env_path     = "${path.module}/demo.env"
  # When enabled, state stores only hashes to detect changes
  content_hash_mode = true
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

resource "arcane_container" "alpine" {
  environment_id = var.environment_id
  name           = "hello-alpine"
  image          = "alpine:latest"
  command        = ["sh", "-c", "sleep 3600"]
  # Ports map: container_port => host_port (numeric strings)
  ports          = { "8081" = "8081" }
  force_delete   = true
  remove_volumes = true
}
