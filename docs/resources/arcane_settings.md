# arcane_settings

Updates Arcane settings using explicit fields.

Arcane settings live in three scopes, each with its own endpoint: **environment** settings
(per environment), **app-wide** settings (`/settings`) and **authentication/OIDC** settings
(`/settings/authentication`). The provider routes each attribute to the right scope automatically.
App-wide and authentication settings (for example `apns_enabled`, `experimental_features_enabled`,
`avatar_max_upload_size_mb`, `auth_*` and `oidc_*`) are **global**: setting them through any
`arcane_settings` resource changes them for every environment, so manage them from a single resource.

## Example Usage

```hcl
resource "arcane_settings" "env" {
  environment_id          = var.environment_id
  base_server_url         = "http://localhost:3552"
  polling_enabled         = "true"
  polling_interval        = "*/10 * * * *"
}
```

## Argument Reference

### Required

- `environment_id` (String) - Environment ID.

### Optional

All optional attributes are strings:

**General Settings**
- `base_server_url` - Base URL for the server.
- `default_shell` - Default shell for terminal sessions.
- `disk_usage_path` - Path for disk usage monitoring.
- `projects_directory` - Directory for compose projects.
- `swarm_stack_sources_directory` - Directory for swarm stack sources.
- `templates_directory` - Directory for templates.
- `enable_gravatar` - Enable Gravatar for user avatars.
- `max_image_upload_size` - Maximum image upload size.
- `avatar_max_upload_size_mb` - Maximum avatar upload size, in MB.
- `experimental_features_enabled` - Enable experimental features.
- `apns_enabled` - Enable Apple push notifications.
- `max_concurrent_activities` - Maximum number of activities run concurrently.
- `tools_image_registry` - Registry used to pull Arcane's helper tool images.
- `update_check_registry` - Registry queried for update checks.
- `follow_project_symlinks` - Follow symlinks when scanning project directories.

**Activity History Settings**
- `activity_history_max_entries` - Maximum number of activity history entries to retain.
- `activity_history_retention_days` - Activity history retention, in days.

**GitOps Sync Settings**
- `git_sync_max_files` - Maximum number of files to sync.
- `git_sync_max_binary_size_mb` - Maximum binary file size to sync, in MB.
- `git_sync_max_total_size_mb` - Maximum total sync size, in MB.

**Lifecycle Hook Settings**
- `lifecycle_enabled` - Gates whether GitOps syncs may configure pre-deploy lifecycle scripts (`"true"`/`"false"`, disabled by default since scripts are repo-trusted code that runs on every deploy).
- `lifecycle_default_runner_image` - Default container image used to run pre-deploy lifecycle scripts when a sync does not set `pre_deploy_runner_image`.
- `lifecycle_max_timeout_sec` - Caps the per-sync `pre_deploy_timeout_sec` admins can configure. `"0"` disables the cap.

**Docker Settings**
- `docker_host` - Docker host socket/URL.
- `docker_api_timeout` - Docker API timeout.
- `docker_client_refresh_interval` - Docker client refresh interval.
- `docker_image_pull_timeout` - Timeout for pulling images.

**Polling Settings**
- `polling_enabled` - Enable polling.
- `polling_interval` - Polling interval (5-field cron expression).

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
- `prune_build_cache_mode`, `prune_build_cache_until`
- `prune_container_mode`, `prune_container_until`
- `prune_image_mode`, `prune_image_until`
- `prune_network_mode`, `prune_network_until`
- `prune_volume_mode`

**Authentication Settings**
- `auth_local_enabled` - Enable local authentication.
- `auth_password_min_length` - Minimum password length.
- `auth_password_require_lowercase`, `auth_password_require_uppercase`, `auth_password_require_number`, `auth_password_require_symbol` - Password complexity requirements (`"true"`/`"false"`).
- `auth_session_timeout` - Session timeout.

