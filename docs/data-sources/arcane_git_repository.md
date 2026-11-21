# arcane_git_repository

Reads an Arcane git repository configuration.

## Example Usage

```hcl
data "arcane_git_repository" "app_repo" {
  id = "repo-123456"
}

output "repository_url" {
  value = data.arcane_git_repository.app_repo.url
}
```

## Argument Reference

- `id` (String, Required) — git repository ID.

## Attributes Reference

- `name` (String) — repository name.
- `url` (String) — git repository URL.
- `auth_type` (String) — authentication type (e.g., ssh, token, none).
- `description` (String) — repository description.
- `enabled` (Bool) — whether the repository is enabled.
- `username` (String) — username for authentication.
- `git_provider` (String) — git forge used to build web links.
- `commit_author_name`, `commit_author_email` (String) — author of the commits Arcane pushes.
- `has_signing_key` (Bool) — whether Arcane holds a signing key for the repository.
- `created_at` (String) — creation timestamp.
- `updated_at` (String) — last update timestamp.

**Note:** Sensitive credentials (SSH keys, tokens) are never exposed in data sources for security reasons.
