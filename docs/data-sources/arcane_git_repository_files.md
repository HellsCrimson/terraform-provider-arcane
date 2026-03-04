# arcane_git_repository_files

Browses files in a Git repository.

## Example Usage

```hcl
data "arcane_git_repository_files" "repo_files" {
  repository_id = "repo-123"
  branch        = "main"
  path          = "deploy"
}

output "files_json" {
  value = data.arcane_git_repository_files.repo_files.files_json
}
```

## Argument Reference

- `repository_id` (String, Required) - repository ID.
- `branch` (String, Optional) - branch to browse.
- `path` (String, Optional) - path within the repository.

## Attributes Reference

- `current_path` (String) - resolved browse path.
- `files_json` (String) - file tree payload as JSON.
