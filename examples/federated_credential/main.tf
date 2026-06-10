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

resource "arcane_role" "deployer" {
  name        = "compose-deployer"
  permissions = ["containers:start", "containers:stop", "containers:view"]
}

resource "arcane_federated_credential" "github_actions" {
  name          = "github-actions-deploy"
  enabled       = true
  issuer_url    = "https://token.actions.githubusercontent.com"
  audiences     = ["https://arcane.example.com"]
  subject_match = "repo:my-org/my-repo:ref:refs/heads/main"
  role_id       = arcane_role.deployer.id

  match_type        = "exact"
  subject_claim     = "sub"
  token_ttl_seconds = 900
}

output "service_username" {
  value = arcane_federated_credential.github_actions.service_username
}
