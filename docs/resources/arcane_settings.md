# arcane_settings

Updates Arcane environment settings using explicit fields.

## Example Usage

```hcl
resource "arcane_settings" "env" {
  environment_id          = var.environment_id
  base_server_url         = "http://localhost:3552"
  polling_enabled         = "true"
  polling_interval        = "10s"
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID.

### Optional

All optional attributes are strings:

**General Settings**
- `accent_color` - UI accent color.
- `base_server_url` - Base URL for the server.
- `default_shell` - Default shell for terminal sessions.
- `disk_usage_path` - Path for disk usage monitoring.
- `projects_directory` - Directory for compose projects.
- `enable_gravatar` - Enable Gravatar for user avatars.
- `keyboard_shortcuts_enabled` - Enable keyboard shortcuts.
- `max_image_upload_size` - Maximum image upload size.

**Docker Settings**
- `docker_host` - Docker host socket/URL.
- `docker_api_timeout` - Docker API timeout.
- `docker_image_pull_timeout` - Timeout for pulling images.
- `docker_prune_mode` - Docker prune mode.

**Polling Settings**
- `polling_enabled` - Enable polling.
- `polling_interval` - Polling interval.

**Auto Update Settings**
- `auto_update` - Enable auto updates.
- `auto_update_interval` - Auto update check interval.
- `auto_update_excluded_containers` - Excluded containers for auto updates.
- `auto_inject_env` - Auto inject environment variables.

**Auto Heal Settings**
- `auto_heal_enabled` - Enable auto heal.
- `auto_heal_excluded_containers` - Excluded containers for auto heal.
- `auto_heal_interval` - Auto heal interval.
- `auto_heal_max_restarts` - Maximum auto heal restarts.
- `auto_heal_restart_window` - Auto heal restart window.

**Scheduled Prune Settings**
- `scheduled_prune_enabled` - Enable scheduled pruning.
- `scheduled_prune_interval` - Prune interval.
- `scheduled_prune_build_cache` - Prune build cache.
- `scheduled_prune_containers` - Prune containers.
- `scheduled_prune_images` - Prune images.
- `scheduled_prune_networks` - Prune networks.
- `scheduled_prune_volumes` - Prune volumes.

**Authentication Settings**
- `auth_local_enabled` - Enable local authentication.
- `auth_oidc_config` - OIDC configuration.
- `auth_password_policy` - Password policy.
- `auth_session_timeout` - Session timeout.

**OIDC Settings**
- `oidc_enabled` - Enable OIDC authentication.
- `oidc_issuer_url` - OIDC issuer URL.
- `oidc_client_id` - OIDC client ID.
- `oidc_client_secret` - OIDC client secret.
- `oidc_scopes` - OIDC scopes.
- `oidc_admin_claim` - OIDC admin claim.
- `oidc_admin_value` - OIDC admin value.
- `oidc_auto_redirect_to_provider` - Auto redirect to OIDC provider.
- `oidc_merge_accounts` - Merge OIDC accounts.
- `oidc_provider_name` - OIDC provider display name.
- `oidc_provider_logo_url` - OIDC provider logo URL.
- `oidc_skip_tls_verify` - Skip TLS verification for OIDC.

**Build Settings**
- `build_provider` - Build provider.
- `build_timeout` - Build timeout.
- `builds_directory` - Builds directory.
- `default_deploy_pull_policy` - Default deploy pull policy.
- `depot_project_id` - Depot project ID.
- `depot_token` - Depot token.

**Timeout Settings**
- `environment_health_interval` - Environment health check interval.
- `git_operation_timeout` - Git operation timeout.
- `http_client_timeout` - HTTP client timeout.
- `proxy_request_timeout` - Proxy request timeout.
- `registry_timeout` - Registry timeout.

**UI Settings**
- `mobile_navigation_mode` - Mobile navigation mode.
- `mobile_navigation_show_labels` - Show labels in mobile navigation.
- `sidebar_hover_expansion` - Enable sidebar hover expansion.
- `oled_mode` - OLED mode.

**Vulnerability Scan Settings**
- `trivy_concurrent_scan_containers` - Trivy concurrent scan containers.
- `trivy_cpu_limit` - Trivy CPU limit.
- `trivy_image` - Trivy image.
- `trivy_memory_limit_mb` - Trivy memory limit in MB.
- `trivy_network` - Trivy network.
- `trivy_resource_limits_enabled` - Enable Trivy resource limits.
- `trivy_scan_timeout` - Trivy scan timeout.
- `vulnerability_scan_enabled` - Enable vulnerability scanning.
- `vulnerability_scan_interval` - Vulnerability scan interval.

## Attributes Reference

- `id` (String) - Same as `environment_id`.
- `applied` (Map of String) - Server values after apply, showing all current settings.

## Import

Import using the environment ID:

```
terraform import arcane_settings.env <environment_id>
```