**OIDC Settings**
- `oidc_enabled` - Enable OIDC authentication.
- `oidc_issuer_url` - OIDC issuer URL.
- `oidc_client_id` - OIDC client ID.
- `oidc_client_secret_wo` - OIDC client secret. Sensitive, write-only: never stored in state, only sent when `oidc_client_secret_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `oidc_client_secret_wo_version` (Number) - Change it to send a new `oidc_client_secret_wo`. Required with `oidc_client_secret_wo`.
- `oidc_scopes` - OIDC scopes.
- `oidc_auto_redirect_to_provider` - Auto redirect to OIDC provider.
- `oidc_merge_accounts` - Merge OIDC accounts.
- `oidc_provider_name` - OIDC provider display name.
- `oidc_provider_logo_url` - OIDC provider logo URL.
- `oidc_skip_tls_verify` - Skip TLS verification for OIDC.
- `oidc_groups_claim` - OIDC groups claim.

**Build Settings**
- `build_provider` - Build provider.
- `build_timeout` - Build timeout.
- `builds_directory` - Builds directory.
- `default_deploy_pull_policy` - Default deploy pull policy.
- `deploy_wait_timeout` - How long a deploy waits for services to become ready.
- `depot_project_id` - Depot project ID.
- `depot_token_wo` - Depot token. Sensitive, write-only: never stored in state, only sent when `depot_token_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `depot_token_wo_version` (Number) - Change it to send a new `depot_token_wo`. Required with `depot_token_wo`.

**Timeout Settings**
- `environment_health_interval` - Environment health check interval.
- `git_operation_timeout` - Git operation timeout.
- `http_client_timeout` - HTTP client timeout.
- `proxy_request_timeout` - Proxy request timeout.
- `registry_timeout` - Registry timeout.
- `volume_helper_idle_timeout` - Idle timeout for the volume browser helper.

**Image Patch Settings**
- `image_auto_patch_enabled` - Enable automatic image patching.
- `image_auto_patch_interval` - Automatic image patch interval.
- `image_event_watcher_enabled` - Watch Docker image events.
- `image_patch_all_platforms` - Patch all platforms of a multi-platform image.
- `image_patch_suffix` - Tag suffix applied to patched images.
- `image_patch_timeout_sec` - Image patch timeout, in seconds.

