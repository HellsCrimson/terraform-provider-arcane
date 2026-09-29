terraform {
  required_version = ">= 1.11.0"
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

# Example with SSH authentication
resource "arcane_git_repository" "ssh_repo" {
  name        = "SSH Repository"
  url         = "git@github.com:user/my-app.git"
  auth_type   = "ssh"
  description = "Private repository with SSH key"
  enabled     = true

  # Write-only: never stored in state. Change ssh_key_wo_version to send a new one.
  ssh_key_wo         = file("~/.ssh/id_rsa")
  ssh_key_wo_version = 1
}

# Example with token authentication
resource "arcane_git_repository" "token_repo" {
  name        = "Token Repository"
  url         = "https://github.com/user/public-repo.git"
  auth_type   = "token"
  description = "Repository with token authentication"
  enabled     = true

  username = "github-user"
  # Write-only: never stored in state. Change token_wo_version to send a new one.
  token_wo         = var.github_token
  token_wo_version = 1
}

# Example with no authentication (public repo)
resource "arcane_git_repository" "public_repo" {
  name        = "Public Repository"
  url         = "https://github.com/user/public-repo.git"
  auth_type   = "none"
  description = "Public repository"
  enabled     = true
}
