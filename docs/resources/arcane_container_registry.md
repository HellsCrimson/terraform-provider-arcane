# arcane_container_registry

Manages a container registry in Arcane.

## Example Usage

```hcl
resource "arcane_container_registry" "example" {
  url         = "https://ghcr.io"
  username    = "bot"
  token       = var.ghcr_token
  description = "GitHub Container Registry"
  insecure    = false
  enabled     = true

  repository_names = ["my-org", "my-org/platform"]
}
```

## Argument Reference

- `url` (String, Required)
- `username` (String, Required)
- `token` (String, Required, Sensitive)
- `description` (String, Optional)
- `insecure` (Bool, Optional)
- `enabled` (Bool, Optional)
- `registry_type` (String, Optional) - Registry implementation type. Defaults to `generic`; use `ecr` for AWS ECR.
- `repository_names` (List of String, Optional) - Pre-configured repository namespaces offered when pushing images to this registry. Terraform owns this list: leaving it out of the configuration clears any names set elsewhere (for example in the Arcane UI) on the next update. Arcane trims and de-duplicates the entries, so values that need normalizing will show up as a diff on the next plan.
- `aws_access_key_id` (String, Optional, Sensitive)
- `aws_secret_access_key` (String, Optional, Sensitive)
- `aws_region` (String, Optional)

## Attributes Reference

- `id` (String)
- `created_at` (String)
- `updated_at` (String)
