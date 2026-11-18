# arcane_container_registry

Manages a container registry in Arcane.

## Example Usage

```hcl
resource "arcane_container_registry" "example" {
  url              = "https://ghcr.io"
  username         = "bot"
  token_wo         = var.ghcr_token
  token_wo_version = 1
  description      = "GitHub Container Registry"
  insecure         = false
  enabled          = true

  repository_names = ["my-org", "my-org/platform"]
}
```

## Argument Reference

- `url` (String, Required)
- `username` (String, Required)
- `token_wo` (String, Required, Sensitive, Write-only) — registry access token or password. Never stored in state or plan files; only sent when `token_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `token_wo_version` (Number, Optional) — change it to send a new `token_wo`. Required with `token_wo`.
- `description` (String, Optional)
- `insecure` (Bool, Optional)
- `enabled` (Bool, Optional)
- `registry_type` (String, Optional) - Registry implementation type. Defaults to `generic`; use `ecr` for AWS ECR.
- `repository_names` (List of String, Optional) - Pre-configured repository namespaces offered when pushing images to this registry. Terraform owns this list: leaving it out of the configuration clears any names set elsewhere (for example in the Arcane UI) on the next update. Arcane trims and de-duplicates the entries, so values that need normalizing will show up as a diff on the next plan.
- `aws_access_key_id_wo` (String, Optional, Sensitive, Write-only) — AWS access key ID (ECR registries). Never stored in state or plan files; only sent when `aws_access_key_id_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `aws_access_key_id_wo_version` (Number, Optional) — change it to send a new `aws_access_key_id_wo`. Required with `aws_access_key_id_wo`.
- `aws_secret_access_key_wo` (String, Optional, Sensitive, Write-only) — AWS secret access key (ECR registries). Never stored in state or plan files; only sent when `aws_secret_access_key_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `aws_secret_access_key_wo_version` (Number, Optional) — change it to send a new `aws_secret_access_key_wo`. Required with `aws_secret_access_key_wo`.
- `aws_region` (String, Optional)

## Attributes Reference

- `id` (String)
- `created_at` (String)
- `updated_at` (String)
