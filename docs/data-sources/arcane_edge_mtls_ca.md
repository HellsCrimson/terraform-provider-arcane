# arcane_edge_mtls_ca

Downloads the Arcane-managed edge mTLS certificate authority.

## Example Usage

```hcl
data "arcane_edge_mtls_ca" "ca" {}
```

## Attributes Reference

- `id` (String) - Static data source ID.
- `content` (String) - CA file content.
- `content_base64` (String) - Base64-encoded CA file content.
