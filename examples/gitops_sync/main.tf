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

variable "github_token" {
  type      = string
  sensitive = true
}

# Create an environment
resource "arcane_environment" "production" {
  name        = "Production"
  api_url     = "http://agent-host:8080"
  enabled     = true
  use_api_key = true
}

# Create a git repository
resource "arcane_git_repository" "app_repo" {
  name      = "Application Repository"
  url       = "https://github.com/user/my-app.git"
  auth_type = "token"
  username  = "github-user"
  token     = var.github_token
  enabled   = true
}

# Create a GitOps sync for production
resource "arcane_gitops_sync" "production_sync" {
  environment_id = arcane_environment.production.id
  name           = "Production Deployment"
  repository_id  = arcane_git_repository.app_repo.id
  branch         = "main"
  compose_path   = "docker-compose.prod.yml"
  project_name   = "my-app-prod"

  auto_sync     = true
  sync_interval = 300 # Sync every 5 minutes
  start_project = true # Start the project after creation (default: true)

  # Environment variables for the deployed project
  environment_variables = {
    DATABASE_URL = "postgresql://user:pass@db:5432/prod"
    REDIS_URL    = "redis://redis:6379"
    APP_ENV      = "production"
    LOG_LEVEL    = "info"
  }
}

# Create a GitOps sync for a specific feature (don't start automatically)
resource "arcane_gitops_sync" "feature_sync" {
  environment_id = arcane_environment.production.id
  name           = "Feature Branch Deployment"
  repository_id  = arcane_git_repository.app_repo.id
  branch         = "feature/new-feature"
  compose_path   = "docker-compose.yml"
  project_name   = "my-app-feature"

  auto_sync     = false
  start_project = false # Don't start the project automatically
  # Note: 'enabled' is a computed (read-only) field that shows the sync status
}

# Output sync information
output "production_sync_id" {
  value = arcane_gitops_sync.production_sync.id
}

output "production_project_id" {
  value       = arcane_gitops_sync.production_sync.project_id
  description = "The project ID created by the GitOps sync"
}
