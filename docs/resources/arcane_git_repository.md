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

  ssh_key_wo         = file("~/.ssh/id_rsa")
  ssh_key_wo_version = 1
}
```

### Token Authentication

```hcl
resource "arcane_git_repository" "public_repo" {
  name      = "Public Repo"
  url       = "https://github.com/user/public-repo.git"
  auth_type = "token"
  enabled   = true

  username         = "github-user"
  token_wo         = var.github_token
  token_wo_version = 1
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
- `ssh_host_key_verification` (String, Optional) — SSH host key verification mode
- `ssh_key` (String, Optional, Sensitive) — SSH private key for authentication (required when auth_type is `ssh`). **Deprecated**: stored in state; use `ssh_key_wo` instead, `ssh_key` will be removed in the next major release.
- `ssh_key_wo` (String, Optional, Sensitive, Write-only) — SSH private key for authentication. Never stored in state or plan files; only sent when `ssh_key_wo_version` changes. Conflicts with `ssh_key`. See [write-only arguments](../index.md#write-only-arguments).
- `ssh_key_wo_version` (Number, Optional) — change it to send a new `ssh_key_wo`. Required with `ssh_key_wo`.
- `token` (String, Optional, Sensitive) — Access token for HTTP/HTTPS authentication (required when auth_type is `token`). **Deprecated**: stored in state; use `token_wo` instead, `token` will be removed in the next major release.
- `token_wo` (String, Optional, Sensitive, Write-only) — Access token for HTTP/HTTPS authentication. Never stored in state or plan files; only sent when `token_wo_version` changes. Conflicts with `token`. See [write-only arguments](../index.md#write-only-arguments).
- `token_wo_version` (Number, Optional) — change it to send a new `token_wo`. Required with `token_wo`.
- `username` (String, Optional) — Username for authentication (used with token auth)

## Attributes Reference

- `id` (String) — Git repository ID
- `created_at` (String) — Creation timestamp
- `updated_at` (String) — Last update timestamp
