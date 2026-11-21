# arcane_network

Manages a Docker network for container communication.

## Example Usage

```hcl
resource "arcane_network" "frontend" {
  environment_id = var.environment_id
  name           = "frontend-network"
  driver         = "bridge"
  attachable     = true
  labels = {
    "tier" = "frontend"
  }
}

# Internal network (no external access)
resource "arcane_network" "backend" {
  environment_id = var.environment_id
  name           = "backend-network"
  driver         = "bridge"
  internal       = true
  attachable     = true
}

# Network with IPv6
resource "arcane_network" "ipv6" {
  environment_id = var.environment_id
  name           = "dual-stack"
  enable_ipv6    = true
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID. Changing this forces a new resource.
- `name` (String) - Name of the network. Changing this forces a new resource.

### Optional

- `driver` (String) - Network driver (e.g., bridge, overlay, host, macvlan). Defaults to 'bridge'. Changing this forces a new resource.
- `attachable` (Boolean) - Allow manual container attachment.
- `internal` (Boolean) - Restrict external access to the network.
- `enable_ipv6` (Boolean) - Enable IPv6 networking.
- `check_duplicate` (Boolean) - Check for duplicate network names.
- `ingress` (Boolean) - Enable routing-mesh for swarm cluster.
- `labels` (Map of String) - User-defined labels for metadata.
- `options` (Map of String) - Driver-specific options.
- `ipam` (Object) - IP address management. Changing this forces a new resource.
  - `driver` (String) - IPAM driver.
  - `options` (Map of String) - IPAM driver options.
  - `config` (List of Object) - Address pools, each with `subnet`, `ip_range`, `gateway` (String) and `aux_address` (Map of String).

```hcl
resource "arcane_network" "static" {
  environment_id = "0"
  name           = "static-net"
  ipam = {
    config = [{
      subnet  = "172.28.0.0/16"
      gateway = "172.28.0.1"
    }]
  }
}
```

## Attributes Reference

- `id` (String) - Unique identifier of the network.
- `scope` (String) - Scope of the network (local or swarm).
- `created` (String) - Creation timestamp.

## Import

Import using the format `environment_id/network_id`:

```
terraform import arcane_network.frontend <environment_id>/<network_id>
```

## Notes

Networks cannot be updated in place. Any changes to the network configuration will force the creation of a new network.
