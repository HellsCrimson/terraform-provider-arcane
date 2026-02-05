# arcane_network

Reads an Arcane Docker network configuration.

## Example Usage

```hcl
data "arcane_network" "backend" {
  environment_id = "env-123456"
  id             = "network-789"
}

output "network_driver" {
  value = data.arcane_network.backend.driver
}

output "network_scope" {
  value = data.arcane_network.backend.scope
}
```

## Argument Reference

- `environment_id` (String, Required) — environment ID.
- `id` (String, Required) — network ID.

## Attributes Reference

- `name` (String) — network name.
- `driver` (String) — network driver.
- `attachable` (Bool) — allow manual container attachment.
- `internal` (Bool) — restrict external access.
- `enable_ipv4` (Bool) — IPv4 enabled.
- `enable_ipv6` (Bool) — IPv6 enabled.
- `scope` (String) — network scope.
- `created` (String) — creation timestamp.
- `labels` (Map of String) — network labels.
- `options` (Map of String) — network options.
