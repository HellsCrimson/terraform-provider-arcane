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

# Create an API key for CI/CD automation
resource "arcane_api_key" "ci_cd" {
  name        = "CI/CD Pipeline"
  description = "API key for GitHub Actions CI/CD pipeline"
  expires_at  = "2025-12-31T23:59:59Z" # Optional expiration
}

# Create an API key without expiration
resource "arcane_api_key" "monitoring" {
  name        = "Monitoring Service"
  description = "API key for Prometheus monitoring integration"
  # No expires_at means it doesn't expire
}

# Output the API key (only available on creation)
output "ci_cd_api_key" {
  value       = arcane_api_key.ci_cd.key
  sensitive   = true
  description = "The API key secret - save this securely, it won't be shown again!"
}

output "ci_cd_key_prefix" {
  value       = arcane_api_key.ci_cd.key_prefix
  description = "The key prefix for identification"
}

output "ci_cd_key_id" {
  value = arcane_api_key.ci_cd.id
}

output "monitoring_api_key" {
  value       = arcane_api_key.monitoring.key
  sensitive   = true
  description = "The monitoring API key secret"
}

# Note: After initial creation, the full key is not retrievable from the API
# Make sure to save the output or use terraform output to retrieve it
# Example: terraform output -raw ci_cd_api_key
