# arcane_environment_mtls_bundle

Downloads the generated mTLS client certificate bundle for an edge environment.

## Example Usage

```hcl
data "arcane_environment_mtls_bundle" "bundle" {
  environment_id = var.environment_id
}
```

## Argument Reference

- `environment_id` (String, Required) - Environment ID.

## Attributes Reference

- `id` (String) - Data source ID.
- `content_base64` (String, Sensitive) - Base64-encoded bundle content.
