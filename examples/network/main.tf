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

# Create an application network
resource "arcane_network" "app_network" {
  environment_id = var.environment_id
  name           = "app-network"
  driver         = "bridge"
  internal       = false

  labels = {
    "com.example.network" = "application"
    "com.example.tier"    = "backend"
  }
}

# Create an internal database network
resource "arcane_network" "db_network" {
  environment_id = var.environment_id
  name           = "db-network"
  driver         = "bridge"
  internal       = true # No external access

  labels = {
    "com.example.network" = "database"
    "com.example.tier"    = "data"
  }
}

# Frontend container
resource "arcane_container" "frontend" {
  environment_id = var.environment_id
  name           = "frontend"
  image          = "nginx:latest"

  networks = [arcane_network.app_network.name]

  ports = {
    "80" = "8080"
  }

  labels = {
    "com.example.service" = "frontend"
  }
}

# Backend container (connected to both networks)
resource "arcane_container" "backend" {
  environment_id = var.environment_id
  name           = "backend"
  image          = "node:18"

  networks = [
    arcane_network.app_network.name,
    arcane_network.db_network.name
  ]

  labels = {
    "com.example.service" = "backend"
  }
}

# Database container (internal network only)
resource "arcane_container" "database" {
  environment_id = var.environment_id
  name           = "database"
  image          = "postgres:14"

  networks = [arcane_network.db_network.name]

  environment = [
    "POSTGRES_DB=myapp",
    "POSTGRES_USER=myuser",
    "POSTGRES_PASSWORD=mypassword"
  ]

  labels = {
    "com.example.service" = "database"
  }
}

output "app_network_id" {
  value = arcane_network.app_network.id
}

output "db_network_id" {
  value = arcane_network.db_network.id
}

output "network_scope" {
  value = arcane_network.app_network.scope
}
