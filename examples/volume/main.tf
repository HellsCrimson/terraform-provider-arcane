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

# Create a volume for PostgreSQL data
resource "arcane_volume" "postgres_data" {
  environment_id = var.environment_id
  name           = "postgres-data"
  driver         = "local"

  labels = {
    "com.example.service" = "database"
    "com.example.backup"  = "daily"
  }
}

# Create a volume for application uploads
resource "arcane_volume" "app_uploads" {
  environment_id = var.environment_id
  name           = "app-uploads"
  driver         = "local"

  labels = {
    "com.example.service" = "application"
    "com.example.type"    = "user-content"
  }
}

# Use the volume in a container
resource "arcane_container" "postgres" {
  environment_id = var.environment_id
  name           = "postgres"
  image          = "postgres:14"

  environment = [
    "POSTGRES_DB=myapp",
    "POSTGRES_USER=myuser",
    "POSTGRES_PASSWORD=mypassword"
  ]

  volumes = [
    "${arcane_volume.postgres_data.name}:/var/lib/postgresql/data"
  ]

  restart_policy = "unless-stopped"
}

output "postgres_volume_id" {
  value = arcane_volume.postgres_data.id
}

output "postgres_mountpoint" {
  value = arcane_volume.postgres_data.mountpoint
}

output "volume_in_use" {
  value = arcane_volume.postgres_data.in_use
}

output "volume_size_bytes" {
  value = arcane_volume.postgres_data.size
}
