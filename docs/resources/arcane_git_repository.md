# arcane_git_repository

Manages Git repository configurations for GitOps workflows.

## Example Usage

### SSH Authentication

```hcl
resource "arcane_git_repository" "my_repo" {
  name        = "My Application Repo"
  url         = "git@github.com:user/repo.git"
  auth_type   = "ssh"
  description = "Main application repository"
  enabled     = true

  ssh_key = file("~/.ssh/id_rsa")
}
```

### Token Authentication

```hcl
resource "arcane_git_repository" "public_repo" {
  name      = "Public Repo"
  url       = "https://github.com/user/public-repo.git"
  auth_type = "token"
  enabled   = true

  username = "github-user"
  token    = var.github_token
}
```

### No Authentication (Public)

```hcl
resource "arcane_git_repository" "public_repo" {
  name      = "Public Repo"
  url       = "https://github.com/user/public-repo.git"
  auth_type = "none"
  enabled   = true
}
```

## Argument Reference

- `name` (String, Required) — Repository name
- `url` (String, Required) — Git repository URL
- `auth_type` (String, Required) — Authentication type: `ssh`, `token`, or `none`
- `description` (String, Optional) — Repository description
- `enabled` (Bool, Optional) — Whether the repository is enabled
- `ssh_key` (String, Optional, Sensitive) — SSH private key for authentication (required when auth_type is `ssh`)
- `token` (String, Optional, Sensitive) — Access token for HTTP/HTTPS authentication (required when auth_type is `token`)
- `username` (String, Optional) — Username for authentication (used with token auth)

## Attributes Reference

- `id` (String) — Git repository ID
- `created_at` (String) — Creation timestamp
- `updated_at` (String) — Last update timestamp
