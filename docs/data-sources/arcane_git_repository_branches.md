# arcane_git_repository_branches

Lists branches for a Git repository.

## Example Usage

```hcl
data "arcane_git_repository_branches" "repo" {
  repository_id = "repo-123"
}

output "default_branch" {
  value = data.arcane_git_repository_branches.repo.default_branch
}
```

## Argument Reference

- `repository_id` (String, Required) - repository ID.

## Attributes Reference

- `branches` (List of String) - branch names.
- `default_branch` (String) - detected default branch.
