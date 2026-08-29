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

# Create a GitOps sync with a pre-deploy lifecycle hook that decrypts
# sops/age-encrypted secrets before every deploy
resource "arcane_gitops_sync" "encrypted_sync" {
  environment_id = arcane_environment.production.id
  name           = "Encrypted App Deployment"
  repository_id  = arcane_git_repository.app_repo.id
  branch         = "main"
  compose_path   = "docker-compose.yml"
  project_name   = "my-encrypted-app"

  auto_sync     = true
  sync_interval = 300

  # Run pre-deploy.sh from the synced repo in a sops container before each
  # deploy. The age key is mounted read-only from the host; the runner gets
  # no network access ("none" is also the server default) and 120 seconds.
  pre_deploy_script_path  = "pre-deploy.sh"
  pre_deploy_runner_image = "ghcr.io/getsops/sops:v3.11.0"
  pre_deploy_env          = "SOPS_AGE_KEY_FILE=/run/secrets/age.key"
  pre_deploy_extra_mounts = "/opt/arcane/secrets/age.key:/run/secrets/age.key:ro"
  pre_deploy_network_mode = "none"
  pre_deploy_timeout_sec  = 120
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