**Vulnerability Scan Settings**
- `trivy_concurrent_scan_containers` - Trivy concurrent scan containers.
- `trivy_cpu_limit` - Trivy CPU limit.
- `trivy_memory_limit_mb` - Trivy memory limit in MB.
- `trivy_network` - Trivy network.
- `trivy_privileged` - Run Trivy with privileged mode.
- `trivy_resource_limits_enabled` - Enable Trivy resource limits.
- `trivy_scan_timeout` - Trivy scan timeout.
- `trivy_security_opts` - Trivy security options.
- `trivy_db_registry` - Registry used to pull the Trivy vulnerability database.
- `trivy_ignore_unfixed` - Ignore vulnerabilities that have no fix available.
- `trivy_server_enabled` - Use an external Trivy server instead of running scans locally.
- `trivy_server_url` - External Trivy server URL.
- `trivy_server_token_wo` - External Trivy server token. Sensitive, write-only: never stored in state, only sent when `trivy_server_token_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `trivy_server_token_wo_version` (Number) - Change it to send a new `trivy_server_token_wo`. Required with `trivy_server_token_wo`.
- `vulnerability_scan_enabled` - Enable vulnerability scanning.
- `vulnerability_scan_interval` - Vulnerability scan interval.

**More Settings**
- `projects_directories` - Additional project directories, as a JSON array.
- `auto_update_delay` - Delay before an available update is applied.
- `event_cleanup_interval` - Event cleanup schedule (5-field cron expression).
- `expired_sessions_cleanup_interval` - Expired sessions cleanup schedule (5-field cron expression).
- `upgrade_log_retention_days` - Upgrade log retention, in days.
- `unmanaged_backup_retention_days` - Retention of backups not managed by a policy, in days.
- `project_scan_max_depth` - Maximum directory depth scanned for projects.
- `project_scan_skip_dirs` - Comma-separated directories skipped when scanning for projects.
- `project_workspace_max_depth` - Maximum depth of the project file browser.
- `project_workspace_max_entries` - Maximum entries listed by the project file browser.
- `project_workspace_max_file_size_mb` - Maximum file size opened by the project file browser, in MB.
- `volume_workspace_max_entries` - Maximum entries listed by the volume file browser.
- `volume_workspace_max_file_size_mb` - Maximum file size opened by the volume file browser, in MB.
- `gpu_monitoring_enabled` - Enable GPU monitoring.
- `gpu_type` - GPU type to monitor (`auto` by default).
- `build_history_retention_days` - Image build history retention, in days.
- `feature_vulnerability_management_enabled` - Enable the vulnerability management feature.
- `feature_swarm_enabled` - Enable the Docker Swarm feature.
- `vulnerability_threat_intel_enabled` - Enrich vulnerabilities with threat intelligence.
- `vulnerability_update_gate_enabled` - Block image updates that exceed the vulnerability limits below.
- `vulnerability_update_gate_max_critical` - Maximum critical vulnerabilities an update may have (`-1` for no limit).
- `vulnerability_update_gate_max_high` - Maximum high vulnerabilities an update may have (`-1` for no limit).
- `vulnerability_update_gate_max_medium` - Maximum medium vulnerabilities an update may have (`-1` for no limit).
- `vulnerability_update_gate_max_low` - Maximum low vulnerabilities an update may have (`-1` for no limit).
- `trivy_config` - Trivy configuration file content.
- `trivy_ignore` - Trivy ignore file content (`.trivyignore`).
- `registry_tag_timeout` - Registry tag listing timeout, in seconds.
- `http2_receive_window_mb` - HTTP/2 receive window, in MB.
- `agent_tunnel_stream_window_mb` - Agent tunnel stream window, in MB.
- `development_branding_enabled` - Show the development branding (global app-wide setting).
- `all_environments_limit` - Maximum environments shown in the all-environments views (global app-wide setting).
- `upgrade_max_parallel_agents` - Maximum agents upgraded in parallel (global app-wide setting).
- `analytics_enabled` - Send anonymous analytics (global app-wide setting).
- `update_check_enabled` - Check for Arcane updates (global app-wide setting).
- `notifications_include_release_links` - Include release links in update notifications (global app-wide setting).
- `hsts_enabled` - Send the HSTS header (global authentication setting).
- `oidc_authorization_endpoint` - OIDC authorization endpoint override (global authentication setting).
- `oidc_token_endpoint` - OIDC token endpoint override (global authentication setting).
- `oidc_userinfo_endpoint` - OIDC userinfo endpoint override (global authentication setting).
- `oidc_jwks_endpoint` - OIDC JWKS endpoint override (global authentication setting).
- `oidc_device_authorization_endpoint` - OIDC device authorization endpoint override (global authentication setting).
- `oidc_mobile_redirect_uris` - Redirect URIs allowed for the mobile app (global authentication setting).
- `github_token_wo` - GitHub token used for GitHub API requests (rate limits). Sensitive, write-only: never stored in state, only sent when `github_token_wo_version` changes. See [write-only arguments](../index.md#write-only-arguments).
- `github_token_wo_version` (Number) - Change it to send a new `github_token_wo`. Required with `github_token_wo`.

## Drift detection

On refresh, every setting set in the configuration is read back from Arcane, so a change made outside Terraform shows up in the next plan and is reverted on apply. Settings not set in the configuration are not managed and never show a diff. Settings missing from Arcane's response keep their last applied value. The write-only secrets (`depot_token_wo`, `oidc_client_secret_wo`, `trivy_server_token_wo`, `github_token_wo`) are never read back.

## Attributes Reference

- `id` (String) - Same as `environment_id`.
- `applied` (Map of String) - Server values after apply: the environment settings plus the global app-wide and authentication settings.

## Import

Import using the environment ID:

```
terraform import arcane_settings.env <environment_id>
```
