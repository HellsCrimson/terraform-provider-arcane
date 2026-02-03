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

# Add a public template registry
resource "arcane_template_registry" "awesome_compose" {
  name        = "Awesome Compose"
  url         = "https://github.com/docker/awesome-compose"
  description = "Official Docker Awesome Compose templates"
  enabled     = true
}

# Add a company internal registry
resource "arcane_template_registry" "internal" {
  name        = "Internal Templates"
  url         = "https://git.company.com/devops/compose-templates"
  description = "Company internal compose templates"
  enabled     = true
}

# Add a community registry (disabled by default)
resource "arcane_template_registry" "community" {
  name        = "Community Templates"
  url         = "https://github.com/community/templates"
  description = "Community-contributed templates"
  enabled     = false # Enable when needed
}

output "awesome_compose_id" {
  value = arcane_template_registry.awesome_compose.id
}

output "registries" {
  value = {
    awesome_compose = arcane_template_registry.awesome_compose.id
    internal        = arcane_template_registry.internal.id
    community       = arcane_template_registry.community.id
  }
}
