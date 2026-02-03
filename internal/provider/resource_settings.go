package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SettingsResource{}
var _ resource.ResourceWithImportState = &SettingsResource{}

type SettingsResource struct {
	client *sdkclient.Client
}

func NewSettingsResource() resource.Resource { return &SettingsResource{} }

func (r *SettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (r *SettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Resource ID (same as environment_id).",
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID.",
			},
			// SettingsUpdate attributes (all strings per OpenAPI schema)
			"accent_color":                  resourceschema.StringAttribute{Optional: true, Description: "accentColor"},
			"auth_local_enabled":            resourceschema.StringAttribute{Optional: true, Description: "authLocalEnabled"},
			"auth_oidc_config":              resourceschema.StringAttribute{Optional: true, Description: "authOidcConfig"},
			"auth_password_policy":          resourceschema.StringAttribute{Optional: true, Description: "authPasswordPolicy"},
			"auth_session_timeout":          resourceschema.StringAttribute{Optional: true, Description: "authSessionTimeout"},
			"auto_inject_env":               resourceschema.StringAttribute{Optional: true, Description: "autoInjectEnv"},
			"auto_update":                   resourceschema.StringAttribute{Optional: true, Description: "autoUpdate"},
			"auto_update_interval":          resourceschema.StringAttribute{Optional: true, Description: "autoUpdateInterval"},
			"base_server_url":               resourceschema.StringAttribute{Optional: true, Description: "baseServerUrl"},
			"default_shell":                 resourceschema.StringAttribute{Optional: true, Description: "defaultShell"},
			"disk_usage_path":               resourceschema.StringAttribute{Optional: true, Description: "diskUsagePath"},
			"docker_host":                   resourceschema.StringAttribute{Optional: true, Description: "dockerHost"},
			"docker_prune_mode":                resourceschema.StringAttribute{Optional: true, Description: "dockerPruneMode"},
			"docker_api_timeout":               resourceschema.StringAttribute{Optional: true, Description: "dockerApiTimeout"},
			"docker_image_pull_timeout":        resourceschema.StringAttribute{Optional: true, Description: "dockerImagePullTimeout"},
			"enable_gravatar":                  resourceschema.StringAttribute{Optional: true, Description: "enableGravatar"},
			"environment_health_interval":      resourceschema.StringAttribute{Optional: true, Description: "environmentHealthInterval"},
			"git_operation_timeout":            resourceschema.StringAttribute{Optional: true, Description: "gitOperationTimeout"},
			"http_client_timeout":              resourceschema.StringAttribute{Optional: true, Description: "httpClientTimeout"},
			"keyboard_shortcuts_enabled":       resourceschema.StringAttribute{Optional: true, Description: "keyboardShortcutsEnabled"},
			"max_image_upload_size":            resourceschema.StringAttribute{Optional: true, Description: "maxImageUploadSize"},
			"mobile_navigation_mode":        resourceschema.StringAttribute{Optional: true, Description: "mobileNavigationMode"},
			"mobile_navigation_show_labels": resourceschema.StringAttribute{Optional: true, Description: "mobileNavigationShowLabels"},
			"oidc_admin_claim":                 resourceschema.StringAttribute{Optional: true, Description: "oidcAdminClaim"},
			"oidc_admin_value":                 resourceschema.StringAttribute{Optional: true, Description: "oidcAdminValue"},
			"oidc_auto_redirect_to_provider":   resourceschema.StringAttribute{Optional: true, Description: "oidcAutoRedirectToProvider"},
			"oidc_client_id":                   resourceschema.StringAttribute{Optional: true, Description: "oidcClientId"},
			"oidc_client_secret":            resourceschema.StringAttribute{Optional: true, Description: "oidcClientSecret"},
			"oidc_enabled":                  resourceschema.StringAttribute{Optional: true, Description: "oidcEnabled"},
			"oidc_issuer_url":               resourceschema.StringAttribute{Optional: true, Description: "oidcIssuerUrl"},
			"oidc_merge_accounts":           resourceschema.StringAttribute{Optional: true, Description: "oidcMergeAccounts"},
			"oidc_provider_logo_url":        resourceschema.StringAttribute{Optional: true, Description: "oidcProviderLogoUrl"},
			"oidc_provider_name":            resourceschema.StringAttribute{Optional: true, Description: "oidcProviderName"},
			"oidc_scopes":                   resourceschema.StringAttribute{Optional: true, Description: "oidcScopes"},
			"oidc_skip_tls_verify":          resourceschema.StringAttribute{Optional: true, Description: "oidcSkipTlsVerify"},
			"polling_enabled":               resourceschema.StringAttribute{Optional: true, Description: "pollingEnabled"},
			"polling_interval":              resourceschema.StringAttribute{Optional: true, Description: "pollingInterval"},
			"projects_directory":            resourceschema.StringAttribute{Optional: true, Description: "projectsDirectory"},
			"proxy_request_timeout":         resourceschema.StringAttribute{Optional: true, Description: "proxyRequestTimeout"},
			"registry_timeout":              resourceschema.StringAttribute{Optional: true, Description: "registryTimeout"},
			"scheduled_prune_build_cache":   resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneBuildCache"},
			"scheduled_prune_containers":    resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneContainers"},
			"scheduled_prune_enabled":       resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneEnabled"},
			"scheduled_prune_images":        resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneImages"},
			"scheduled_prune_interval":      resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneInterval"},
			"scheduled_prune_networks":      resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneNetworks"},
			"scheduled_prune_volumes":       resourceschema.StringAttribute{Optional: true, Description: "scheduledPruneVolumes"},
			"sidebar_hover_expansion":       resourceschema.StringAttribute{Optional: true, Description: "sidebarHoverExpansion"},

			// Computed applied map
			"applied": resourceschema.MapAttribute{
				Computed:    true,
				Description: "All environment settings after apply (key -> value).",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *SettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type settingsModel struct {
	ID                         types.String `tfsdk:"id"`
	EnvironmentID              types.String `tfsdk:"environment_id"`
	AccentColor                types.String `tfsdk:"accent_color"`
	AuthLocalEnabled           types.String `tfsdk:"auth_local_enabled"`
	AuthOidcConfig             types.String `tfsdk:"auth_oidc_config"`
	AuthPasswordPolicy         types.String `tfsdk:"auth_password_policy"`
	AuthSessionTimeout         types.String `tfsdk:"auth_session_timeout"`
	AutoInjectEnv              types.String `tfsdk:"auto_inject_env"`
	AutoUpdate                 types.String `tfsdk:"auto_update"`
	AutoUpdateInterval         types.String `tfsdk:"auto_update_interval"`
	BaseServerUrl              types.String `tfsdk:"base_server_url"`
	DefaultShell               types.String `tfsdk:"default_shell"`
	DiskUsagePath              types.String `tfsdk:"disk_usage_path"`
	DockerApiTimeout           types.String `tfsdk:"docker_api_timeout"`
	DockerHost                 types.String `tfsdk:"docker_host"`
	DockerImagePullTimeout     types.String `tfsdk:"docker_image_pull_timeout"`
	DockerPruneMode            types.String `tfsdk:"docker_prune_mode"`
	EnableGravatar             types.String `tfsdk:"enable_gravatar"`
	EnvironmentHealthInterval  types.String `tfsdk:"environment_health_interval"`
	GitOperationTimeout        types.String `tfsdk:"git_operation_timeout"`
	HttpClientTimeout          types.String `tfsdk:"http_client_timeout"`
	KeyboardShortcutsEnabled   types.String `tfsdk:"keyboard_shortcuts_enabled"`
	MaxImageUploadSize         types.String `tfsdk:"max_image_upload_size"`
	MobileNavigationMode       types.String `tfsdk:"mobile_navigation_mode"`
	MobileNavigationShowLabels types.String `tfsdk:"mobile_navigation_show_labels"`
	OidcAdminClaim             types.String `tfsdk:"oidc_admin_claim"`
	OidcAdminValue             types.String `tfsdk:"oidc_admin_value"`
	OidcAutoRedirectToProvider types.String `tfsdk:"oidc_auto_redirect_to_provider"`
	OidcClientId               types.String `tfsdk:"oidc_client_id"`
	OidcClientSecret           types.String `tfsdk:"oidc_client_secret"`
	OidcEnabled                types.String `tfsdk:"oidc_enabled"`
	OidcIssuerUrl              types.String `tfsdk:"oidc_issuer_url"`
	OidcMergeAccounts          types.String `tfsdk:"oidc_merge_accounts"`
	OidcProviderLogoUrl        types.String `tfsdk:"oidc_provider_logo_url"`
	OidcProviderName           types.String `tfsdk:"oidc_provider_name"`
	OidcScopes                 types.String `tfsdk:"oidc_scopes"`
	OidcSkipTlsVerify          types.String `tfsdk:"oidc_skip_tls_verify"`
	PollingEnabled             types.String `tfsdk:"polling_enabled"`
	PollingInterval            types.String `tfsdk:"polling_interval"`
	ProjectsDirectory          types.String `tfsdk:"projects_directory"`
	ProxyRequestTimeout        types.String `tfsdk:"proxy_request_timeout"`
	RegistryTimeout            types.String `tfsdk:"registry_timeout"`
	ScheduledPruneBuildCache   types.String `tfsdk:"scheduled_prune_build_cache"`
	ScheduledPruneContainers   types.String `tfsdk:"scheduled_prune_containers"`
	ScheduledPruneEnabled      types.String `tfsdk:"scheduled_prune_enabled"`
	ScheduledPruneImages       types.String `tfsdk:"scheduled_prune_images"`
	ScheduledPruneInterval     types.String `tfsdk:"scheduled_prune_interval"`
	ScheduledPruneNetworks     types.String `tfsdk:"scheduled_prune_networks"`
	ScheduledPruneVolumes      types.String `tfsdk:"scheduled_prune_volumes"`
	SidebarHoverExpansion      types.String `tfsdk:"sidebar_hover_expansion"`
	Applied                    types.Map    `tfsdk:"applied"`
}

func (r *SettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := plan.EnvironmentID.ValueString()
	vals := buildSettingsMapFromModel(plan)
	if len(vals) > 0 {
		if _, err := r.client.UpdateSettings(ctx, envID, vals); err != nil {
			resp.Diagnostics.AddError("update settings failed", err.Error())
			return
		}
	}

	applied, err := r.client.GetSettings(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read settings failed", err.Error())
		return
	}
	state := plan
	state.ID = types.StringValue(envID)
	state.EnvironmentID = types.StringValue(envID)
	state.Applied = stringMapToMap(ctx, applied)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state settingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()

	applied, err := r.client.GetSettings(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read settings failed", err.Error())
		return
	}
	state.ID = types.StringValue(envID)
	state.Applied = stringMapToMap(ctx, applied)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan settingsModel
	var state settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	vals := buildSettingsMapFromModel(plan)
	if len(vals) > 0 {
		if _, err := r.client.UpdateSettings(ctx, envID, vals); err != nil {
			resp.Diagnostics.AddError("update settings failed", err.Error())
			return
		}
	}
	applied, err := r.client.GetSettings(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read settings failed", err.Error())
		return
	}
	// Update state with plan values and applied settings from API
	state = plan
	state.ID = types.StringValue(envID)
	state.EnvironmentID = types.StringValue(envID)
	state.Applied = stringMapToMap(ctx, applied)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Not reverting settings on delete; just remove from state.
}

func (r *SettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by environment ID
	resource.ImportStatePassthroughID(ctx, path.Root("environment_id"), req, resp)
}

// Helpers
func addIfSet(m map[string]string, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueString()
	}
}

func buildSettingsMapFromModel(s settingsModel) map[string]string {
	out := map[string]string{}
	addIfSet(out, "accentColor", s.AccentColor)
	addIfSet(out, "authLocalEnabled", s.AuthLocalEnabled)
	addIfSet(out, "authOidcConfig", s.AuthOidcConfig)
	addIfSet(out, "authPasswordPolicy", s.AuthPasswordPolicy)
	addIfSet(out, "authSessionTimeout", s.AuthSessionTimeout)
	addIfSet(out, "autoInjectEnv", s.AutoInjectEnv)
	addIfSet(out, "autoUpdate", s.AutoUpdate)
	addIfSet(out, "autoUpdateInterval", s.AutoUpdateInterval)
	addIfSet(out, "baseServerUrl", s.BaseServerUrl)
	addIfSet(out, "defaultShell", s.DefaultShell)
	addIfSet(out, "diskUsagePath", s.DiskUsagePath)
	addIfSet(out, "dockerApiTimeout", s.DockerApiTimeout)
	addIfSet(out, "dockerHost", s.DockerHost)
	addIfSet(out, "dockerImagePullTimeout", s.DockerImagePullTimeout)
	addIfSet(out, "dockerPruneMode", s.DockerPruneMode)
	addIfSet(out, "enableGravatar", s.EnableGravatar)
	addIfSet(out, "environmentHealthInterval", s.EnvironmentHealthInterval)
	addIfSet(out, "gitOperationTimeout", s.GitOperationTimeout)
	addIfSet(out, "httpClientTimeout", s.HttpClientTimeout)
	addIfSet(out, "keyboardShortcutsEnabled", s.KeyboardShortcutsEnabled)
	addIfSet(out, "maxImageUploadSize", s.MaxImageUploadSize)
	addIfSet(out, "mobileNavigationMode", s.MobileNavigationMode)
	addIfSet(out, "mobileNavigationShowLabels", s.MobileNavigationShowLabels)
	addIfSet(out, "oidcAdminClaim", s.OidcAdminClaim)
	addIfSet(out, "oidcAdminValue", s.OidcAdminValue)
	addIfSet(out, "oidcAutoRedirectToProvider", s.OidcAutoRedirectToProvider)
	addIfSet(out, "oidcClientId", s.OidcClientId)
	addIfSet(out, "oidcClientSecret", s.OidcClientSecret)
	addIfSet(out, "oidcEnabled", s.OidcEnabled)
	addIfSet(out, "oidcIssuerUrl", s.OidcIssuerUrl)
	addIfSet(out, "oidcMergeAccounts", s.OidcMergeAccounts)
	addIfSet(out, "oidcProviderLogoUrl", s.OidcProviderLogoUrl)
	addIfSet(out, "oidcProviderName", s.OidcProviderName)
	addIfSet(out, "oidcScopes", s.OidcScopes)
	addIfSet(out, "oidcSkipTlsVerify", s.OidcSkipTlsVerify)
	addIfSet(out, "pollingEnabled", s.PollingEnabled)
	addIfSet(out, "pollingInterval", s.PollingInterval)
	addIfSet(out, "projectsDirectory", s.ProjectsDirectory)
	addIfSet(out, "proxyRequestTimeout", s.ProxyRequestTimeout)
	addIfSet(out, "registryTimeout", s.RegistryTimeout)
	addIfSet(out, "scheduledPruneBuildCache", s.ScheduledPruneBuildCache)
	addIfSet(out, "scheduledPruneContainers", s.ScheduledPruneContainers)
	addIfSet(out, "scheduledPruneEnabled", s.ScheduledPruneEnabled)
	addIfSet(out, "scheduledPruneImages", s.ScheduledPruneImages)
	addIfSet(out, "scheduledPruneInterval", s.ScheduledPruneInterval)
	addIfSet(out, "scheduledPruneNetworks", s.ScheduledPruneNetworks)
	addIfSet(out, "scheduledPruneVolumes", s.ScheduledPruneVolumes)
	addIfSet(out, "sidebarHoverExpansion", s.SidebarHoverExpansion)
	return out
}

func stringMapToMap(ctx context.Context, m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	mv, _ := types.MapValue(types.StringType, elems)
	return mv
}
