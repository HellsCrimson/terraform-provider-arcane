# arcane_environment_mtls_file

Downloads an individual generated mTLS certificate asset for an edge environment.

## Example Usage

```hcl
data "arcane_environment_mtls_file" "cert" {
  environment_id = var.environment_id
  file_name      = "client.crt"
}
```

## Argument Reference

- `environment_id` (String, Required) - Environment ID.
- `file_name` (String, Required) - mTLS asset filename.

## Attributes Reference

- `id` (String) - Data source ID.
- `content` (String, Sensitive) - File content.
- `content_base64` (String, Sensitive) - Base64-encoded file content.
