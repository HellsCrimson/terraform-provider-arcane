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

# Example with SSH authentication
resource "arcane_git_repository" "ssh_repo" {
  name        = "SSH Repository"
  url         = "git@github.com:user/my-app.git"
  auth_type   = "ssh"
  description = "Private repository with SSH key"
  enabled     = true

  ssh_key = file("~/.ssh/id_rsa")
}

# Example with token authentication
resource "arcane_git_repository" "token_repo" {
  name        = "Token Repository"
  url         = "https://github.com/user/public-repo.git"
  auth_type   = "token"
  description = "Repository with token authentication"
  enabled     = true

  username = "github-user"
  token    = var.github_token
}

# Example with no authentication (public repo)
resource "arcane_git_repository" "public_repo" {
  name        = "Public Repository"
  url         = "https://github.com/user/public-repo.git"
  auth_type   = "none"
  description = "Public repository"
  enabled     = true
}
